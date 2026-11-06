package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The page can call every bound method, so a compromised page could hand any local path to the ones that read or
// write files. They now accept only paths the user chose in a native dialog, or that the app itself listed.
func TestPathGrants(t *testing.T) {
	now := time.Now()
	g := &pathGrants{m: map[string]time.Time{}, now: func() time.Time { return now }}

	if err := g.require("/home/u/secret"); err == nil || !strings.Contains(err.Error(), "file dialog") {
		t.Fatalf("an unchosen path must be refused with a hint about the dialog, got %v", err)
	}
	g.allow("/home/u/docs/backup.age")
	if err := g.require("/home/u/docs/backup.age"); err != nil {
		t.Fatalf("a chosen path is allowed: %v", err)
	}
	if err := g.require("/home/u/docs/../docs/./backup.age"); err != nil {
		t.Fatalf("the same file spelled differently is still the chosen one: %v", err)
	}
	for _, other := range []string{"/home/u/docs", "/home/u/docs/backup.age/x", "/home/u/docs/other.age", "/home/u/docs/backup.age.bak", ""} {
		if g.require(other) == nil {
			t.Errorf("choosing one file must not allow %q", other)
		}
	}
	now = now.Add(grantLife + time.Second)
	if g.require("/home/u/docs/backup.age") == nil {
		t.Error("a choice must expire")
	}
	g.allow("") // a cancelled dialog returns an empty path: nothing to allow
	if g.require("") == nil {
		t.Error("the empty path must never be allowed")
	}
}

func TestGrantPickedOnlyRecordsRealChoices(t *testing.T) {
	grants = &pathGrants{m: map[string]time.Time{}, now: time.Now}
	if p, err := grantPicked("", nil); p != "" || err != nil || grants.require("") == nil {
		t.Fatal("a cancelled dialog must grant nothing")
	}
	if _, err := grantPicked("/tmp/x", os.ErrPermission); err == nil || grants.require("/tmp/x") == nil {
		t.Fatal("a failed dialog must grant nothing")
	}
	if p, err := grantPicked("/tmp/chosen.age", nil); p != "/tmp/chosen.age" || err != nil || grants.require("/tmp/chosen.age") != nil {
		t.Fatal("a chosen path must be granted and returned")
	}
}

// Each method that touches a local file refuses a path the user never chose, before doing anything.
func TestFileMethodsRefuseUnchosenPaths(t *testing.T) {
	t.Setenv("RELAY_BENCH", "")
	root := t.TempDir()
	t.Setenv("RELAY_SSH_HOME", root)
	grants = &pathGrants{m: map[string]time.Time{}, now: time.Now}
	tools := ToolsService{}

	dest := filepath.Join(root, "stolen.age")
	if err := tools.Backup(dest, "pass"); err == nil || !strings.Contains(err.Error(), "file dialog") {
		t.Errorf("Backup to an unchosen path: %v", err)
	}
	if _, err := os.Stat(dest); err == nil {
		t.Error("Backup wrote a file to a path nobody chose")
	}
	if err := tools.Restore("/etc/hosts", ""); err == nil || !strings.Contains(err.Error(), "file dialog") {
		t.Errorf("Restore from an unchosen path: %v", err)
	}
	if err := tools.AdoptKey("/etc/hosts", ""); err == nil || !strings.Contains(err.Error(), "file dialog") {
		t.Errorf("AdoptKey of an unchosen path: %v", err)
	}
	if tools.KeyNeedsPassphrase("/etc/hosts") {
		t.Error("KeyNeedsPassphrase must not read an unchosen path")
	}
	files := &FilesService{}
	if err := files.Upload("any", "/etc/hosts", "/tmp/x"); err == nil || !strings.Contains(err.Error(), "file dialog") {
		t.Errorf("Upload of an unchosen path: %v", err)
	}
	if err := files.Download("any", "/tmp/x", filepath.Join(root, "dl")); err == nil || !strings.Contains(err.Error(), "file dialog") {
		t.Errorf("Download to an unchosen path: %v", err)
	}
}

// A path the app itself listed for the user (the keys in ~/.ssh and the vault) is fine to hand back.
func TestListedKeysCanBeAdopted(t *testing.T) {
	t.Setenv("RELAY_BENCH", "")
	root := t.TempDir()
	t.Setenv("RELAY_SSH_HOME", root)
	grants = &pathGrants{m: map[string]time.Time{}, now: time.Now}
	ssh := filepath.Join(root, ".ssh")
	if err := os.MkdirAll(ssh, 0o700); err != nil {
		t.Fatal(err)
	}
	key := filepath.Join(ssh, "id_listed")
	os.WriteFile(key, []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nAAAA\n-----END OPENSSH PRIVATE KEY-----\n"), 0o600)

	if err := (ToolsService{}).AdoptKey(key, ""); err == nil || !strings.Contains(err.Error(), "file dialog") {
		t.Fatalf("before it is listed, the key is not allowed: %v", err)
	}
	listed, err := (ToolsService{}).Keys()
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, k := range listed {
		found = found || k.Path == key
	}
	if !found {
		t.Fatalf("Keys() did not list %s: %+v", key, listed)
	}
	if err := (ToolsService{}).AdoptKey(key, ""); err != nil && strings.Contains(err.Error(), "file dialog") {
		t.Fatalf("a key the app listed must be allowed: %v", err)
	}
}

// The benchmark and smoke harness (RELAY_BENCH) drives these methods with paths of its own.
func TestBenchModeSkipsPathGrants(t *testing.T) {
	t.Setenv("RELAY_BENCH", "smoke")
	grants = &pathGrants{m: map[string]time.Time{}, now: time.Now}
	if err := grants.require("/anywhere"); err != nil {
		t.Fatalf("bench mode: %v", err)
	}
}
