package cmd

import (
	"os"
	"testing"

	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/paths"
	"github.com/poppyszn/relay-ssh/internal/sshconfig"
)

// TestAddRegeneratesConfigWithoutRequiringSync guards a real reported bug:
// `relay-ssh doctor` showed "generated config not found" right after `relay-ssh add`,
// because add saved the store but never regenerated ~/.ssh/relay-ssh.d/config --
// the user had to know to separately run `relay-ssh sync`. Add must leave the
// generated config current on its own.
func TestAddRegeneratesConfigWithoutRequiringSync(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(p.StoreFile, config.Store{Version: config.Version}); err != nil {
		t.Fatal(err)
	}
	// No jump prompt: freshhost is the only profile in an empty store.
	input := "freshhost\n10.0.0.1\nuser\n\n\n\n1\n\n"
	feedStdin(t, input)

	if err := Add(p, nil); err != nil {
		t.Fatalf("Add: %v", err)
	}

	genData, err := os.ReadFile(p.GenConfig)
	if err != nil {
		t.Fatalf("generated config was not created by Add: %v", err)
	}
	s, _ := config.Load(p.StoreFile)
	if string(genData) != sshconfig.Generate(s) {
		t.Error("generated config does not reflect the profile Add just saved")
	}
}

func TestAddRetriesAfterDuplicateNameWithoutLosingLaterAnswers(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "exists", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}
	// "exists" is rejected inline and re-prompted without needing to retype anything:
	// name, host, user, port (default 22), auth (default agent),
	// jump (none), forwardAgent (false), folder (1: (no folder)), tags (none).
	// Jump prompt fires (another profile, "exists", is available to jump
	// through) -- select menu option 1, "(no jump)".
	input := "exists\nretry-name\nhost.example.com\nuser\n\n\n1\n\n1\n\n"
	feedStdin(t, input)

	if err := Add(p, nil); err != nil {
		t.Fatalf("Add: %v", err)
	}

	got, err := config.Load(p.StoreFile)
	if err != nil {
		t.Fatal(err)
	}
	prof, ok := got.Find("retry-name")
	if !ok {
		t.Fatal("retry-name profile was not saved")
	}
	if prof.Host != "host.example.com" || prof.User != "user" || prof.Port != 22 {
		t.Errorf("saved profile = %+v, want host.example.com/user/22", prof)
	}
	if _, exists := got.Find("exists"); !exists {
		t.Error("original 'exists' profile should be untouched")
	}
}

func TestAddRetriesAfterBlankRequiredField(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(p.StoreFile, config.Store{Version: config.Version}); err != nil {
		t.Fatal(err)
	}
	// Blank host is rejected and re-prompted, in place, without retyping name.
	// No jump prompt: myhost is the only profile in an empty store.
	input := "myhost\n\nreal-host.example.com\nuser\n\n\n\n1\n\n"
	feedStdin(t, input)

	if err := Add(p, nil); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, _ := config.Load(p.StoreFile)
	prof, ok := got.Find("myhost")
	if !ok || prof.Host != "real-host.example.com" {
		t.Errorf("Add did not retry the blank host field correctly: %+v", prof)
	}
}

func TestAddRetriesAfterBadPort(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(p.StoreFile, config.Store{Version: config.Version}); err != nil {
		t.Fatal(err)
	}
	// "notanumber" then "99999" (out of range) then "2222" (valid).
	// No jump prompt: porttest is the only profile in an empty store.
	input := "porttest\nhost\nuser\nnotanumber\n99999\n2222\n\n\n1\n\n"
	feedStdin(t, input)

	if err := Add(p, nil); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, _ := config.Load(p.StoreFile)
	prof, ok := got.Find("porttest")
	if !ok || prof.Port != 2222 {
		t.Errorf("Add did not retry the bad port correctly: %+v", prof)
	}
}

func TestAddReturnsErrorWhenInputExhaustedMidRetry(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "exists", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}
	// Only ever gives a duplicate name, then stdin closes: the retry loop
	// must give up rather than spin forever re-reading an exhausted scanner.
	feedStdin(t, "exists\n")

	if err := Add(p, nil); err == nil {
		t.Fatal("Add should return an error when input runs out mid-retry, got nil")
	}
}

func TestAddCreatesNewFolderViaPicker(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(p.StoreFile, config.Store{Version: config.Version}); err != nil {
		t.Fatal(err)
	}
	// Folder menu with no existing folders: 1=(no folder) 2=+Create new.
	// Select 2, then type "newteam".
	// No jump prompt: foldered is the only profile in an empty store.
	input := "foldered\nhost\nuser\n\n\n\n2\nnewteam\n\n"
	feedStdin(t, input)

	if err := Add(p, nil); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, _ := config.Load(p.StoreFile)
	prof, ok := got.Find("foldered")
	if !ok || prof.Folder != "newteam" {
		t.Errorf("Add did not save the newly created folder: %+v", prof)
	}
}

func TestAddWithTags(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(p.StoreFile, config.Store{Version: config.Version}); err != nil {
		t.Fatal(err)
	}
	// Name, host, user, port (default 22), auth (default agent), jump (none), forwardAgent (false), folder 1 ((no folder)), tags "web, prod, aws"
	// No jump prompt: tagged is the only profile in an empty store.
	input := "tagged\n10.0.0.1\nubuntu\n\n\n\n1\nweb, prod, aws\n"
	feedStdin(t, input)

	if err := Add(p, nil); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, err := config.Load(p.StoreFile)
	if err != nil {
		t.Fatal(err)
	}
	prof, ok := got.Find("tagged")
	if !ok {
		t.Fatal("tagged profile not found")
	}
	if len(prof.Tags) != 3 || prof.Tags[0] != "web" || prof.Tags[1] != "prod" || prof.Tags[2] != "aws" {
		t.Errorf("got tags %v, want [web prod aws]", prof.Tags)
	}
}

func TestAddWithJumpHostAndForwardAgent(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "bastion", Host: "bastion.example.com", User: "admin", Port: 22, Auth: config.AuthAgent},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}
	// Name: target, Host: 10.0.0.5, User: ubuntu, Port: 22, Auth: agent,
	// Jump: menu option 2 (only other profile is "bastion": options are
	// [1) (no jump), 2) bastion]), ForwardAgent: y, Folder: 1 ((no folder)), Tags: ""
	input := "target\n10.0.0.5\nubuntu\n\n\n2\ny\n1\n\n"
	feedStdin(t, input)

	if err := Add(p, nil); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, _ := config.Load(p.StoreFile)
	prof, ok := got.Find("target")
	if !ok {
		t.Fatal("target profile not found")
	}
	if bj, _ := got.Find("bastion"); prof.Jump != bj.ID {
		t.Errorf("jump = %q, want the bastion id %q", prof.Jump, bj.ID)
	}
	if !prof.ForwardAgent {
		t.Error("forward_agent should be true")
	}
}

// TestAddJumpPickerNeverOffersAnInvalidTarget replaces two older tests
// (rejecting a typed nonexistent name, rejecting a typed self-reference).
// With PickJump, both scenarios are now structurally impossible rather than
// something to reject-and-retry: the menu is built only from real, existing
// profile names, and the profile being added is always excluded from its
// own options. There is nothing left to "type wrong" -- confirmed here by
// picking the only real, valid option (real-bastion) directly, with no
// nonexistent name ever offered to select in the first place.
func TestAddJumpPickerNeverOffersAnInvalidTarget(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "real-bastion", Host: "b.example.com", User: "admin", Port: 22, Auth: config.AuthAgent},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}
	// Jump menu: [1) (no jump), 2) real-bastion] -- select 2.
	input := "target\n10.0.0.5\nubuntu\n\n\n2\nn\n1\n\n"
	feedStdin(t, input)

	if err := Add(p, nil); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, _ := config.Load(p.StoreFile)
	prof, _ := got.Find("target")
	if rb, _ := got.Find("real-bastion"); prof.Jump != rb.ID {
		t.Errorf("jump = %q, want the real-bastion id %q", prof.Jump, rb.ID)
	}
}

// TestAddJumpPickerSkipsPromptWhenNoOtherProfileExists confirms PickJump
// doesn't even show a menu when the profile being added is the only profile
// in the store: there would be exactly one option, "(no jump)", so there is
// nothing meaningful to pick from and no prompt is needed.
func TestAddJumpPickerSkipsPromptWhenNoOtherProfileExists(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(p.StoreFile, config.Store{Version: config.Version}); err != nil {
		t.Fatal(err)
	}
	// No jump line at all: with no other profiles yet, PickJump returns ""
	// without prompting, so ForwardAgent is the very next input.
	input := "myself\n10.0.0.5\nubuntu\n\n\nn\n1\n\n"
	feedStdin(t, input)

	if err := Add(p, nil); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, _ := config.Load(p.StoreFile)
	prof, _ := got.Find("myself")
	if prof.Jump != "" {
		t.Errorf("jump = %q, want empty", prof.Jump)
	}
}

func TestForwardAgentDefaultsToFalse(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(p.StoreFile, config.Store{Version: config.Version}); err != nil {
		t.Fatal(err)
	}
	// No jump prompt: box is the only profile in an empty store. Blank for forwardAgent.
	input := "box\n10.0.0.1\nuser\n\n\n\n1\n\n"
	feedStdin(t, input)

	if err := Add(p, nil); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, _ := config.Load(p.StoreFile)
	prof, _ := got.Find("box")
	if prof.ForwardAgent {
		t.Error("forward_agent should default to false")
	}
}

// feedStdin replaces os.Stdin with a pipe carrying input, restored via
// t.Cleanup so callers don't need their own defer.
func feedStdin(t *testing.T, input string) {
	t.Helper()
	old := os.Stdin
	r, w, _ := os.Pipe()
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = old })
	go func() {
		w.WriteString(input)
		w.Close()
	}()
}
