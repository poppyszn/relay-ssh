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

func TestSaveProfileAddsUpdatesAndRenames(t *testing.T) {
	p := newStore(t, agentProfile("bastion", ""), agentProfile("db", ""))

	// add
	web := agentProfile("web", "prod")
	web.Jump = "bastion"
	if err := SaveProfile(p, "", web); err != nil {
		t.Fatal(err)
	}
	if err := SaveProfile(p, "", web); err == nil {
		t.Error("adding a duplicate name should fail")
	}

	// update in place
	web.Host = "new-host"
	if err := SaveProfile(p, "web", web); err != nil {
		t.Fatal(err)
	}

	// rename a jump target: dependents follow it
	renamed := agentProfile("gateway", "")
	if err := SaveProfile(p, "bastion", renamed); err != nil {
		t.Fatal(err)
	}
	s, _ := config.Load(p.StoreFile)
	got, _ := s.Find("web")
	if got.Host != "new-host" || got.Jump != "gateway" {
		t.Errorf("web = %+v, want host new-host and jump gateway", got)
	}
	if _, ok := s.Find("bastion"); ok {
		t.Error("old name should be gone after rename")
	}

	// renaming onto another profile's name, or an invalid profile, must not touch the store
	if err := SaveProfile(p, "db", agentProfile("gateway", "")); err == nil {
		t.Error("rename onto an existing name should fail")
	}
	bad := agentProfile("db", "")
	bad.Port = 0
	if err := SaveProfile(p, "db", bad); err == nil {
		t.Error("invalid profile should fail")
	}
	if err := SaveProfile(p, "ghost", agentProfile("x", "")); err == nil {
		t.Error("updating an unknown profile should fail")
	}
	gen, _ := os.ReadFile(p.GenConfig)
	if !strings.Contains(string(gen), "Host gateway") {
		t.Errorf("generated config not regenerated:\n%s", gen)
	}
}

func TestImportHandlesAdjacentHostsDefaultsAndSkipsBadEntries(t *testing.T) {
	p := newStore(t)
	in := "Host first\n  HostName 1.1.1.1\n  User a\nHost second\n  HostName 2.2.2.2\n  User b\n" + // no blank line between blocks
		"Host bare\n" + // no HostName or User: ssh falls back to the alias and the current user
		"Host bad:name\n  HostName 3.3.3.3\n  User c\n" + // colon is not allowed in a profile name
		"Host *.internal\n  User wild\n" +
		"Host last\n  HostName 4.4.4.4\n  User d\n"
	res, err := ImportSSHConfigDetailed(p, strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(res.Added, ","); got != "first,second,bare,last" {
		t.Errorf("added = %q, want first,second,bare,last", got)
	}
	if len(res.Skipped) != 1 || !strings.Contains(res.Skipped[0], "bad:name") {
		t.Errorf("skipped = %v, want only bad:name", res.Skipped)
	}
	s, _ := config.Load(p.StoreFile)
	bare, ok := s.Find("bare")
	if !ok || bare.Host != "bare" || bare.User == "" || bare.Port != 22 {
		t.Errorf("bare = %+v, want host=alias, a user, port 22", bare)
	}
	// the legacy entry point still reports a count
	n, err := ImportSSHConfig(newStore(t), strings.NewReader("Host x\n  HostName 9.9.9.9\n  User u\n"))
	if err != nil || n != 1 {
		t.Errorf("ImportSSHConfig = %d, %v", n, err)
	}
}
