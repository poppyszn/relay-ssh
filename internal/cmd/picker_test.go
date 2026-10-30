package cmd

import (
	"bufio"
	"strings"
	"testing"

	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/paths"
)

func TestPickRejectsEmptyStore(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(p.StoreFile, config.Store{Version: config.Version}); err != nil {
		t.Fatal(err)
	}
	_, err := Pick(p)
	if err == nil || !strings.Contains(err.Error(), "no profiles") {
		t.Errorf("Pick on empty store: got %v, want 'no profiles'", err)
	}
}

func TestPickReturnsSingleProfile(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "only", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}
	name, err := Pick(p)
	if err != nil {
		t.Fatalf("Pick with single profile: %v", err)
	}
	if name != "only" {
		t.Errorf("Pick returned %q, want 'only'", name)
	}
}

func TestMenuPickValid(t *testing.T) {
	in := bufio.NewScanner(strings.NewReader("2\n"))
	name, err := menuPick(in, []string{"alpha", "bravo", "charlie"})
	if err != nil {
		t.Fatalf("menuPick with valid input: %v", err)
	}
	if name != "bravo" {
		t.Errorf("menuPick returned %q, want 'bravo'", name)
	}
}

func TestMenuPickInvalid(t *testing.T) {
	tests := []string{"0", "4", "abc"}
	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			in := bufio.NewScanner(strings.NewReader(input + "\n"))
			_, err := menuPick(in, []string{"a", "b", "c"})
			if err == nil {
				t.Errorf("menuPick with %q should error", input)
			}
		})
	}
}

func TestPickSkipsRdpProfiles(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "ssh-only", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent},
		{Protocol: config.ProtocolRDP, Name: "win-rdp", Host: "h", User: "u", Port: 3389},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}
	// With one SSH profile and one RDP profile, Pick should immediately return "ssh-only"
	// (because names length is 1 for SSH profiles).
	name, err := Pick(p)
	if err != nil {
		t.Fatalf("Pick failed: %v", err)
	}
	if name != "ssh-only" {
		t.Errorf("Pick returned %q, want 'ssh-only'", name)
	}

	// With only RDP profiles, Pick should error with "no profiles"
	pOnlyRDP := paths.New(t.TempDir())
	if err := paths.EnsureDirs(pOnlyRDP); err != nil {
		t.Fatal(err)
	}
	sOnlyRDP := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Protocol: config.ProtocolRDP, Name: "win-rdp", Host: "h", User: "u", Port: 3389},
	}}
	if err := config.Save(pOnlyRDP.StoreFile, sOnlyRDP); err != nil {
		t.Fatal(err)
	}
	_, err = Pick(pOnlyRDP)
	if err == nil || !strings.Contains(err.Error(), "no profiles") {
		t.Errorf("Pick on RDP-only store should error with 'no profiles', got %v", err)
	}
}
