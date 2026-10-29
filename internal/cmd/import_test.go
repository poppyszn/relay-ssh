package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/paths"
)

func TestImportParsesHostEntries(t *testing.T) {
	home := t.TempDir()
	sshDir := filepath.Join(home, ".ssh")
	os.MkdirAll(sshDir, 0o700)

	sshConfig := `Host prod
HostName prod.example.com
User deploy
Port 2222
IdentityFile ~/.ssh/id_prod

Host staging
HostName staging.local
User admin
`
	if err := os.WriteFile(filepath.Join(sshDir, "config"), []byte(sshConfig), 0o600); err != nil {
		t.Fatal(err)
	}

	old := os.Getenv("HOME")
	os.Setenv("HOME", home)
	defer os.Setenv("HOME", old)

	p := paths.New(home)
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(p.StoreFile, config.Store{Version: config.Version}); err != nil {
		t.Fatal(err)
	}

	err := ImportCmd(p, nil)
	if err != nil {
		t.Fatalf("ImportCmd: %v", err)
	}

	s, _ := config.Load(p.StoreFile)
	if len(s.Profiles) != 2 {
		t.Errorf("import: got %d profiles, want 2", len(s.Profiles))
	}

	prod, ok := s.Find("prod")
	if !ok {
		t.Fatal("prod profile not found after import")
	}
	if prod.Host != "prod.example.com" || prod.User != "deploy" || prod.Port != 2222 {
		t.Errorf("prod profile mismatch: %+v", prod)
	}
	if prod.Auth != config.AuthKey || prod.Key != "~/.ssh/id_prod" {
		t.Errorf("prod auth mismatch: auth=%v key=%q", prod.Auth, prod.Key)
	}
}

func TestImportSkipsDuplicates(t *testing.T) {
	home := t.TempDir()
	sshDir := filepath.Join(home, ".ssh")
	os.MkdirAll(sshDir, 0o700)

	sshConfig := `Host existing
HostName example.com
User u
`
	if err := os.WriteFile(filepath.Join(sshDir, "config"), []byte(sshConfig), 0o600); err != nil {
		t.Fatal(err)
	}

	old := os.Getenv("HOME")
	os.Setenv("HOME", home)
	defer os.Setenv("HOME", old)

	p := paths.New(home)
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	existing := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "existing", Host: "already.com", User: "u", Port: 22, Auth: config.AuthAgent},
	}}
	if err := config.Save(p.StoreFile, existing); err != nil {
		t.Fatal(err)
	}

	if err := ImportCmd(p, nil); err != nil {
		t.Fatalf("ImportCmd: %v", err)
	}

	s, _ := config.Load(p.StoreFile)
	if len(s.Profiles) != 1 {
		t.Errorf("import should skip duplicates: got %d, want 1", len(s.Profiles))
	}
	if s.Profiles[0].Host != "already.com" {
		t.Error("import overwrote existing profile")
	}
}
