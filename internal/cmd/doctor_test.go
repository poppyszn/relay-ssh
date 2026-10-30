package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/paths"
	"github.com/poppyszn/relay-ssh/internal/sshconfig"
)

// TestDoctorFixSurfacesFailureInsteadOfSwallowingIt guards against a real
// review finding: --fix used to discard sshconfig.Write/EnsureInclude errors
// with `_ = ...`, so a fix that failed (permissions, disk full, whatever)
// left the user staring at a recheck that still failed with zero indication
// why the "fix" didn't help.
func TestDoctorFixSurfacesFailureInsteadOfSwallowingIt(t *testing.T) {
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
	// Force sshconfig.Write to fail: replace the sandbox directory with a
	// plain file, so creating sandbox/config can't succeed.
	if err := os.RemoveAll(p.Sandbox); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Sandbox, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}

	out := capture(func() error {
		return DoctorCmd(p, "--fix")
	})
	lower := strings.ToLower(out)
	if !strings.Contains(lower, "fix") || !strings.Contains(lower, "failed") {
		t.Errorf("doctor --fix should report the failure instead of silently swallowing it, got: %q", out)
	}
}

func TestDoctorHealthySystem(t *testing.T) {
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
	if err := sshconfig.Write(p, s); err != nil {
		t.Fatal(err)
	}
	if err := sshconfig.EnsureInclude(p); err != nil {
		t.Fatal(err)
	}

	// Doctor should run without error
	if err := DoctorCmd(p); err != nil {
		t.Fatalf("DoctorCmd on healthy system: %v", err)
	}
}

func TestDoctorDetectsMissingConnections(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	// Don't create connections.toml
	if err := DoctorCmd(p); err != nil {
		t.Fatalf("DoctorCmd should handle missing store, got: %v", err)
	}
}

func TestDoctorDetectsOutOfDateConfig(t *testing.T) {
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

	// Write stale generated config
	if err := os.WriteFile(p.GenConfig, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}

	// Doctor should still run (and detect the issue)
	if err := DoctorCmd(p); err != nil {
		t.Fatalf("DoctorCmd should detect staleness, got: %v", err)
	}
}

func TestDoctorFixRegeneratesStaleConfig(t *testing.T) {
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
	if err := os.WriteFile(p.GenConfig, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := DoctorCmd(p, "--fix"); err != nil {
		t.Fatalf("DoctorCmd --fix: %v", err)
	}

	data, err := os.ReadFile(p.GenConfig)
	if err != nil {
		t.Fatal(err)
	}
	expected := sshconfig.Generate(s)
	if string(data) != expected {
		t.Errorf("genConfig = %q, want %q", string(data), expected)
	}
}

func TestDoctorFixAddsMissingIncludeBlockPreservingExisting(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	existingContent := "Host myhost\n  User myuser\n"
	if err := os.WriteFile(p.SSHConfig, []byte(existingContent), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := DoctorCmd(p, "--fix"); err != nil {
		t.Fatalf("DoctorCmd --fix: %v", err)
	}

	data, err := os.ReadFile(p.SSHConfig)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !sshconfig.HasBlock(content) {
		t.Error("Include block missing after doctor --fix")
	}
	if !strings.Contains(content, "Host myhost\n  User myuser") {
		t.Error("user config was lost during doctor --fix")
	}
}

func TestDoctorFixMissingAdoptedKeyNotFabricated(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "keybox", Host: "h", User: "u", Port: 22, Auth: config.AuthKey, Key: "id_nonexistent"},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}

	if err := DoctorCmd(p, "--fix"); err != nil {
		t.Fatalf("DoctorCmd --fix: %v", err)
	}

	keyPath := filepath.Join(p.KeysDir, "id_nonexistent")
	if _, err := os.Stat(keyPath); !os.IsNotExist(err) {
		t.Errorf("doctor --fix should not create nonexistent key, but file exists: %v", err)
	}
}

func TestDoctorNoFlagLeavesStaleConfigUntouched(t *testing.T) {
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
	if err := os.WriteFile(p.GenConfig, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := DoctorCmd(p); err != nil {
		t.Fatalf("DoctorCmd: %v", err)
	}

	data, _ := os.ReadFile(p.GenConfig)
	if string(data) != "stale" {
		t.Errorf("expected GenConfig to remain stale, got %q", string(data))
	}
}

func TestDoctorWithRdpProfile(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "test-ssh", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent},
		{Protocol: config.ProtocolRDP, Name: "win-rdp", Host: "win.local", User: "admin", Port: 3389},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}
	if err := sshconfig.Write(p, s); err != nil {
		t.Fatal(err)
	}
	if err := sshconfig.EnsureInclude(p); err != nil {
		t.Fatal(err)
	}

	out := capture(func() error {
		return DoctorCmd(p)
	})
	if !strings.Contains(out, "Health: OK") {
		t.Errorf("doctor should report Health: OK with RDP profile, got:\n%s", out)
	}
}
