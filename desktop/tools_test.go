package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func newHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("SSSH_HOME", home)
	return home
}

func TestHealthAndFix(t *testing.T) {
	newHome(t)
	svc := ProfilesService{}
	if err := svc.Save("", Profile{Name: "web", Host: "h", User: "u", Port: 22, Auth: "agent"}); err != nil {
		t.Fatal(err)
	}
	tools := ToolsService{}
	if err := os.WriteFile(filepath.Join(os.Getenv("SSSH_HOME"), ".ssh", "sssh.d", "config"), []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, _ := tools.Health()
	stale := false
	for _, c := range before {
		stale = stale || (c.Name == "generated config" && c.Status == "warn")
	}
	if !stale {
		t.Errorf("stale config should warn: %+v", before)
	}
	rep, err := tools.FixHealth()
	if err != nil || len(rep.Errors) != 0 {
		t.Fatalf("FixHealth = %+v, %v", rep, err)
	}
	for _, c := range rep.Checks {
		if c.Status != "ok" {
			t.Errorf("after fix %s = %s %s", c.Name, c.Status, c.Detail)
		}
	}
}

func TestBackupRestoreThroughTheService(t *testing.T) {
	home := newHome(t)
	if err := (ProfilesService{}).Save("", Profile{Name: "web", Host: "h", User: "u", Port: 22, Auth: "agent"}); err != nil {
		t.Fatal(err)
	}
	tools := ToolsService{}
	dest := filepath.Join(t.TempDir(), "b")
	if err := tools.Backup(dest, ""); err == nil {
		t.Error("an empty passphrase must be rejected")
	}
	if err := tools.Backup(dest, "pw"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dest + ".age"); err != nil {
		t.Fatalf("backup should get the .age suffix: %v", err)
	}
	if err := (ProfilesService{}).Delete("web"); err != nil {
		t.Fatal(err)
	}
	if err := tools.Restore(dest+".age", "wrong"); err == nil || !strings.Contains(err.Error(), "passphrase") {
		t.Errorf("wrong passphrase err = %v", err)
	}
	if err := tools.Restore(dest+".age", "pw"); err != nil {
		t.Fatal(err)
	}
	if got, _ := (ProfilesService{}).List(); len(got) != 1 || got[0].Name != "web" {
		t.Errorf("restored profiles = %v (home %s)", got, home)
	}
}

func TestImportReportsAddedAndSkipped(t *testing.T) {
	home := newHome(t)
	cfg := "Host a\n  HostName 1.1.1.1\n  User x\nHost b\n  HostName 2.2.2.2\n  User y\nHost bad:name\n  HostName 3.3.3.3\n  User z\n"
	if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".ssh", "config"), []byte(cfg), 0o600); err != nil {
		t.Fatal(err)
	}
	rep, err := (ToolsService{}).ImportSSHConfig()
	if err != nil || len(rep.Added) != 2 || len(rep.Skipped) != 1 {
		t.Errorf("ImportSSHConfig = %+v, %v", rep, err)
	}
}

func TestKeysListAndAdopt(t *testing.T) {
	home := newHome(t)
	ssh := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(ssh, 0o700); err != nil {
		t.Fatal(err)
	}
	key := filepath.Join(ssh, "id_ed25519")
	if err := os.WriteFile(key, []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nabc\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := (ProfilesService{}).Save("", Profile{Name: "web", Host: "h", User: "u", Port: 22, Auth: "key", Key: key}); err != nil {
		t.Fatal(err)
	}
	tools := ToolsService{}
	ks, err := tools.Keys()
	if err != nil || len(ks) != 1 || ks[0].Adopted || len(ks[0].UsedBy) != 1 || ks[0].UsedBy[0] != "web" {
		t.Fatalf("before adopt: %+v, %v", ks, err)
	}
	if err := tools.AdoptKey(key, ""); err != nil {
		t.Fatal(err)
	}
	ks, _ = tools.Keys()
	if len(ks) != 1 || !ks[0].Adopted || len(ks[0].UsedBy) != 1 {
		t.Errorf("after adopt: %+v", ks)
	}
	if got, _ := (ProfilesService{}).List(); got[0].Key != "id_ed25519" {
		t.Errorf("profile should now use the adopted key by name, got %q", got[0].Key)
	}
}

// Adopting a key that has a passphrase needs the passphrase: it is checked before anything is copied,
// so a wrong one leaves no half-adopted key behind.
func TestAdoptAskForTheKeysPassphrase(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SSSH_HOME", home)
	t.Setenv("SSH_AUTH_SOCK", startAgent(t))
	keychainEnabled = false // never touch the real Keychain from a test
	t.Cleanup(func() { keychainEnabled = runtime.GOOS == "darwin" })
	ssh := filepath.Join(home, ".ssh")
	_ = os.MkdirAll(ssh, 0o700)
	locked := makeKey(t, ssh, "id_locked", "hunter2")
	plain := makeKey(t, ssh, "id_plain", "")
	tools := ToolsService{}

	if !tools.KeyNeedsPassphrase(locked) || tools.KeyNeedsPassphrase(plain) {
		t.Fatal("KeyNeedsPassphrase gave the wrong answer")
	}
	ks, _ := tools.Keys()
	for _, k := range ks {
		if k.Encrypted != (k.Name == "id_locked") {
			t.Errorf("%s: Encrypted = %v", k.Name, k.Encrypted)
		}
	}
	if err := tools.AdoptKey(locked, ""); err == nil {
		t.Error("an encrypted key was adopted without its passphrase")
	}
	if err := tools.AdoptKey(locked, "wrong"); err == nil {
		t.Error("an encrypted key was adopted with the wrong passphrase")
	}
	vault := filepath.Join(home, ".ssh", "sssh.d", "keys", "id_locked")
	if _, err := os.Stat(vault); err == nil {
		t.Error("a failed adoption left a copy of the key in the vault")
	}
	if err := tools.AdoptKey(locked, "hunter2"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(vault); err != nil {
		t.Errorf("the adopted key is not in the vault: %v", err)
	}
	if got := agentKeys(t, os.Getenv("SSH_AUTH_SOCK")); !strings.Contains(got, "ED25519") {
		t.Errorf("the adopted key was not loaded into the agent:\n%s", got)
	}
	if err := tools.AdoptKey(plain, ""); err != nil {
		t.Errorf("a key without a passphrase needs none: %v", err)
	}
}
