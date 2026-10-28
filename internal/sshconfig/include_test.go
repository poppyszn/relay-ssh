package sshconfig

import (
	"os"
	"testing"

	"github.com/poppyszn/super-ssh-v2/internal/paths"
)

func TestHasBlock(t *testing.T) {
	tests := map[string]bool{
		BeginMarker + "\n" + EndMarker:                                     true,
		"some config\n" + BeginMarker + "\nsome line\n" + EndMarker + "\n": true,
		"no managed block here":                                            false,
		"":                                                                 false,
		BeginMarker + "\n":                                                 false,
		EndMarker + "\n":                                                   false,
	}
	for content, want := range tests {
		if got := HasBlock(content); got != want {
			t.Errorf("HasBlock(%q) = %v, want %v", content, got, want)
		}
	}
}

func TestAddBlockIdempotent(t *testing.T) {
	content := "Host example\n    HostName example.com\n"
	once := AddBlock(content)
	twice := AddBlock(once)
	if once != twice {
		t.Errorf("AddBlock not idempotent\n--- first ---\n%s\n--- second ---\n%s", once, twice)
	}
}

func TestAddBlockPlacesAtTop(t *testing.T) {
	content := "Host example\n    HostName example.com\n"
	got := AddBlock(content)
	if !HasBlock(got) || !BlockIsFirst(got) {
		t.Error("AddBlock failed to place block at top")
	}
}

func TestAddBlockToEmptyConfig(t *testing.T) {
	got := AddBlock("")
	if got != Block {
		t.Errorf("AddBlock on empty config = %q, want %q", got, Block)
	}
}

func TestRemoveBlockIsInverse(t *testing.T) {
	tests := []string{
		"Host example\n    HostName example.com\n",
		"Host a\n    HostName a.com\nHost b\n    HostName b.com\n",
		"",
		"# Some comment\n",
	}
	for _, orig := range tests {
		got := RemoveBlock(AddBlock(orig))
		if got != orig {
			t.Errorf("RemoveBlock(AddBlock(x)) != x for %q\n--- got ---\n%s", orig, got)
		}
	}
}

func TestRemoveBlockLeavesUnmanagedContent(t *testing.T) {
	content := "# user comment\nHost example\n    HostName example.com\n"
	managed := AddBlock(content)
	removed := RemoveBlock(managed)
	if !contains(removed, "# user comment") {
		t.Error("RemoveBlock deleted user content")
	}
	if !contains(removed, "Host example") {
		t.Error("RemoveBlock deleted user content")
	}
	if contains(removed, BeginMarker) {
		t.Error("RemoveBlock left a marker")
	}
}

func TestRemoveBlockBothMarkers(t *testing.T) {
	content := BeginMarker + "\n" + IncludeLine + "\n" + EndMarker + "\nHost a\n    HostName a.com\n"
	got := RemoveBlock(content)
	if contains(got, BeginMarker) || contains(got, EndMarker) {
		t.Errorf("RemoveBlock left markers:\n%s", got)
	}
}

func TestBlockIsFirstTrue(t *testing.T) {
	content := BeginMarker + "\n" + EndMarker + "\nHost example\n    HostName example.com\n"
	if !BlockIsFirst(content) {
		t.Error("BlockIsFirst returned false for block at top")
	}
}

func TestBlockIsFirstFalse(t *testing.T) {
	tests := []string{
		"Host *\n    IdentityFile ~/.ssh/id_ed25519\n" + BeginMarker,
		"Match exec \"test\"\n" + BeginMarker,
	}
	for _, content := range tests {
		if BlockIsFirst(content) {
			t.Errorf("BlockIsFirst returned true but block is not first: %s", content)
		}
	}
}

func TestEnsureIncludeCreatesNewFile(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	if err := EnsureInclude(p); err != nil {
		t.Fatalf("EnsureInclude: %v", err)
	}
	data, err := os.ReadFile(p.SSHConfig)
	if err != nil {
		t.Fatalf("reading config: %v", err)
	}
	if string(data) != Block {
		t.Errorf("created config = %q, want %q", string(data), Block)
	}
	info, err := os.Stat(p.SSHConfig)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode = %o, want 600", perm)
	}
}

func TestEnsureIncludeIsIdempotent(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	if err := EnsureInclude(p); err != nil {
		t.Fatalf("first EnsureInclude: %v", err)
	}
	data1, _ := os.ReadFile(p.SSHConfig)
	if err := EnsureInclude(p); err != nil {
		t.Fatalf("second EnsureInclude: %v", err)
	}
	data2, _ := os.ReadFile(p.SSHConfig)
	if string(data1) != string(data2) {
		t.Errorf("EnsureInclude not idempotent")
	}
}

func TestEnsureIncludePreservesPermissions(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	existingConfig := "Host legacy\n    HostName legacy.example.com\n"
	if err := os.WriteFile(p.SSHConfig, []byte(existingConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := EnsureInclude(p); err != nil {
		t.Fatalf("EnsureInclude: %v", err)
	}
	info, err := os.Stat(p.SSHConfig)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o644 {
		t.Errorf("mode = %o, want 644 (preserved)", perm)
	}
}

func TestEnsureIncludePrependsToExistingConfig(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	existingConfig := "Host legacy\n    HostName legacy.example.com\n"
	if err := os.WriteFile(p.SSHConfig, []byte(existingConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := EnsureInclude(p); err != nil {
		t.Fatalf("EnsureInclude: %v", err)
	}
	data, _ := os.ReadFile(p.SSHConfig)
	content := string(data)
	if !BlockIsFirst(content) {
		t.Error("block not at top after EnsureInclude")
	}
	if !contains(content, "Host legacy") {
		t.Error("existing config was deleted")
	}
}

func contains(s, substr string) bool {
	// Just a helper since we only import testing
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
