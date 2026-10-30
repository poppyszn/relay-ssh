package cmd

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/core"
	"github.com/poppyszn/relay-ssh/internal/paths"
)

func TestBackupCreatesEncryptedFile(t *testing.T) {
	t.Setenv("RELAY_SSH_BACKUP_PASSPHRASE", "secret123")
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "test", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}

	old, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(old)

	if err := BackupCmd(p, nil); err != nil {
		t.Fatalf("BackupCmd: %v", err)
	}

	if _, err := os.Stat("relay-ssh-backup.tar.gz.age"); err != nil {
		t.Errorf("expected relay-ssh-backup.tar.gz.age to be created, got: %v", err)
	}
}

func TestBackupAppendsAgeSuffix(t *testing.T) {
	t.Setenv("RELAY_SSH_BACKUP_PASSPHRASE", "secret123")
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "test", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}

	old, _ := os.Getwd()
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)
	defer os.Chdir(old)

	// Test 1: -o without .age suffix gets .age appended
	if err := BackupCmd(p, []string{"-o", "mybackup.tar.gz"}); err != nil {
		t.Fatalf("BackupCmd: %v", err)
	}
	if _, err := os.Stat("mybackup.tar.gz.age"); err != nil {
		t.Errorf("expected mybackup.tar.gz.age to be created, got: %v", err)
	}

	// Test 2: -o already ending in .age is not modified
	if err := BackupCmd(p, []string{"-o", "exact.age"}); err != nil {
		t.Fatalf("BackupCmd: %v", err)
	}
	if _, err := os.Stat("exact.age"); err != nil {
		t.Errorf("expected exact.age to be created, got: %v", err)
	}
}

func TestBackupRestoreRoundTripEncrypted(t *testing.T) {
	passphrase := "correct-horse-battery-staple"
	t.Setenv("RELAY_SSH_BACKUP_PASSPHRASE", passphrase)

	pSrc := paths.New(t.TempDir())
	if err := paths.EnsureDirs(pSrc); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "prod", Host: "prod.example.com", User: "admin", Port: 2222, Auth: config.AuthAgent},
	}}
	if err := config.Save(pSrc.StoreFile, s); err != nil {
		t.Fatal(err)
	}

	backupDir := t.TempDir()
	old, _ := os.Getwd()
	os.Chdir(backupDir)
	defer os.Chdir(old)

	if err := BackupCmd(pSrc, nil); err != nil {
		t.Fatalf("BackupCmd: %v", err)
	}

	backupPath := filepath.Join(backupDir, "relay-ssh-backup.tar.gz.age")
	pDst := paths.New(t.TempDir())
	if err := paths.EnsureDirs(pDst); err != nil {
		t.Fatal(err)
	}

	if err := RestoreCmd(pDst, []string{backupPath}); err != nil {
		t.Fatalf("RestoreCmd: %v", err)
	}

	restored, err := config.Load(pDst.StoreFile)
	if err != nil {
		t.Fatalf("loading restored config: %v", err)
	}
	if len(restored.Profiles) != 1 || restored.Profiles[0].Name != "prod" {
		t.Errorf("restored profiles = %v, want [prod]", restored.Names())
	}
	if restored.Profiles[0].Port != 2222 {
		t.Errorf("restored port = %d, want 2222", restored.Profiles[0].Port)
	}
}

func TestRestoreWrongPassphraseProducesSpecificError(t *testing.T) {
	t.Setenv("RELAY_SSH_BACKUP_PASSPHRASE", "good-passphrase")

	pSrc := paths.New(t.TempDir())
	if err := paths.EnsureDirs(pSrc); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "prod", Host: "prod.example.com", User: "admin", Port: 22, Auth: config.AuthAgent},
	}}
	if err := config.Save(pSrc.StoreFile, s); err != nil {
		t.Fatal(err)
	}

	backupDir := t.TempDir()
	old, _ := os.Getwd()
	os.Chdir(backupDir)
	defer os.Chdir(old)

	if err := BackupCmd(pSrc, nil); err != nil {
		t.Fatalf("BackupCmd: %v", err)
	}

	backupPath := filepath.Join(backupDir, "relay-ssh-backup.tar.gz.age")
	pDst := paths.New(t.TempDir())
	if err := paths.EnsureDirs(pDst); err != nil {
		t.Fatal(err)
	}

	// Attempt restore with wrong passphrase
	t.Setenv("RELAY_SSH_BACKUP_PASSPHRASE", "wrong-passphrase")
	err := RestoreCmd(pDst, []string{backupPath})
	if err == nil {
		t.Fatal("expected restore with wrong passphrase to fail, got nil")
	}
	wantMsg := "incorrect passphrase (or the file is corrupted)"
	if err.Error() != wantMsg {
		t.Errorf("error = %q, want %q", err.Error(), wantMsg)
	}
}

func TestRestoreLegacyPlaintextBackup(t *testing.T) {
	// Create a plaintext .tar.gz backup
	pSrc := paths.New(t.TempDir())
	if err := paths.EnsureDirs(pSrc); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "legacy", Host: "legacy.example.com", User: "root", Port: 22, Auth: config.AuthAgent},
	}}
	if err := config.Save(pSrc.StoreFile, s); err != nil {
		t.Fatal(err)
	}

	backupDir := t.TempDir()
	plainTarGz := filepath.Join(backupDir, "legacy-backup.tar.gz")
	f, err := os.Create(plainTarGz)
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	if err := core.AddFileToTar(tw, pSrc.StoreFile, "connections.toml"); err != nil {
		t.Fatal(err)
	}
	tw.Close()
	gz.Close()
	f.Close()

	// Restore into pDst without setting RELAY_SSH_BACKUP_PASSPHRASE
	t.Setenv("RELAY_SSH_BACKUP_PASSPHRASE", "")
	pDst := paths.New(t.TempDir())
	if err := paths.EnsureDirs(pDst); err != nil {
		t.Fatal(err)
	}

	if err := RestoreCmd(pDst, []string{plainTarGz}); err != nil {
		t.Fatalf("RestoreCmd on plaintext backup: %v", err)
	}

	restored, err := config.Load(pDst.StoreFile)
	if err != nil {
		t.Fatalf("loading restored config: %v", err)
	}
	if len(restored.Profiles) != 1 || restored.Profiles[0].Name != "legacy" {
		t.Errorf("restored profiles = %v, want [legacy]", restored.Names())
	}
}

func TestBackupAndRestorePreservesRdpProfile(t *testing.T) {
	t.Setenv("RELAY_SSH_BACKUP_PASSPHRASE", "test-passphrase")

	pSrc := paths.New(t.TempDir())
	if err := paths.EnsureDirs(pSrc); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{
			Protocol: config.ProtocolRDP,
			Name:     "win-backup-test",
			Host:     "10.0.0.99",
			User:     "Administrator",
			Domain:   "TESTDOM",
			Port:     3389,
		},
	}}
	if err := config.Save(pSrc.StoreFile, s); err != nil {
		t.Fatal(err)
	}

	backupDir := t.TempDir()
	backupPath := filepath.Join(backupDir, "backup.tar.gz.age")

	if err := BackupCmd(pSrc, []string{"-o", backupPath}); err != nil {
		t.Fatalf("BackupCmd: %v", err)
	}

	pDst := paths.New(t.TempDir())
	if err := paths.EnsureDirs(pDst); err != nil {
		t.Fatal(err)
	}

	if err := RestoreCmd(pDst, []string{backupPath}); err != nil {
		t.Fatalf("RestoreCmd: %v", err)
	}

	restored, err := config.Load(pDst.StoreFile)
	if err != nil {
		t.Fatalf("loading restored config: %v", err)
	}
	if len(restored.Profiles) != 1 {
		t.Fatalf("restored profiles count = %d, want 1", len(restored.Profiles))
	}
	p := restored.Profiles[0]
	if p.Name != "win-backup-test" || p.Protocol != config.ProtocolRDP || p.Domain != "TESTDOM" || p.Port != 3389 {
		t.Errorf("restored RDP profile mismatch: %+v", p)
	}
}
