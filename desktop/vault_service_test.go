package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/poppyszn/relay-ssh/internal/paths"
)

// Section 6.1: No bound Wails method returns a secret.
func TestVaultServiceNoBoundMethodReturnsSecret(t *testing.T) {
	svc := VaultService{}
	svcType := reflect.TypeOf(svc)

	for i := 0; i < svcType.NumMethod(); i++ {
		method := svcType.Method(i)
		mType := method.Type

		for outIdx := 0; outIdx < mType.NumOut(); outIdx++ {
			outType := mType.Out(outIdx)

			// No string or byte slice return types
			if outType.Kind() == reflect.String {
				t.Fatalf("method %s returns a string (potential secret leak)", method.Name)
			}
			if outType == reflect.TypeOf([]byte{}) {
				t.Fatalf("method %s returns []byte (potential secret leak)", method.Name)
			}

			// If it's a struct (like VaultStatus), verify it has no secret fields
			if outType.Kind() == reflect.Struct {
				for f := 0; f < outType.NumField(); f++ {
					field := outType.Field(f)
					fieldName := strings.ToLower(field.Name)
					if strings.Contains(fieldName, "secret") || strings.Contains(fieldName, "password") || strings.Contains(fieldName, "passphrase") {
						t.Fatalf("method %s returns struct with suspicious field %s", method.Name, field.Name)
					}
				}
			}
		}
	}
}

// Section 4.5: ProfilesService hooks keep secrets consistent with connections,
// and locked vault never fails a profile save.
func TestProfilesServiceKeepsSecretsConsistent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SSSH_HOME", home)

	p, err := paths.Default()
	if err != nil {
		t.Fatalf("paths.Default failed: %v", err)
	}
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatalf("EnsureDirs failed: %v", err)
	}

	vpath := filepath.Join(filepath.Dir(p.StoreFile), "vault.age")
	v := NewVault(vpath)
	cleanup := setVaultForTest(v)
	defer cleanup()

	master := "correct-horse-battery-staple"
	if err := v.Create(master); err != nil {
		t.Fatalf("vault Create failed: %v", err)
	}

	ps := ProfilesService{}

	// 1. Save new profile with auth: "password"
	prof := Profile{
		Name: "web-server",
		Host: "192.168.1.10",
		User: "ubuntu",
		Port: 22,
		Auth: "password",
	}
	if err := ps.Save("", prof); err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if err := v.Set("web-server", "alpha-web-pass"); err != nil {
		t.Fatalf("Set secret failed: %v", err)
	}
	if !v.Has("web-server") {
		t.Fatalf("expected web-server to have secret")
	}

	// 2. Rename connection: secret should follow
	profRenamed := prof
	profRenamed.Name = "web-production"
	if err := ps.Save("web-server", profRenamed); err != nil {
		t.Fatalf("Save rename failed: %v", err)
	}
	if v.Has("web-server") {
		t.Fatalf("expected old name web-server to be gone")
	}
	if !v.Has("web-production") {
		t.Fatalf("expected new name web-production to have secret")
	}
	sec, ok, _ := v.Get("web-production")
	if !ok || sec != "alpha-web-pass" {
		t.Fatalf("expected secret alpha-web-pass, got %q", sec)
	}

	// 3. Change auth away from password: secret should be forgotten
	profKeyAuth := profRenamed
	profKeyAuth.Auth = "key"
	profKeyAuth.Key = "id_ed25519"
	if err := ps.Save("web-production", profKeyAuth); err != nil {
		t.Fatalf("Save auth change failed: %v", err)
	}
	if v.Has("web-production") {
		t.Fatalf("expected secret to be forgotten after auth changed to key")
	}

	// 4. Set secret again for another profile, then delete profile
	prof2 := Profile{
		Name: "db-server",
		Host: "192.168.1.20",
		User: "postgres",
		Port: 22,
		Auth: "password",
	}
	if err := ps.Save("", prof2); err != nil {
		t.Fatalf("Save db-server failed: %v", err)
	}
	_ = v.Set("db-server", "db-secret-pass")
	if !v.Has("db-server") {
		t.Fatalf("expected db-server to have secret")
	}
	if err := ps.Delete("db-server"); err != nil {
		t.Fatalf("Delete db-server failed: %v", err)
	}
	if v.Has("db-server") {
		t.Fatalf("expected secret to be forgotten after Delete")
	}

	// 5. Locked vault never fails a profile save or delete
	v.Lock()
	prof3 := Profile{
		Name: "cache-server",
		Host: "192.168.1.30",
		User: "redis",
		Port: 22,
		Auth: "password",
	}
	if err := ps.Save("", prof3); err != nil {
		t.Fatalf("Save while locked failed: %v", err)
	}
	prof3Renamed := prof3
	prof3Renamed.Name = "cache-cluster"
	if err := ps.Save("cache-server", prof3Renamed); err != nil {
		t.Fatalf("Save rename while locked failed: %v", err)
	}
	if err := ps.Delete("cache-cluster"); err != nil {
		t.Fatalf("Delete while locked failed: %v", err)
	}
}

// Section 6.2: No secret appears in logs, error strings, JSON sent to frontend,
// connections.toml, or generated ssh config.
func TestNoSecretLeakInConfigsOrLogs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SSSH_HOME", home)

	p, err := paths.Default()
	if err != nil {
		t.Fatalf("paths.Default failed: %v", err)
	}
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatalf("EnsureDirs failed: %v", err)
	}

	vpath := filepath.Join(filepath.Dir(p.StoreFile), "vault.age")
	v := NewVault(vpath)
	cleanup := setVaultForTest(v)
	defer cleanup()

	master := "correct-horse-battery-staple"
	distinctiveSecret := "canary-super-secret-password-xyz-987654"

	vs := VaultService{}
	if err := vs.Create(master); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	ps := ProfilesService{}
	prof := Profile{
		Name: "leak-test-host",
		Host: "10.0.0.1",
		User: "admin",
		Port: 22,
		Auth: "password",
	}
	if err := ps.Save("", prof); err != nil {
		t.Fatalf("Save profile failed: %v", err)
	}
	if err := vs.SetSecret("leak-test-host", distinctiveSecret); err != nil {
		t.Fatalf("SetSecret failed: %v", err)
	}

	// Grep connections.toml
	connData, err := os.ReadFile(p.StoreFile)
	if err != nil {
		t.Fatalf("reading connections.toml: %v", err)
	}
	if bytes.Contains(connData, []byte(distinctiveSecret)) {
		t.Fatalf("secret leaked in connections.toml!")
	}

	// Grep generated ssh config
	genConfig := ps.Preview("", prof).Config
	if strings.Contains(genConfig, distinctiveSecret) {
		t.Fatalf("secret leaked in generated ssh config preview!")
	}

	// Grep whole config dir except vault.age
	configDir := filepath.Dir(p.StoreFile)
	entries, err := os.ReadDir(configDir)
	if err != nil {
		t.Fatalf("reading config dir: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() == "vault.age" {
			continue
		}
		path := filepath.Join(configDir, entry.Name())
		if !entry.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			if bytes.Contains(data, []byte(distinctiveSecret)) {
				t.Fatalf("secret leaked in file %s!", path)
			}
		}
	}

	// Verify vault.age does NOT contain the secret in plaintext
	vaultData, err := os.ReadFile(vpath)
	if err != nil {
		t.Fatalf("reading vault.age: %v", err)
	}
	if bytes.Contains(vaultData, []byte(distinctiveSecret)) {
		t.Fatalf("plaintext secret leaked in vault.age!")
	}
}
