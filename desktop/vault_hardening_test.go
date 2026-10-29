package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func newTestVault(t *testing.T, opts ...VaultOption) *Vault {
	t.Helper()
	v := NewVault(filepath.Join(t.TempDir(), "vault.age"), opts...)
	if err := v.Create("correct-horse-battery-staple"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	return v
}

// A password is only released for the host, user and port it was saved for.
func TestVaultReleasesSecretOnlyForItsBinding(t *testing.T) {
	v := newTestVault(t)
	if err := v.Set("prod", "hunter2-hunter2", "10.0.0.5|root|22|"); err != nil {
		t.Fatal(err)
	}
	if sec, ok, err := v.Get("prod", "10.0.0.5|root|22|"); err != nil || !ok || sec != "hunter2-hunter2" {
		t.Fatalf("matching binding: %q %v %v", sec, ok, err)
	}
	sec, ok, err := v.Get("prod", "evil.example|root|22|")
	if !errors.Is(err, ErrBindingChanged) || ok || sec != "" {
		t.Fatalf("changed binding must refuse, got %q %v %v", sec, ok, err)
	}
}

// Renaming moves the secret and its binding; a rename that also edits the host rebinds only
// when the stored binding was still the old, intact one.
func TestVaultRenameRebindsOnlyAnIntactBinding(t *testing.T) {
	v := newTestVault(t)
	_ = v.Set("a", "secret-for-a-1", "h1|u|22|")
	if err := v.Rename("a", "b", "h1|u|22|", "h2|u|22|"); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := v.Get("b", "h2|u|22|"); err != nil || !ok {
		t.Fatalf("intact binding should follow the edit: %v %v", ok, err)
	}

	_ = v.Set("c", "secret-for-c-1", "h1|u|22|")
	// someone changed the host outside the app (stored profile no longer matches the binding)
	if err := v.Rename("c", "c", "tampered|u|22|", "tampered2|u|22|"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := v.Get("c", "tampered2|u|22|"); !errors.Is(err, ErrBindingChanged) {
		t.Fatalf("a tampered profile must not be blessed by a later save, got %v", err)
	}
	if _, ok, err := v.Get("c", "h1|u|22|"); err != nil || !ok {
		t.Fatalf("original binding should still work: %v %v", ok, err)
	}
}

func TestVaultRefusesPasswordsWithControlCharacters(t *testing.T) {
	v := newTestVault(t)
	for _, bad := range []string{"", "line\nbreak", "ctrl-c\x03here", "tab\there", "del\x7fchar"} {
		if err := v.Set("x", bad, ""); !errors.Is(err, ErrBadSecret) {
			t.Errorf("Set(%q) = %v, want ErrBadSecret", bad, err)
		}
	}
	if err := v.Set("x", "ünïcode pass phrase ✓", ""); err != nil {
		t.Errorf("ordinary unicode password refused: %v", err)
	}
}

// Asking whether a password exists is not "use": a page that polls it must not keep the vault open.
func TestVaultHasDoesNotKeepItUnlocked(t *testing.T) {
	clock := newTestFakeClock()
	v := newTestVault(t, WithVaultClock(clock), WithIdleTimeout(15*time.Minute))
	_ = v.Set("dev", "dev-secret-42x", "")
	for i := 0; i < 5; i++ {
		clock.Advance(4 * time.Minute)
		v.Has("dev")
		v.Status()
	}
	if v.Status().Unlocked {
		t.Fatal("polling Has/Status for 20 minutes kept the vault unlocked")
	}
}

// A timer that fired a moment before a real use must not lock the vault under that use.
func TestVaultUseAtTheBoundaryWins(t *testing.T) {
	clock := newTestFakeClock()
	v := newTestVault(t, WithVaultClock(clock), WithIdleTimeout(15*time.Minute))
	_ = v.Set("dev", "dev-secret-42x", "")
	clock.Advance(14 * time.Minute)
	if _, _, err := v.Get("dev", ""); err != nil {
		t.Fatal(err)
	}
	clock.Advance(14 * time.Minute)
	if !v.Status().Unlocked {
		t.Fatal("use 14 minutes ago should have restarted the countdown")
	}
}

func TestVaultWithoutAPathRefusesInsteadOfWritingToTheWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	_ = os.Chdir(dir)
	t.Cleanup(func() { _ = os.Chdir(old) })

	v := NewVault("")
	if err := v.Create("correct-horse-battery-staple"); !errors.Is(err, ErrNoVaultPath) {
		t.Fatalf("Create = %v, want ErrNoVaultPath", err)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("wrote %d file(s) into the working directory", len(entries))
	}
}

// A write that fails leaves the in-memory vault as it was, so what the UI shows matches the file.
func TestVaultFailedSaveRollsBack(t *testing.T) {
	dir := t.TempDir()
	v := NewVault(filepath.Join(dir, "vault.age"))
	if err := v.Create("correct-horse-battery-staple"); err != nil {
		t.Fatal(err)
	}
	_ = v.Set("keep", "original-secret-1", "b1")
	if err := os.Chmod(dir, 0o500); err != nil {
		t.Skip("cannot make the folder read-only here")
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })
	if os.Geteuid() == 0 {
		t.Skip("root ignores folder permissions")
	}

	if err := v.Set("keep", "replacement-secret", "b2"); err == nil {
		t.Fatal("expected the save to fail")
	}
	if sec, ok, err := v.Get("keep", "b1"); err != nil || !ok || sec != "original-secret-1" {
		t.Fatalf("failed Set changed the secret: %q %v %v", sec, ok, err)
	}
	if err := v.Rename("keep", "moved", "b1", "b1"); err == nil {
		t.Fatal("expected the rename save to fail")
	}
	if !v.Has("keep") || v.Has("moved") {
		t.Fatal("failed Rename changed the vault in memory")
	}
}

// Changing the master while locked proves the old one, then must not leave plaintext loaded after a failure.
func TestVaultChangeMasterFailureLeavesNothingLoaded(t *testing.T) {
	v := newTestVault(t)
	_ = v.Set("a", "secret-for-a-1", "")
	v.Lock()
	if err := v.ChangeMaster("not-the-master-pass", "another-long-master"); !errors.Is(err, ErrIncorrectPassword) {
		t.Fatalf("got %v", err)
	}
	if v.secretCountForTest() != 0 {
		t.Fatal("secrets stayed in memory after a failed change")
	}
}

// Vaults written before connections were bound still open; their entries just have no binding.
func TestVaultReadsTheOlderFormat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.age")
	old := NewVault(path)
	enc, err := old.encryptRawForTest("correct-horse-battery-staple", []byte(`{"version":1,"secrets":{"old":"legacy-password"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, enc, 0o600); err != nil {
		t.Fatal(err)
	}
	v := NewVault(path)
	if err := v.Unlock("correct-horse-battery-staple"); err != nil {
		t.Fatalf("legacy vault: %v", err)
	}
	if sec, ok, err := v.Get("old", ""); err != nil || !ok || sec != "legacy-password" {
		t.Fatalf("legacy secret: %q %v %v", sec, ok, err)
	}
	if _, _, err := v.Get("old", "host|user|22|"); !errors.Is(err, ErrBindingChanged) {
		t.Fatalf("an unbound legacy entry must not be released to a real host, got %v", err)
	}
}
