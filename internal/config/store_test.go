package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissingFileYieldsEmptyStore(t *testing.T) {
	s, err := Load(filepath.Join(t.TempDir(), "connections.toml"))
	if err != nil {
		t.Fatalf("Load: %v, want nil (a missing file is a first run, not an error)", err)
	}
	if s.Version != Version {
		t.Errorf("Version = %d, want %d", s.Version, Version)
	}
	if len(s.Profiles) != 0 {
		t.Errorf("Profiles = %d, want 0", len(s.Profiles))
	}
}

func TestSaveThenLoadRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.toml")
	want := Store{Version: Version, Profiles: []Profile{
		{
			Name: "prod-web", Host: "10.0.0.5", User: "deploy", Port: 2222,
			Auth: AuthKey, Key: "prod-web", Folder: "work/prod",
			Tags: []string{"web", "prod"}, Jump: "bastion", ForwardAgent: true, Blocks: "on",
			Tunnels: []Tunnel{{TunnelLocal, "5432", "db.internal:5432"}},
		},
		{Name: "bastion", Host: "b.example.com", User: "admin", Port: 22, Auth: AuthAgent},
	}}
	if err := Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(got.Profiles) != 2 {
		t.Fatalf("loaded %d profiles, want 2", len(got.Profiles))
	}
	p, ok := got.Find("prod-web")
	if !ok {
		t.Fatal("prod-web not found after round trip")
	}
	if p.Host != "10.0.0.5" || p.User != "deploy" || p.Port != 2222 {
		t.Errorf("host/user/port = %s/%s/%d", p.Host, p.User, p.Port)
	}
	if p.Auth != AuthKey || p.Key != "prod-web" {
		t.Errorf("auth/key = %s/%s", p.Auth, p.Key)
	}
	if p.Folder != "work/prod" || p.Jump != "bastion" || !p.ForwardAgent {
		t.Errorf("folder/jump/forwardagent = %s/%s/%v", p.Folder, p.Jump, p.ForwardAgent)
	}
	if p.Blocks != "on" {
		t.Errorf("blocks = %q, want on", p.Blocks)
	}
	if len(p.Tags) != 2 || p.Tags[0] != "web" {
		t.Errorf("tags = %v", p.Tags)
	}
	if len(p.Tunnels) != 1 || p.Tunnels[0].Type != TunnelLocal || p.Tunnels[0].Target != "db.internal:5432" {
		t.Errorf("tunnels = %+v", p.Tunnels)
	}
}

func TestSaveSortsProfilesByName(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.toml")
	s := Store{Version: Version, Profiles: []Profile{
		{Name: "zulu", Host: "z", User: "u", Port: 22, Auth: AuthAgent},
		{Name: "alpha", Host: "a", User: "u", Port: 22, Auth: AuthAgent},
	}}
	if err := Save(path, s); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if names := got.Names(); names[0] != "alpha" || names[1] != "zulu" {
		t.Errorf("Names() = %v, want [alpha zulu]", names)
	}
}

func TestSaveWritesAt0600(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.toml")
	if err := Save(path, Store{Version: Version}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode = %o, want 600", perm)
	}
}

func TestSaveRejectsAnInvalidStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.toml")
	s := Store{Version: Version, Profiles: []Profile{{Name: "no-host", User: "u", Port: 22, Auth: AuthAgent}}}
	if err := Save(path, s); err == nil {
		t.Fatal("expected Save to reject a profile with no host, got nil")
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("Save wrote a file despite rejecting the store")
	}
}

func TestLoadRejectsMalformedTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.toml")
	if err := os.WriteFile(path, []byte("this is not = = toml"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected an error for malformed TOML, got nil")
	}
	if !strings.Contains(err.Error(), path) {
		t.Errorf("error %q does not name the file", err)
	}
}

func TestLoadRejectsAnInvalidStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.toml")
	body := "version = 1\n\n[[profile]]\n  name = \"a\"\n  host = \"h\"\n  user = \"u\"\n  port = 22\n  auth = \"key\"\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected an error for key auth with no key, got nil")
	}
}

func TestStoreKeepsExplicitFoldersSortedAndDeduplicated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "connections.toml")
	s := Store{Version: Version, Folders: []string{"b", "a/x", "a", "a"}, Profiles: []Profile{{Name: "p", Host: "h", User: "u", Port: 22, Auth: AuthAgent}}}
	if err := Save(path, s); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got.Folders, ",") != "a,a/x,b" {
		t.Errorf("folders = %v, want sorted and deduplicated", got.Folders)
	}
	// an older file with no folders key still loads
	old := filepath.Join(t.TempDir(), "old.toml")
	if err := os.WriteFile(old, []byte("version = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(old); err != nil {
		t.Errorf("a store without folders must load: %v", err)
	}
	// and a bad folder is refused before it reaches disk
	if err := Save(path, Store{Version: Version, Folders: []string{"/bad"}}); err == nil {
		t.Error("an invalid folder path must be refused")
	}
}
