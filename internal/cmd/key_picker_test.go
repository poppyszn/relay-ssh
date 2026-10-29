package cmd

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/poppyszn/relay-ssh/internal/core"
)

func TestFindKeyFilesDetectsPrivateKeys(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "id_ed25519"), "-----BEGIN OPENSSH PRIVATE KEY-----\nabc\n")
	writeFile(t, filepath.Join(dir, "id_rsa"), "-----BEGIN RSA PRIVATE KEY-----\nabc\n")
	writeFile(t, filepath.Join(dir, "id_ed25519.pub"), "ssh-ed25519 AAAA...")
	writeFile(t, filepath.Join(dir, "known_hosts"), "github.com ssh-rsa AAAA...")
	writeFile(t, filepath.Join(dir, "config"), "Host *\n")
	writeFile(t, filepath.Join(dir, "authorized_keys"), "ssh-ed25519 AAAA...")
	writeFile(t, filepath.Join(dir, "notes.txt"), "just some notes")

	got := core.FindKeyFiles(dir)
	if len(got) != 2 {
		t.Fatalf("findKeyFiles = %v, want 2 keys", got)
	}
	base0, base1 := filepath.Base(got[0]), filepath.Base(got[1])
	if base0 != "id_ed25519" || base1 != "id_rsa" {
		t.Errorf("findKeyFiles = [%s %s], want [id_ed25519 id_rsa] (sorted)", base0, base1)
	}
}

func TestFindKeyFilesIgnoresDirectories(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "id_ed25519"), "-----BEGIN OPENSSH PRIVATE KEY-----\n")
	got := core.FindKeyFiles(dir)
	if len(got) != 1 {
		t.Errorf("findKeyFiles should skip directories, got %v", got)
	}
}

func TestFindKeyFilesOnMissingDir(t *testing.T) {
	got := core.FindKeyFiles(filepath.Join(t.TempDir(), "nonexistent"))
	if got != nil {
		t.Errorf("findKeyFiles on missing dir = %v, want nil", got)
	}
}

func TestPickKeyFileFallsBackToPromptWhenNoneFound(t *testing.T) {
	old := os.Getenv("HOME")
	os.Setenv("HOME", t.TempDir()) // empty ~/.ssh
	defer os.Setenv("HOME", old)

	in := bufio.NewScanner(strings.NewReader("/manual/path/to/key\n"))
	got, err := PickKeyFile(in)
	if err != nil {
		t.Fatalf("PickKeyFile: %v", err)
	}
	if got != "/manual/path/to/key" {
		t.Errorf("PickKeyFile fallback = %q, want manual path", got)
	}
}

func TestPickKeyFileSelectsFromMenu(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	home := t.TempDir()
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(sshDir, "id_ed25519"), "-----BEGIN OPENSSH PRIVATE KEY-----\n")
	writeFile(t, filepath.Join(sshDir, "id_rsa"), "-----BEGIN RSA PRIVATE KEY-----\n")

	old := os.Getenv("HOME")
	os.Setenv("HOME", home)
	defer os.Setenv("HOME", old)

	// With fzf forced off, this exercises the numbered-menu fallback:
	// select option 1 (id_ed25519, first alphabetically).
	in := bufio.NewScanner(strings.NewReader("1\n"))
	got, err := PickKeyFile(in)
	if err != nil {
		t.Fatalf("PickKeyFile: %v", err)
	}
	if filepath.Base(got) != "id_ed25519" {
		t.Errorf("PickKeyFile = %q, want id_ed25519", got)
	}
}

func TestPickKeyFileManualEntryOption(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	home := t.TempDir()
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(sshDir, "id_ed25519"), "-----BEGIN OPENSSH PRIVATE KEY-----\n")

	old := os.Getenv("HOME")
	os.Setenv("HOME", home)
	defer os.Setenv("HOME", old)

	// Option 2 is "Type a path manually...", which should prompt again.
	in := bufio.NewScanner(strings.NewReader("2\n/typed/path\n"))
	got, err := PickKeyFile(in)
	if err != nil {
		t.Fatalf("PickKeyFile: %v", err)
	}
	if got != "/typed/path" {
		t.Errorf("PickKeyFile manual entry = %q, want /typed/path", got)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
