package launch

import (
	"os"
	"testing"

	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/paths"
)

func TestSSHBuildsCorrectArgv(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.GenConfig, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	profile := config.Profile{Name: "prod-web"}

	var capturedCmd string
	var capturedArgv []string
	origExec := execFunc
	execFunc = func(cmd string, argv []string) error {
		capturedCmd = cmd
		capturedArgv = append(capturedArgv, argv...)
		return nil
	}
	defer func() { execFunc = origExec }()

	SSH(p, profile)
	if capturedCmd != "ssh" {
		t.Errorf("command = %q, want ssh", capturedCmd)
	}
	if len(capturedArgv) < 4 || capturedArgv[0] != "ssh" || capturedArgv[1] != "-F" || capturedArgv[3] != "prod-web" {
		t.Errorf("argv = %v, expected [ssh -F <path> prod-web]", capturedArgv)
	}
}

func TestSCPBuildsCorrectArgv(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.GenConfig, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	profile := config.Profile{Name: "backup-host"}

	var capturedCmd string
	var capturedArgv []string
	origExec := execFunc
	execFunc = func(cmd string, argv []string) error {
		capturedCmd = cmd
		capturedArgv = append(capturedArgv, argv...)
		return nil
	}
	defer func() { execFunc = origExec }()

	SCP(p, profile, "data/", "remote:", false)
	if capturedCmd != "scp" {
		t.Errorf("command = %q, want scp", capturedCmd)
	}
	if len(capturedArgv) < 5 || capturedArgv[0] != "scp" {
		t.Errorf("argv[0] = %q, want scp", capturedArgv[0])
	}
	if capturedArgv[len(capturedArgv)-2] != "data/" || capturedArgv[len(capturedArgv)-1] != "remote:" {
		t.Errorf("argv src/dst not correct: %v", capturedArgv)
	}
}

func TestSCPIncludesRecursiveFlag(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.GenConfig, []byte(""), 0o600); err != nil {
		t.Fatal(err)
	}
	profile := config.Profile{Name: "backup"}

	var capturedArgv []string
	origExec := execFunc
	execFunc = func(cmd string, argv []string) error {
		capturedArgv = append(capturedArgv, argv...)
		return nil
	}
	defer func() { execFunc = origExec }()

	SCP(p, profile, "data/", "remote:", true)
	hasR := false
	for _, arg := range capturedArgv {
		if arg == "-r" {
			hasR = true
			break
		}
	}
	if !hasR {
		t.Errorf("argv missing -r flag: %v", capturedArgv)
	}
}
