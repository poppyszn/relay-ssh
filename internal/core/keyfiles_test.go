package core

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestLooksLikePrivateKey(t *testing.T) {
	dir := t.TempDir()
	keyPath := filepath.Join(dir, "key")
	writeFile(t, keyPath, "-----BEGIN OPENSSH PRIVATE KEY-----\nabc\n")
	if !looksLikePrivateKey(keyPath) {
		t.Error("looksLikePrivateKey should detect a real key file")
	}

	notKeyPath := filepath.Join(dir, "not-key")
	writeFile(t, notKeyPath, "just some text\n")
	if looksLikePrivateKey(notKeyPath) {
		t.Error("looksLikePrivateKey should reject a non-key file")
	}
}
