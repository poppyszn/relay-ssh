package keys

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/poppyszn/relay-ssh/internal/paths"
)

func TestAdoptCopiesKeyAtCorrectMode(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	srcPath := filepath.Join(t.TempDir(), "id_test")
	if err := os.WriteFile(srcPath, []byte("private key data"), 0o400); err != nil {
		t.Fatal(err)
	}
	got, err := Adopt(p, srcPath, "test-key")
	if err != nil {
		t.Fatalf("Adopt: %v", err)
	}
	if got != "test-key" {
		t.Errorf("Adopt returned %q, want test-key", got)
	}
	dstPath := filepath.Join(p.KeysDir, "test-key")
	data, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("reading adopted key: %v", err)
	}
	if string(data) != "private key data" {
		t.Errorf("adopted key = %q, want 'private key data'", string(data))
	}
	info, err := os.Stat(dstPath)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode = %o, want 600", perm)
	}
}

func TestAdoptHandlesTildePath(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	testdir := filepath.Join(home, ".sssh-test-keys-temp")
	if err := os.MkdirAll(testdir, 0o700); err != nil {
		t.Fatal(err)
	}

	srcPath := filepath.Join(testdir, "id_rsa")
	if err := os.WriteFile(srcPath, []byte("rsa key"), 0o400); err != nil {
		t.Fatal(err)
	}
	tildePath := "~/.sssh-test-keys-temp/id_rsa"
	_, err := Adopt(p, tildePath, "rsa-key")
	if err != nil {
		t.Fatalf("Adopt with tilde path: %v", err)
	}
	dstPath := filepath.Join(p.KeysDir, "rsa-key")
	data, _ := os.ReadFile(dstPath)
	if string(data) != "rsa key" {
		t.Error("tilde path not expanded correctly")
	}
}

func TestAdoptOverwritesExisting(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	srcPath := filepath.Join(t.TempDir(), "id_test")
	if err := os.WriteFile(srcPath, []byte("key data v1"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Adopt(p, srcPath, "test-key"); err != nil {
		t.Fatalf("first Adopt: %v", err)
	}
	dstPath := filepath.Join(p.KeysDir, "test-key")
	first, _ := os.ReadFile(dstPath)

	if err := os.WriteFile(srcPath, []byte("key data v2"), 0o600); err != nil {
		t.Fatalf("rewriting source: %v", err)
	}
	if _, err := Adopt(p, srcPath, "test-key"); err != nil {
		t.Fatalf("second Adopt: %v", err)
	}
	second, _ := os.ReadFile(dstPath)
	if string(second) != "key data v2" {
		t.Error("Adopt did not overwrite existing key")
	}
	if string(first) == string(second) {
		t.Error("adopted key was not updated")
	}
}

func TestAdoptRejectsAbsentSource(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	_, err := Adopt(p, "/nonexistent/key", "test")
	if err == nil {
		t.Fatal("expected error for missing source, got nil")
	}
}

func TestValidateAdoptedDetectsMissing(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(p.KeysDir, "present-key")
	if err := os.WriteFile(keyPath, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	missing := ValidateAdopted(p, []string{"present-key", "missing-key"})
	if len(missing) != 1 || missing[0] != "missing-key" {
		t.Errorf("ValidateAdopted = %v, want [missing-key]", missing)
	}
}

func TestValidateAdoptedEmptyList(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	missing := ValidateAdopted(p, []string{})
	if len(missing) != 0 {
		t.Errorf("ValidateAdopted on empty list = %v, want []", missing)
	}
}

func TestFingerprintMatches(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	adoptedPath := filepath.Join(p.KeysDir, "test-key")
	if err := os.WriteFile(adoptedPath, []byte("key content"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"matching", "key content", true},
		{"different", "key content modified", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FingerprintMatches(p, "test-key", []byte(tt.content)); got != tt.want {
				t.Errorf("FingerprintMatches = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestListKeysEmptyDir(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	keys, err := List(p)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("List on empty dir = %v, want []", keys)
	}
}

func TestListKeysReturnsAllKeys(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	keyNames := []string{"key1", "key2", "key3"}
	for _, name := range keyNames {
		path := filepath.Join(p.KeysDir, name)
		if err := os.WriteFile(path, []byte("data"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := List(p)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 3 {
		t.Errorf("List = %v keys, want 3", len(got))
	}
	// Check all expected keys are present (order doesn't matter).
	for _, expected := range keyNames {
		found := false
		for _, key := range got {
			if key == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("List missing key %q", expected)
		}
	}
}

func TestListKeysIgnoresDirectories(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(p.KeysDir, "subdir"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(p.KeysDir, "key1"), []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := List(p)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 || got[0] != "key1" {
		t.Errorf("List = %v, want [key1]", got)
	}
}

func TestListKeysNonexistentDir(t *testing.T) {
	p := paths.New(t.TempDir())
	// Don't create keys dir
	got, err := List(p)
	if err != nil {
		t.Fatalf("List on nonexistent dir should not error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("List on nonexistent dir = %v, want nil", got)
	}
}
