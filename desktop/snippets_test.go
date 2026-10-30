package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSnippetsRoundTripSortedAndPrivate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("RELAY_SSH_HOME", home)
	svc := SnippetsService{}

	if got, err := svc.List(); err != nil || len(got) != 0 {
		t.Fatalf("no file yet: List = %v, %v", got, err)
	}
	for _, s := range []Snippet{
		{Name: "tail nginx", Command: "tail -f /var/log/nginx/error.log"},
		{Name: "Disk usage", Command: "df -h\ndu -sh *"},
	} {
		if err := svc.Save("", s); err != nil {
			t.Fatalf("Save %q: %v", s.Name, err)
		}
	}
	got, err := svc.List()
	if err != nil || len(got) != 2 || got[0].Name != "Disk usage" || got[1].Name != "tail nginx" {
		t.Fatalf("List = %+v, %v (want sorted, case-insensitively)", got, err)
	}
	if got[0].Command != "df -h\ndu -sh *" {
		t.Errorf("a multi-line command did not survive: %q", got[0].Command)
	}
	info, err := os.Stat(filepath.Join(home, ".config", "relay-ssh", "snippets.toml"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("snippets.toml = %v, %v (want 0600)", info, err)
	}
}

func TestSnippetsRenameEditAndDelete(t *testing.T) {
	t.Setenv("RELAY_SSH_HOME", t.TempDir())
	svc := SnippetsService{}
	_ = svc.Save("", Snippet{Name: "a", Command: "echo a"})
	_ = svc.Save("", Snippet{Name: "b", Command: "echo b"})

	if err := svc.Save("a", Snippet{Name: "a2", Command: "echo changed"}); err != nil {
		t.Fatal(err)
	}
	got, _ := svc.List()
	if len(got) != 2 || got[0].Name != "a2" || got[0].Command != "echo changed" {
		t.Errorf("after rename: %+v", got)
	}
	if err := svc.Save("a2", Snippet{Name: "b", Command: "x"}); err == nil {
		t.Error("renaming onto an existing snippet should be refused")
	}
	if err := svc.Save("", Snippet{Name: "b", Command: "x"}); err == nil {
		t.Error("a new snippet may not reuse a name")
	}
	if err := svc.Delete("a2"); err != nil {
		t.Fatal(err)
	}
	if got, _ := svc.List(); len(got) != 1 || got[0].Name != "b" {
		t.Errorf("after delete: %+v", got)
	}
	if err := svc.Delete("nope"); err == nil {
		t.Error("deleting a missing snippet should say so")
	}
}

func TestSnippetsRejectBadInput(t *testing.T) {
	t.Setenv("RELAY_SSH_HOME", t.TempDir())
	svc := SnippetsService{}
	for name, s := range map[string]Snippet{
		"empty name":      {Name: "  ", Command: "ls"},
		"empty command":   {Name: "x", Command: " \n "},
		"long name":       {Name: strings.Repeat("n", 81), Command: "ls"},
		"control in name": {Name: "a\x1bb", Command: "ls"},
		"NUL in command":  {Name: "x", Command: "ls\x00"},
		"huge command":    {Name: "x", Command: strings.Repeat("a", 20001)},
	} {
		if err := svc.Save("", s); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if got, _ := svc.List(); len(got) != 0 {
		t.Errorf("a rejected snippet was saved: %+v", got)
	}
}

func TestSnippetsRefuseAGarbledFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("RELAY_SSH_HOME", home)
	dir := filepath.Join(home, ".config", "relay-ssh")
	_ = os.MkdirAll(dir, 0o700)
	_ = os.WriteFile(filepath.Join(dir, "snippets.toml"), []byte("[[snippet\nname = "), 0o600)
	if _, err := (SnippetsService{}).List(); err == nil {
		t.Error("a malformed file should be an error, not an empty list (saving would overwrite it)")
	}
	if err := (SnippetsService{}).Save("", Snippet{Name: "x", Command: "ls"}); err == nil {
		t.Error("saving over a malformed file would destroy it")
	}
}
