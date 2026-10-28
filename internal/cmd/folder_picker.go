package cmd

import (
	"bufio"
	"fmt"
	"sort"

	"github.com/poppyszn/super-ssh-v2/internal/config"
	"github.com/poppyszn/super-ssh-v2/internal/style"
)

const (
	noFolderOption     = "(no folder)"
	createFolderOption = "+ Create a new folder..."
)

// PickFolder offers the folders already in use across the store, plus options
// for no folder and creating a new one. Browsing existing folders means a
// profile lands in "prod" and not "Prod" or "prod " by a typo -- the whole
// point of an organized view falls apart if two folders differ only by a
// stray character.
func PickFolder(in *bufio.Scanner, s config.Store) (string, error) {
	folders := existingFolders(s)
	options := append([]string{noFolderOption}, folders...)
	options = append(options, createFolderOption)

	fmt.Println(style.Header("Folder:"))
	choice, err := tryFzf(options)
	if err != nil {
		choice, err = menuPick(in, options)
		if err != nil {
			return "", err
		}
	}

	switch choice {
	case noFolderOption:
		return "", nil
	case createFolderOption:
		return promptNewFolder(in)
	default:
		return choice, nil
	}
}

// promptNewFolder loops until it gets a folder name that passes validation,
// or an empty line, which means "changed my mind, no folder."
func promptNewFolder(in *bufio.Scanner) (string, error) {
	for {
		name := prompt(in, "New folder name")
		if name == "" {
			return "", nil
		}
		if err := config.ValidateFolder(name); err != nil {
			fmt.Println(style.Fail("✗ " + err.Error()))
			continue
		}
		return name, nil
	}
}

// existingFolders returns the sorted, deduplicated set of folders already
// used by profiles in the store.
func existingFolders(s config.Store) []string {
	seen := make(map[string]bool)
	var folders []string
	for _, p := range s.Profiles {
		if p.Folder != "" && !seen[p.Folder] {
			seen[p.Folder] = true
			folders = append(folders, p.Folder)
		}
	}
	sort.Strings(folders)
	return folders
}
