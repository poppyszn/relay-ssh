package core

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/poppyszn/super-ssh-v2/internal/config"
	"github.com/poppyszn/super-ssh-v2/internal/paths"
)

func newStore(t *testing.T, profiles ...config.Profile) paths.Paths {
	t.Helper()
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: profiles}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}
	return p
}

func agentProfile(name, folder string) config.Profile {
	return config.Profile{Name: name, Folder: folder, Host: "h", User: "u", Port: 22, Auth: config.AuthAgent}
}

func TestResolveProfileAcceptsFolderQualifiedName(t *testing.T) {
	s := config.Store{Profiles: []config.Profile{agentProfile("web", "prod")}}
	if _, ok := ResolveProfile(s, "prod/web"); !ok {
		t.Error("folder/name should resolve")
	}
	if _, ok := ResolveProfile(s, "other/web"); ok {
		t.Error("wrong folder must not resolve")
	}
}

func TestRemoveProfileRegeneratesConfig(t *testing.T) {
	p := newStore(t, agentProfile("a", ""), agentProfile("b", ""))
	removed, err := RemoveProfile(p, "a")
	if err != nil || removed.Name != "a" {
		t.Fatalf("RemoveProfile = %v, %v", removed, err)
	}
	gen, err := os.ReadFile(p.GenConfig)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(gen), "Host a\n") || !strings.Contains(string(gen), "Host b") {
		t.Errorf("generated config not updated:\n%s", gen)
	}
	if _, err := RemoveProfile(p, "nope"); err == nil {
		t.Error("unknown profile should error")
	}
}

func TestImportSSHConfigSkipsWildcardsAndDuplicates(t *testing.T) {
	p := newStore(t, agentProfile("old", ""))
	in := "Host *\n  User x\n\nHost fresh\n  HostName 1.2.3.4\n  User bob\n  Port 2222\n\nHost old\n  HostName z\n"
	n, err := ImportSSHConfig(p, strings.NewReader(in))
	if err != nil || n != 1 {
		t.Fatalf("ImportSSHConfig = %d, %v", n, err)
	}
	s, _ := config.Load(p.StoreFile)
	got, ok := s.Find("fresh")
	if !ok || got.Host != "1.2.3.4" || got.User != "bob" || got.Port != 2222 {
		t.Errorf("imported profile = %+v, %v", got, ok)
	}
}

func TestBackupRestoreRoundTripAndWrongPassphrase(t *testing.T) {
	src := newStore(t, agentProfile("web", ""))
	var buf bytes.Buffer
	if err := Backup(src, &buf, "pw"); err != nil {
		t.Fatal(err)
	}
	dst := paths.New(t.TempDir())
	if err := Restore(dst, bytes.NewReader(buf.Bytes()), "wrong", true); !errors.Is(err, ErrIncorrectPassphrase) {
		t.Errorf("wrong passphrase err = %v", err)
	}
	if err := paths.EnsureDirs(dst); err != nil {
		t.Fatal(err)
	}
	if err := Restore(dst, bytes.NewReader(buf.Bytes()), "pw", true); err != nil {
		t.Fatal(err)
	}
	s, err := config.Load(dst.StoreFile)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := s.Find("web"); !ok {
		t.Error("restored store is missing profile")
	}
}

func TestDiagnoseReportsStaleConfigAsWarn(t *testing.T) {
	p := newStore(t, agentProfile("web", ""))
	if errs := Fix(p); len(errs) != 0 {
		t.Fatal(errs)
	}
	for _, c := range Diagnose(p) {
		if c.Status != OK {
			t.Errorf("after Fix, %s = %v %s", c.Name, c.Status, c.Detail)
		}
	}
	if err := os.WriteFile(p.GenConfig, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	var warned bool
	for _, c := range Diagnose(p) {
		if c.Name == "generated config" && c.Status == Warn {
			warned = true
		}
	}
	if !warned {
		t.Error("stale generated config should warn")
	}
}
