package main

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestRdpCertStore(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "rdp_hosts.toml")

	store, err := LoadRdpCertStore(storePath)
	if err != nil {
		t.Fatalf("LoadRdpCertStore: %v", err)
	}

	endpoint := "windows.local:3389"
	fp1 := "SHA256:11:22:33:44:55:66:77:88:99:AA:BB:CC:DD:EE:FF:00:11:22:33:44:55:66:77:88:99:AA:BB:CC:DD:EE:FF:00"
	fp2 := "SHA256:FF:EE:DD:CC:BB:AA:99:88:77:66:55:44:33:22:11:00:FF:EE:DD:CC:BB:AA:99:88:77:66:55:44:33:22:11:00"

	// 1. Unknown endpoint
	dec, stored := store.Check(endpoint, fp1)
	if dec != CertUnknown {
		t.Errorf("Check unknown endpoint = %v, want CertUnknown", dec)
	}
	if stored != "" {
		t.Errorf("stored fingerprint for unknown = %q, want empty", stored)
	}

	// 2. Trust first use
	if err := store.Trust(endpoint, fp1); err != nil {
		t.Fatalf("Trust: %v", err)
	}

	// Verify file permissions
	info, err := os.Stat(storePath)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file mode = %o, want 600", perm)
	}

	// 3. Known matching endpoint
	dec, stored = store.Check(endpoint, fp1)
	if dec != CertTrust {
		t.Errorf("Check matching endpoint = %v, want CertTrust", dec)
	}
	if stored != fp1 {
		t.Errorf("stored fingerprint = %q, want %q", stored, fp1)
	}

	// 4. Changed certificate (mismatch)
	dec, stored = store.Check(endpoint, fp2)
	if dec != CertMismatch {
		t.Errorf("Check changed endpoint = %v, want CertMismatch", dec)
	}
	if stored != fp1 {
		t.Errorf("mismatch old fingerprint = %q, want %q", stored, fp1)
	}

	// 5. Explicitly trusting new certificate overwrites
	if err := store.Trust(endpoint, fp2); err != nil {
		t.Fatalf("Trust new cert: %v", err)
	}
	dec, stored = store.Check(endpoint, fp2)
	if dec != CertTrust || stored != fp2 {
		t.Errorf("Check after re-trust = (%v, %q), want (CertTrust, %q)", dec, stored, fp2)
	}

	// 6. Persistence across reload
	reloaded, err := LoadRdpCertStore(storePath)
	if err != nil {
		t.Fatalf("reloading store: %v", err)
	}
	dec, stored = reloaded.Check(endpoint, fp2)
	if dec != CertTrust || stored != fp2 {
		t.Errorf("reloaded Check = (%v, %q), want (CertTrust, %q)", dec, stored, fp2)
	}
}

func TestRdpCertStoreCorruptedFile(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "rdp_hosts.toml")
	if err := os.WriteFile(storePath, []byte("invalid toml [[]]]["), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRdpCertStore(storePath); err == nil {
		t.Fatal("expected error loading corrupted TOML, got nil")
	}
}

func TestRdpCertStoreConcurrency(t *testing.T) {
	dir := t.TempDir()
	storePath := filepath.Join(dir, "rdp_hosts.toml")
	store, err := LoadRdpCertStore(storePath)
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ep := "host:3389"
			fp := "SHA256:AA"
			store.Check(ep, fp)
			_ = store.Trust(ep, fp)
			store.Check(ep, fp)
		}(i)
	}
	wg.Wait()
}
