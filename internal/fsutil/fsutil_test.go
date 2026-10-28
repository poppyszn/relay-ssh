package fsutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileCreatesWithMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.toml")
	if err := WriteFile(path, []byte("hello"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("content = %q, want %q", data, "hello")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode = %o, want 600", perm)
	}
}

func TestWriteFileOverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.toml")
	if err := os.WriteFile(path, []byte("old content that is long"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(path, []byte("new"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "new" {
		t.Errorf("content = %q, want %q", data, "new")
	}
}

// The temp file must be created in the target's own directory, not in /tmp,
// so that the final rename is atomic rather than a cross-device copy.
func TestWriteFileLeavesNoTempFileBehind(t *testing.T) {
	dir := t.TempDir()
	if err := WriteFile(filepath.Join(dir, "out.toml"), []byte("x"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("temp file left behind: %s", e.Name())
		}
	}
	if len(entries) != 1 {
		t.Errorf("directory has %d entries, want 1", len(entries))
	}
}

func TestWriteFileFailsOnMissingDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nope", "out.toml")
	if err := WriteFile(path, []byte("x"), 0o600); err == nil {
		t.Fatal("expected an error writing into a missing directory, got nil")
	}
}
