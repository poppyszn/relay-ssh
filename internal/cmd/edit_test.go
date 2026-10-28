package cmd

import (
	"testing"

	"github.com/poppyszn/super-ssh-v2/internal/config"
	"github.com/poppyszn/super-ssh-v2/internal/paths"
)

func TestEditPressingEnterKeepsAllFields(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	original := config.Profile{
		Name:         "mybox",
		Host:         "192.168.1.50",
		User:         "ubuntu",
		Port:         2200,
		Auth:         config.AuthAgent,
		Jump:         "bastion",
		ForwardAgent: true,
		Folder:       "servers",
		Tags:         []string{"primary", "dc1"},
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "bastion", Host: "bastion.example.com", User: "admin", Port: 22, Auth: config.AuthAgent},
		original,
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}

	// 9 prompts: name, host, user, port, auth, jump, forwardAgent, folder, tags.
	// Jump menu shows "1) Keep current (bastion)" since original.Jump is set --
	// blank isn't a valid menu selection, so pick "1" to keep it.
	feedStdin(t, "\n\n\n\n\n1\n\n\n\n")

	if err := EditCmd(p, []string{"mybox"}); err != nil {
		t.Fatalf("EditCmd: %v", err)
	}

	after, err := config.Load(p.StoreFile)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := after.Find("mybox")
	if !ok {
		t.Fatal("profile mybox not found after edit")
	}
	if got.Name != original.Name || got.Host != original.Host || got.User != original.User ||
		got.Port != original.Port || got.Auth != original.Auth || got.Jump != original.Jump ||
		got.ForwardAgent != original.ForwardAgent || got.Folder != original.Folder ||
		len(got.Tags) != 2 || got.Tags[0] != "primary" || got.Tags[1] != "dc1" {
		t.Errorf("profile changed when all prompts received enter: got %+v, want %+v", got, original)
	}
}

func TestEditModifiesSingleField(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	original := config.Profile{
		Name:   "web",
		Host:   "10.0.0.1",
		User:   "deploy",
		Port:   22,
		Auth:   config.AuthAgent,
		Folder: "infra",
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{original}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}

	// Keep name, host -> 10.0.0.99, keep user, port, auth, jump, forwardAgent, folder, tags
	feedStdin(t, "\n10.0.0.99\n\n\n\n\n\n\n\n")

	if err := EditCmd(p, []string{"web"}); err != nil {
		t.Fatalf("EditCmd: %v", err)
	}

	after, err := config.Load(p.StoreFile)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := after.Find("web")
	if !ok {
		t.Fatal("profile web not found")
	}
	if got.Host != "10.0.0.99" {
		t.Errorf("got host %q, want 10.0.0.99", got.Host)
	}
	if got.User != original.User || got.Port != original.Port || got.Auth != original.Auth || got.Folder != original.Folder {
		t.Errorf("other fields were modified: %+v", got)
	}
}

func TestEditTags(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	original := config.Profile{
		Name: "tagbox",
		Host: "10.0.0.1",
		User: "deploy",
		Port: 22,
		Auth: config.AuthAgent,
		Tags: []string{"old1", "old2"},
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{original}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}

	// Keep name, host, user, port, auth (5 enters). No jump prompt at all:
	// tagbox is the only profile in the store, so PickJump has nothing to
	// offer besides "(no jump)" and skips prompting entirely. Then
	// forwardAgent, folder (2 more enters), then change tags to "new1, new2".
	feedStdin(t, "\n\n\n\n\n\n\nnew1, new2\n")

	if err := EditCmd(p, []string{"tagbox"}); err != nil {
		t.Fatalf("EditCmd: %v", err)
	}

	after, err := config.Load(p.StoreFile)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := after.Find("tagbox")
	if len(got.Tags) != 2 || got.Tags[0] != "new1" || got.Tags[1] != "new2" {
		t.Errorf("expected tags [new1 new2], got %v", got.Tags)
	}

	// Now clear tags using "-" (same 7-enter count, no jump prompt)
	feedStdin(t, "\n\n\n\n\n\n\n-\n")
	if err := EditCmd(p, []string{"tagbox"}); err != nil {
		t.Fatalf("EditCmd clearing tags: %v", err)
	}
	after, _ = config.Load(p.StoreFile)
	got, _ = after.Find("tagbox")
	if len(got.Tags) != 0 {
		t.Errorf("expected empty tags after '-', got %v", got.Tags)
	}
}

// TestEditJumpPickerExcludesSelfEvenWhileRenaming is a regression test for a
// real bug: PickJump was called with the profile's *new*, not-yet-saved name
// as the exclusion target during a rename. Since that new name doesn't exist
// yet in the store, nothing actually got excluded -- the profile's own
// current (old) name slipped through as a selectable, self-referencing jump
// option. Fixed by always excluding prof.Name (the name as it exists in s
// right now), regardless of what new name the user is choosing.
func TestEditJumpPickerExcludesSelfEvenWhileRenaming(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "other", Host: "1.1.1.1", User: "root", Port: 22, Auth: config.AuthAgent},
		{Name: "renameme", Host: "2.2.2.2", User: "root", Port: 22, Auth: config.AuthAgent},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}

	// Rename renameme -> renamed. Jump menu must offer only "(no jump)" and
	// "other" -- never "renameme", its own old name, as a selectable jump
	// target. If the bug were present, "renameme" would appear as option 2
	// and this would try to select "other" (menu position 3) or crash.
	feedStdin(t, "renamed\n\n\n\n\n2\n\n\n\n")

	if err := EditCmd(p, []string{"renameme"}); err != nil {
		t.Fatalf("EditCmd: %v", err)
	}
	after, _ := config.Load(p.StoreFile)
	got, ok := after.Find("renamed")
	if !ok {
		t.Fatal("renamed not found")
	}
	if got.Jump != "other" {
		t.Errorf("jump = %q, want other (menu option 2 in a 2-option list: no-jump, other)", got.Jump)
	}
}

func TestEditRenamingRejectsExistingCollision(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "box-a", Host: "1.1.1.1", User: "root", Port: 22, Auth: config.AuthAgent},
		{Name: "box-b", Host: "2.2.2.2", User: "root", Port: 22, Auth: config.AuthAgent},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}

	// Edit box-b: try renaming to box-a (collision, rejected inline),
	// then enter box-c (valid), then keep host/user/port/auth (4 blanks).
	// Jump menu: [1) (no jump), 2) box-a] -- box-b (self) correctly excluded
	// even though the rename hasn't been saved yet -- select "1".
	// Then forwardAgent, folder, tags (3 blanks).
	feedStdin(t, "box-a\nbox-c\n\n\n\n\n1\n\n\n\n")

	if err := EditCmd(p, []string{"box-b"}); err != nil {
		t.Fatalf("EditCmd: %v", err)
	}

	after, err := config.Load(p.StoreFile)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := after.Find("box-b"); ok {
		t.Error("box-b should have been renamed")
	}
	got, ok := after.Find("box-c")
	if !ok {
		t.Fatal("box-c not found after rename")
	}
	if got.Host != "2.2.2.2" {
		t.Errorf("box-c has wrong host: %s", got.Host)
	}
}

func TestEditRenamingToSameNameIsAccepted(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "box-a", Host: "1.1.1.1", User: "root", Port: 22, Auth: config.AuthAgent},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}

	// Explicitly type "box-a" for name prompt, then keep remaining 8 fields
	feedStdin(t, "box-a\n\n\n\n\n\n\n\n\n")

	if err := EditCmd(p, []string{"box-a"}); err != nil {
		t.Fatalf("EditCmd: %v", err)
	}

	after, err := config.Load(p.StoreFile)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := after.Find("box-a"); !ok {
		t.Error("box-a should still exist")
	}
}

func TestEditJumpThreeHopCycleRejectedAtSaveAndRepromptsJump(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	// Setup 3-hop chain: hop-a -> hop-b -> hop-c -> (none)
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "hop-a", Host: "1.1.1.1", User: "u", Port: 22, Auth: config.AuthAgent, Jump: "hop-b"},
		{Name: "hop-b", Host: "2.2.2.2", User: "u", Port: 22, Auth: config.AuthAgent, Jump: "hop-c"},
		{Name: "hop-c", Host: "3.3.3.3", User: "u", Port: 22, Auth: config.AuthAgent},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}

	// Edit hop-c:
	// 1-5: name, host, user, port, auth (all enter, keep current)
	// 6: Jump picker: [1) (no jump), 2) hop-a, 3) hop-b] -- pick "2" (hop-a)!
	//    Forms a 3-hop cycle: hop-c -> hop-a -> hop-b -> hop-c.
	// 7-10: ForwardAgent, Folder-change, Tags, "Add a port forward?" (all enter)
	// Caught at save by Store.Validate(). System re-prompts ONLY the jump
	// picker: [1) (no jump), 2) hop-a, 3) hop-b] -- pick "1" to clear it.
	feedStdin(t, "\n\n\n\n\n2\n\n\n\n\n1\n")

	if err := EditCmd(p, []string{"hop-c"}); err != nil {
		t.Fatalf("EditCmd: %v", err)
	}

	after, err := config.Load(p.StoreFile)
	if err != nil {
		t.Fatal(err)
	}
	prof, ok := after.Find("hop-c")
	if !ok {
		t.Fatal("hop-c not found")
	}
	if prof.Jump != "" {
		t.Errorf("hop-c jump should have been cleared, got %q", prof.Jump)
	}
}
