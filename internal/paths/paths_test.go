package paths

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

func TestNewDerivesEveryLocationFromRoot(t *testing.T) {
	p := New("/home/u")
	cases := map[string]string{
		"ConfigDir":  "/home/u/.config/relay-ssh",
		"StoreFile":  "/home/u/.config/relay-ssh/connections.toml",
		"SSHDir":     "/home/u/.ssh",
		"SSHConfig":  "/home/u/.ssh/config",
		"Sandbox":    "/home/u/.ssh/relay-ssh.d",
		"GenConfig":  "/home/u/.ssh/relay-ssh.d/config",
		"KeysDir":    "/home/u/.ssh/relay-ssh.d/keys",
		"KnownHosts": "/home/u/.ssh/known_hosts",
	}
	got := map[string]string{
		"ConfigDir": p.ConfigDir, "StoreFile": p.StoreFile, "SSHDir": p.SSHDir,
		"SSHConfig": p.SSHConfig, "Sandbox": p.Sandbox, "GenConfig": p.GenConfig,
		"KeysDir": p.KeysDir, "KnownHosts": p.KnownHosts,
	}
	for field, want := range cases {
		if got[field] != want {
			t.Errorf("%s = %q, want %q", field, got[field], want)
		}
	}
}

func TestEnsureDirsCreatesEverythingAt0700(t *testing.T) {
	p := New(t.TempDir())
	if err := EnsureDirs(p); err != nil {
		t.Fatalf("EnsureDirs: %v", err)
	}
	for _, dir := range []string{p.ConfigDir, p.SSHDir, p.Sandbox, p.KeysDir} {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatalf("stat %s: %v", dir, err)
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", dir)
		}
		if perm := info.Mode().Perm(); perm != fs.FileMode(0o700) {
			t.Errorf("%s mode = %o, want 700", dir, perm)
		}
	}
}

func TestEnsureDirsIsIdempotent(t *testing.T) {
	p := New(t.TempDir())
	if err := EnsureDirs(p); err != nil {
		t.Fatalf("first EnsureDirs: %v", err)
	}
	if err := EnsureDirs(p); err != nil {
		t.Fatalf("second EnsureDirs: %v", err)
	}
}

func TestDefaultHonoursSSSHHome(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RELAY_SSH_HOME", dir)
	p, err := Default()
	if err != nil {
		t.Fatalf("Default: %v", err)
	}
	if p.Root != dir {
		t.Errorf("Root = %q, want %q", p.Root, dir)
	}
	if p.StoreFile != filepath.Join(dir, ".config", "relay-ssh", "connections.toml") {
		t.Errorf("StoreFile = %q", p.StoreFile)
	}
}

func TestDefaultFallsBackToHomeDir(t *testing.T) {
	t.Setenv("RELAY_SSH_HOME", "")
	p, err := Default()
	if err != nil {
		t.Fatalf("Default: %v", err)
	}
	if p.Root == "" {
		t.Error("Root is empty")
	}
}

func TestDefaultPrefersTheNewHomeVariableAndStillHonoursTheOld(t *testing.T) {
	t.Setenv("RELAY_SSH_HOME", "")
	t.Setenv("SSSH_HOME", "/old/home")
	p, err := Default()
	if err != nil || p.Root != "/old/home" {
		t.Fatalf("old variable: %q %v", p.Root, err)
	}
	t.Setenv("RELAY_SSH_HOME", "/new/home")
	if p, _ := Default(); p.Root != "/new/home" {
		t.Fatalf("new variable should win, got %q", p.Root)
	}
}
