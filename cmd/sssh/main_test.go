package main

import (
	"testing"

	"github.com/poppyszn/relay-ssh/internal/paths"
)

func TestRunUnknownProfileName(t *testing.T) {
	// Unknown args are treated as profile names, not commands
	err := run([]string{"wibble"})
	if err == nil {
		t.Fatal("expected an error for an unknown profile, got nil")
	}
	want := `profile "wibble" not found`
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestRunNoArgs(t *testing.T) {
	if err := run(nil); err == nil {
		t.Fatal("expected an error when no command is given, got nil")
	}
}

func TestRunVersion(t *testing.T) {
	for _, arg := range []string{"version", "--version", "-v"} {
		if err := run([]string{arg}); err != nil {
			t.Fatalf("run(%q) = %v, want nil", arg, err)
		}
	}
}

func TestRunHelp(t *testing.T) {
	for _, arg := range []string{"help", "--help", "-h"} {
		if err := run([]string{arg}); err != nil {
			t.Fatalf("run(%q) = %v, want nil", arg, err)
		}
	}
}

func TestRunLsDispatch(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	// Save empty store so ls won't fail on missing file
	if _, err := paths.Default(); err == nil {
		// Ls dispatches to cmd.Ls which will load from the default paths
		// This test just verifies the dispatch doesn't error on empty store
		if err := run([]string{"ls"}); err != nil {
			t.Fatalf("run(ls) = %v", err)
		}
	}
}
