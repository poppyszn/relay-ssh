package cmd

import (
	"strings"
	"testing"

	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/core"
	"github.com/poppyszn/relay-ssh/internal/paths"
)

func TestListProfileAddressesFormatsFolderQualifiedNames(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "home-server", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent, Folder: "Personal"},
		{Name: "solo", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}
	got, err := ListProfileAddresses(p)
	if err != nil {
		t.Fatalf("ListProfileAddresses: %v", err)
	}
	want := []string{"Personal/home-server", "solo"}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("ListProfileAddresses = %v, want %v", got, want)
	}
}

func TestListProfileAddressesEmptyStore(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(p.StoreFile, config.Store{Version: config.Version}); err != nil {
		t.Fatal(err)
	}
	got, err := ListProfileAddresses(p)
	if err != nil {
		t.Fatalf("ListProfileAddresses: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("ListProfileAddresses on empty store = %v, want none", got)
	}
}

// TestListProfileAddressesAreResolvable is the point of the whole feature:
// every address this function offers for completion must actually work when
// passed back to resolveProfile, or tab-completing a name would hand the
// user something that then fails with "profile not found".
func TestListProfileAddressesAreResolvable(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "dev-app-server", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent, Folder: "Acme"},
		{Name: "home-server", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent, Folder: "Personal"},
		{Name: "solo", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}
	addrs, err := ListProfileAddresses(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, addr := range addrs {
		if _, ok := core.ResolveProfile(s, addr); !ok {
			t.Errorf("completion offered %q but resolveProfile can't find it", addr)
		}
	}
}

func TestCompletionScriptKnownShells(t *testing.T) {
	for _, shell := range []string{"zsh", "bash"} {
		script, err := CompletionScript(shell)
		if err != nil {
			t.Fatalf("CompletionScript(%q): %v", shell, err)
		}
		if !strings.Contains(script, "__complete-profiles") {
			t.Errorf("CompletionScript(%q) doesn't call the completion helper: %q", shell, script)
		}
	}
}

func TestCompletionScriptRejectsUnknownShell(t *testing.T) {
	_, err := CompletionScript("fish")
	if err == nil {
		t.Fatal("CompletionScript(fish) should return an error, got nil")
	}
}
