package main

import "testing"

// A plain `go build` or `wails3 dev` has no stamped version; release builds set it with -ldflags "-X main.version=...".
func TestVersionIsDevUnlessStamped(t *testing.T) {
	if got := (&ToolsService{}).Version(); got != "dev" {
		t.Fatalf("unstamped version = %q, want dev", got)
	}
}
