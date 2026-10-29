package cmd

import (
	"bufio"
	"strings"
	"testing"

	"github.com/poppyszn/relay-ssh/internal/config"
)

func storeWithFolders() config.Store {
	return config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "web1", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent, Folder: "prod"},
		{Name: "web2", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent, Folder: "prod"},
		{Name: "db", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent, Folder: "staging"},
		{Name: "local", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent},
	}}
}

func TestExistingFoldersSortedAndDeduplicated(t *testing.T) {
	got := existingFolders(storeWithFolders())
	if len(got) != 2 || got[0] != "prod" || got[1] != "staging" {
		t.Errorf("existingFolders = %v, want [prod staging]", got)
	}
}

func TestExistingFoldersEmptyStore(t *testing.T) {
	got := existingFolders(config.Store{Version: config.Version})
	if len(got) != 0 {
		t.Errorf("existingFolders on empty store = %v, want none", got)
	}
}

func TestPickFolderNoFolderOption(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	s := storeWithFolders()
	// Menu order: (no folder), prod, staging, + Create a new folder...
	// Option 1 is "(no folder)".
	in := bufio.NewScanner(strings.NewReader("1\n"))
	got, err := PickFolder(in, s)
	if err != nil {
		t.Fatalf("PickFolder: %v", err)
	}
	if got != "" {
		t.Errorf("PickFolder (no folder) = %q, want empty", got)
	}
}

func TestPickFolderSelectsExisting(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	s := storeWithFolders()
	// Option 2 is "prod" (first existing folder, alphabetically).
	in := bufio.NewScanner(strings.NewReader("2\n"))
	got, err := PickFolder(in, s)
	if err != nil {
		t.Fatalf("PickFolder: %v", err)
	}
	if got != "prod" {
		t.Errorf("PickFolder existing = %q, want prod", got)
	}
}

func TestPickFolderCreateNew(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	s := storeWithFolders()
	// Options: 1=(no folder) 2=prod 3=staging 4=+ Create a new folder...
	in := bufio.NewScanner(strings.NewReader("4\nnewteam\n"))
	got, err := PickFolder(in, s)
	if err != nil {
		t.Fatalf("PickFolder: %v", err)
	}
	if got != "newteam" {
		t.Errorf("PickFolder create-new = %q, want newteam", got)
	}
}

func TestPickFolderCreateNewRetriesOnInvalidName(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	s := storeWithFolders()
	// First attempt "/bad" is invalid (leading slash), retry with "good".
	in := bufio.NewScanner(strings.NewReader("4\n/bad\ngood\n"))
	got, err := PickFolder(in, s)
	if err != nil {
		t.Fatalf("PickFolder: %v", err)
	}
	if got != "good" {
		t.Errorf("PickFolder after retry = %q, want good", got)
	}
}

func TestPickFolderCreateNewEmptyMeansNoFolder(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	s := storeWithFolders()
	in := bufio.NewScanner(strings.NewReader("4\n\n"))
	got, err := PickFolder(in, s)
	if err != nil {
		t.Fatalf("PickFolder: %v", err)
	}
	if got != "" {
		t.Errorf("PickFolder blank new-folder name = %q, want empty (no folder)", got)
	}
}
