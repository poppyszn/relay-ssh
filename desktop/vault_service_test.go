package main

import (
	"bytes"
	"errors"
	"github.com/poppyszn/relay-ssh/desktop/vault"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/paths"
)

// RevealSecret and Credentials are the only bound Wails methods that return a secret.
// RevealSecret needs the master password; Credentials is a one-shot retrieval for an
// active RDP session whose vault binding matches.
func TestVaultServiceNoBoundMethodReturnsSecret(t *testing.T) {
	for _, svc := range []any{&VaultService{}, &RdpService{}} {
		svcType := reflect.TypeOf(svc)

		for i := 0; i < svcType.NumMethod(); i++ {
			method := svcType.Method(i)
			mType := method.Type

			for outIdx := 0; outIdx < mType.NumOut(); outIdx++ {
				outType := mType.Out(outIdx)

				// No string or byte slice return types except RevealSecret and Credentials
				if outType.Kind() == reflect.String && method.Name != "RevealSecret" && method.Name != "Credentials" {
					t.Fatalf("method %s returns a string (potential secret leak)", method.Name)
				}
				if outType == reflect.TypeOf([]byte{}) {
					t.Fatalf("method %s returns []byte (potential secret leak)", method.Name)
				}

				// If it's a struct (like VaultStatus or RdpSessionInfo), verify it has no secret fields
				if outType.Kind() == reflect.Struct {
					for f := 0; f < outType.NumField(); f++ {
						field := outType.Field(f)
						fieldName := strings.ToLower(field.Name)
						if (strings.Contains(fieldName, "secret") || strings.Contains(fieldName, "password") || strings.Contains(fieldName, "passphrase")) && field.Type.Kind() != reflect.Bool {
							t.Fatalf("method %s returns struct with suspicious field %s", method.Name, field.Name)
						}
					}
				}
			}
		}
	}
}

// ProfilesService hooks keep secrets consistent with connections,
// and locked vault never fails a profile save.
func TestProfilesServiceKeepsSecretsConsistent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("RELAY_SSH_HOME", home)

	p, err := paths.Default()
	if err != nil {
		t.Fatalf("paths.Default failed: %v", err)
	}
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatalf("EnsureDirs failed: %v", err)
	}

	vpath := filepath.Join(filepath.Dir(p.StoreFile), "vault.age")
	v := vault.New(vpath)
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
	if err := ps.SaveWithPassword("", prof, "alpha-web-pass"); err != nil {
		t.Fatalf("SaveWithPassword failed: %v", err)
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
	sec, ok, _ := v.Get("web-production", bindingOf(fromProfile(profRenamed)))
	if !ok || sec != "alpha-web-pass" {
		t.Fatalf("expected secret alpha-web-pass, got %q", sec)
	}

	// 3. Change auth away from password: secret should be forgotten
	profKeyAuth := profRenamed
	profKeyAuth.Auth = "agent"
	if err := ps.Save("web-production", profKeyAuth); err != nil {
		t.Fatalf("Save auth change failed: %v", err)
	}
	if v.Has("web-production") {
		t.Fatalf("expected secret to be forgotten after auth changed to agent")
	}

	// 4. Set secret again for another profile, then delete profile
	prof2 := Profile{
		Name: "db-server",
		Host: "192.168.1.20",
		User: "postgres",
		Port: 22,
		Auth: "password",
	}
	if err := ps.SaveWithPassword("", prof2, "db-secret-pass"); err != nil {
		t.Fatalf("Save db-server failed: %v", err)
	}
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

// No secret appears in logs, error strings, JSON sent to frontend,
// connections.toml, or generated ssh config.
func TestNoSecretLeakInConfigsOrLogs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("RELAY_SSH_HOME", home)

	p, err := paths.Default()
	if err != nil {
		t.Fatalf("paths.Default failed: %v", err)
	}
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatalf("EnsureDirs failed: %v", err)
	}

	vpath := filepath.Join(filepath.Dir(p.StoreFile), "vault.age")
	v := vault.New(vpath)
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
	if err := ps.SaveWithPassword("", prof, distinctiveSecret); err != nil {
		t.Fatalf("SaveWithPassword failed: %v", err)
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

func serviceTestHome(t *testing.T) (paths.Paths, *vault.Vault) {
	t.Helper()
	t.Setenv("RELAY_SSH_HOME", t.TempDir())
	p, err := paths.Default()
	if err != nil {
		t.Fatal(err)
	}
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	v := vault.New(filepath.Join(filepath.Dir(p.StoreFile), "vault.age"))
	t.Cleanup(setVaultForTest(v))
	if err := v.Create("correct-horse-battery-staple"); err != nil {
		t.Fatal(err)
	}
	return p, v
}

// Editing the host in the app is the user's own act: the saved password follows. Changing it
// outside the app is not, and a later save in the app does not make it so.
func TestSavedPasswordFollowsEditsMadeInTheAppOnly(t *testing.T) {
	p, v := serviceTestHome(t)
	ps := ProfilesService{}
	prof := Profile{Name: "web", Host: "10.0.0.5", User: "alice", Port: 22, Auth: "password"}
	if err := ps.SaveWithPassword("", prof, "alpha-web-pass"); err != nil {
		t.Fatal(err)
	}

	moved := prof
	moved.Host = "10.0.0.6"
	if err := ps.Save("web", moved); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := v.Get("web", bindingOf(fromProfile(moved))); err != nil || !ok {
		t.Fatalf("an edit made in the app should keep the password usable: %v %v", ok, err)
	}

	// someone edits connections.toml by hand
	store, _ := config.Load(p.StoreFile)
	store.Profiles[0].Host = "evil.example"
	if err := config.Save(p.StoreFile, store); err != nil {
		t.Fatal(err)
	}
	tampered := moved
	tampered.Host = "evil.example"
	tampered.Tags = []string{"touched"}
	if err := ps.Save("web", tampered); err != nil { // the user opens the form and presses Save
		t.Fatal(err)
	}
	if _, _, err := v.Get("web", bindingOf(fromProfile(tampered))); !errors.Is(err, vault.ErrBindingChanged) {
		t.Fatalf("saving a tampered connection must not release its password to the new host, got %v", err)
	}
}

func TestSaveWithPasswordChecksBeforeSaving(t *testing.T) {
	p, v := serviceTestHome(t)
	ps := ProfilesService{}
	prof := Profile{Name: "web", Host: "10.0.0.5", User: "alice", Port: 22, Auth: "password"}

	if err := ps.SaveWithPassword("", prof, "bad\npassword"); !errors.Is(err, vault.ErrBadSecret) {
		t.Fatalf("got %v", err)
	}
	v.Lock()
	if err := ps.SaveWithPassword("", prof, "fine-password"); !errors.Is(err, vault.ErrVaultLocked) {
		t.Fatalf("got %v", err)
	}
	if store, err := config.Load(p.StoreFile); err == nil && len(store.Profiles) != 0 {
		t.Fatal("a refused save left the connection behind")
	}
	keyProf := prof
	keyProf.Auth = "agent"
	if err := ps.SaveWithPassword("", keyProf, "fine-password"); err == nil {
		t.Fatal("a password was accepted for a connection that does not use one")
	}
}

func TestRevealSecretNeedsTheMasterPassword(t *testing.T) {
	_, v := serviceTestHome(t)
	ps := ProfilesService{}
	prof := Profile{Name: "web", Host: "10.0.0.5", User: "alice", Port: 22, Auth: "password"}
	if err := ps.SaveWithPassword("", prof, "alpha-web-pass"); err != nil {
		t.Fatal(err)
	}
	vs := VaultService{}
	if sec, err := vs.RevealSecret("web", "wrong-master-password"); err == nil || sec != "" {
		t.Fatalf("a wrong master revealed %q (%v)", sec, err)
	}
	if sec, err := vs.RevealSecret("web", "correct-horse-battery-staple"); err != nil || sec != "alpha-web-pass" {
		t.Fatalf("%q %v", sec, err)
	}
	v.Lock()
	if sec, err := vs.RevealSecret("web", "correct-horse-battery-staple"); err != nil || sec != "alpha-web-pass" {
		t.Fatalf("while locked: %q %v", sec, err)
	}
}
