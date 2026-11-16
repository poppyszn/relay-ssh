package main

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateKeyThenSignAManifest(t *testing.T) {
	dir := t.TempDir()
	keyFile := filepath.Join(dir, "private.key")
	var out bytes.Buffer
	if err := run([]string{"-genkey", "-out", keyFile}, &out); err != nil {
		t.Fatal(err)
	}
	pub, err := base64.StdEncoding.DecodeString(strings.TrimSpace(out.String()))
	if err != nil || len(pub) != 32 {
		t.Fatalf("public key output %q", out.String())
	}
	if st, err := os.Stat(keyFile); err != nil || st.Mode().Perm() != 0o600 {
		t.Fatalf("private key file must be owner-only: %v %v", st, err)
	}

	key, _ := os.ReadFile(keyFile)
	t.Setenv("RELAY_UPDATE_SIGNING_KEY", string(key))
	art := filepath.Join(dir, "Relay-linux-amd64.tar.gz")
	_ = os.WriteFile(art, []byte("x"), 0o600)
	manifest := filepath.Join(dir, "update.json")
	if err := run([]string{"-version", "1.0.0", "-out", manifest, "linux/amd64=" + art}, &out); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(manifest)
	if !strings.Contains(string(data), `"signatureAlgo": "ed25519"`) || !strings.Contains(string(data), "Relay-linux-amd64.tar.gz") {
		t.Fatalf("unexpected manifest:\n%s", data)
	}
}

func TestSigningWithoutAKeyIsRefused(t *testing.T) {
	t.Setenv("RELAY_UPDATE_SIGNING_KEY", "")
	if err := run([]string{"-version", "1.0.0", "-out", filepath.Join(t.TempDir(), "u.json"), "linux/amd64=x"}, &bytes.Buffer{}); err == nil {
		t.Fatal("expected an error without a signing key")
	}
}

func TestMalformedArtifactArgumentIsRefused(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(make([]byte, 32))
	t.Setenv("RELAY_UPDATE_SIGNING_KEY", key)
	if err := run([]string{"-version", "1.0.0", "-out", filepath.Join(t.TempDir(), "u.json"), "linux-amd64-x"}, &bytes.Buffer{}); err == nil {
		t.Fatal("expected an error")
	}
}
