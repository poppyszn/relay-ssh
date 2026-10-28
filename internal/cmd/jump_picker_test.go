package cmd

import (
	"bufio"
	"strings"
	"testing"

	"github.com/poppyszn/super-ssh-v2/internal/config"
)

func storeWithPeers() config.Store {
	return config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "alpha", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent},
		{Name: "beta", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent},
	}}
}

func TestPickJumpExcludesSelf(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	// Options for "alpha" must be [1) (no jump), 2) beta] -- never alpha.
	in := bufio.NewScanner(strings.NewReader("2\n"))
	got, err := PickJump(in, storeWithPeers(), "alpha", "")
	if err != nil {
		t.Fatalf("PickJump: %v", err)
	}
	if got != "beta" {
		t.Errorf("PickJump = %q, want beta (alpha must never be offered as its own jump target)", got)
	}
}

func TestPickJumpNoJumpOption(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	in := bufio.NewScanner(strings.NewReader("1\n"))
	got, err := PickJump(in, storeWithPeers(), "alpha", "")
	if err != nil {
		t.Fatalf("PickJump: %v", err)
	}
	if got != "" {
		t.Errorf("PickJump (no jump) = %q, want empty", got)
	}
}

func TestPickJumpSkipsPromptWhenNoPeersExist(t *testing.T) {
	solo := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "alone", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent},
	}}
	// Empty input: if this actually tried to prompt, it would hit EOF and
	// return an error. Returning "", nil with no read at all proves it
	// skipped the prompt because there was nothing to offer.
	in := bufio.NewScanner(strings.NewReader(""))
	got, err := PickJump(in, solo, "alone", "")
	if err != nil {
		t.Fatalf("PickJump should skip prompting, got error: %v", err)
	}
	if got != "" {
		t.Errorf("PickJump with no peers = %q, want empty", got)
	}
}

func TestPickJumpOffersKeepCurrentFirst(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	// current="beta" -> options: [1) Keep current (beta), 2) (no jump)]
	// (alpha is self, beta is current -- both excluded from the plain list).
	in := bufio.NewScanner(strings.NewReader("1\n"))
	got, err := PickJump(in, storeWithPeers(), "alpha", "beta")
	if err != nil {
		t.Fatalf("PickJump: %v", err)
	}
	if got != "beta" {
		t.Errorf("PickJump keep-current = %q, want beta", got)
	}
}
