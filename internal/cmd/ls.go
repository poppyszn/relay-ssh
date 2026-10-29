// Package cmd implements the CLI commands.
package cmd

import (
	"fmt"
	"sort"

	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/paths"
	"github.com/poppyszn/relay-ssh/internal/style"
)

// Ls lists all profiles, optionally in tree view.
//
// Tree view shows connections grouped by folder: folder/name style.
// Flat view (the default) shows one per line, sorted, including the full path.
func Ls(p paths.Paths, args []string) error {
	tree := false
	var filterTag string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--tree":
			tree = true
		case "--tag":
			if i+1 < len(args) {
				filterTag = args[i+1]
				i++
			}
		}
	}
	s, err := config.Load(p.StoreFile)
	if err != nil {
		return err
	}
	if filterTag != "" {
		var filtered []config.Profile
		for _, prof := range s.Profiles {
			if containsTag(prof.Tags, filterTag) {
				filtered = append(filtered, prof)
			}
		}
		s.Profiles = filtered
	}
	if len(s.Profiles) == 0 {
		fmt.Println(style.Dim("(no profiles)"))
		return nil
	}
	if tree {
		return lsTree(s)
	}
	return lsFlat(s)
}

func containsTag(tags []string, target string) bool {
	for _, t := range tags {
		if t == target {
			return true
		}
	}
	return false
}

func lsFlat(s config.Store) error {
	for _, name := range s.Names() {
		p, _ := s.Find(name)
		if p.Folder == "" {
			fmt.Println(name)
		} else {
			fmt.Printf("%s%s%s\n", style.Accent(p.Folder), style.Dim("/"), name)
		}
	}
	return nil
}

// lsTree groups profiles by folder and renders each group with box-drawing
// connectors. Folder names are sorted explicitly: map iteration order is
// randomized in Go, and a tree view that reorders itself between runs would
// undercut the whole point of an "organized" view.
func lsTree(s config.Store) error {
	groups := make(map[string][]string)
	var root, folders []string
	for _, name := range s.Names() {
		p, _ := s.Find(name)
		if p.Folder == "" {
			root = append(root, name)
			continue
		}
		if _, seen := groups[p.Folder]; !seen {
			folders = append(folders, p.Folder)
		}
		groups[p.Folder] = append(groups[p.Folder], name)
	}
	sort.Strings(folders)

	for _, name := range root {
		fmt.Println(name)
	}
	for _, folder := range folders {
		fmt.Println(style.Accent(folder))
		members := groups[folder]
		for i, name := range members {
			connector := "├─ "
			if i == len(members)-1 {
				connector = "└─ "
			}
			fmt.Printf("  %s%s\n", style.Dim(connector), name)
		}
	}
	return nil
}
