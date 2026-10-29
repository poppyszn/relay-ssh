package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/poppyszn/super-ssh-v2/internal/config"
	"github.com/poppyszn/super-ssh-v2/internal/core"
	"github.com/poppyszn/super-ssh-v2/internal/paths"
)

// Folder operations for the sidebar. A folder is a path ("prod/eu"); connections
// name theirs, and the store's explicit folders list holds the empty ones.

func mutate(fn func(*config.Store) error) error {
	p, err := setup()
	if err != nil {
		return err
	}
	return mutateAt(p, fn)
}

func mutateAt(p paths.Paths, fn func(*config.Store) error) error {
	s, err := config.Load(p.StoreFile)
	if err != nil {
		return err
	}
	if err := fn(&s); err != nil {
		return err
	}
	return config.Save(p.StoreFile, s)
}

// allFolders is every folder, sorted: the explicit ones, the ones connections
// live in, and every ancestor of those, so "a/b" implies "a".
func allFolders(s config.Store) []string {
	set := map[string]bool{}
	add := func(path string) {
		for path != "" {
			set[path] = true
			i := strings.LastIndex(path, "/")
			if i < 0 {
				break
			}
			path = path[:i]
		}
	}
	for _, f := range s.Folders {
		add(f)
	}
	for _, pr := range s.Profiles {
		add(pr.Folder)
	}
	out := make([]string, 0, len(set))
	for f := range set {
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

func hasFolder(s config.Store, path string) bool {
	for _, f := range allFolders(s) {
		if f == path {
			return true
		}
	}
	return false
}

func validNewFolder(path string) error {
	if path == "" {
		return fmt.Errorf("a folder needs a name")
	}
	return config.ValidateFolder(path)
}

// Folders returns every folder, including empty ones and ancestors.
func (ProfilesService) Folders() ([]string, error) {
	p, err := paths.Default()
	if err != nil {
		return nil, err
	}
	s, err := config.Load(p.StoreFile)
	if err != nil {
		return nil, err
	}
	return allFolders(s), nil
}

// CreateFolder adds an empty folder, or subfolder when path contains slashes.
func (ProfilesService) CreateFolder(path string) error {
	if err := validNewFolder(path); err != nil {
		return err
	}
	return mutate(func(s *config.Store) error {
		if hasFolder(*s, path) {
			return fmt.Errorf("folder %q already exists", path)
		}
		s.Folders = append(s.Folders, path)
		return nil
	})
}

// MoveProfile puts a connection in a folder ("" for the top level).
func (ProfilesService) MoveProfile(name, folder string) error {
	if err := config.ValidateFolder(folder); err != nil {
		return err
	}
	return mutate(func(s *config.Store) error {
		for i := range s.Profiles {
			if s.Profiles[i].Name != name {
				continue
			}
			old := s.Profiles[i].Folder
			s.Profiles[i].Folder = folder
			if old != folder {
				core.KeepFolder(s, old)
			}
			return nil
		}
		return fmt.Errorf("profile %q not found", name)
	})
}

// RenameFolder renames a folder or moves it elsewhere (a move is a rename to a
// new path). Every subfolder and connection beneath it follows.
func (ProfilesService) RenameFolder(from, to string) error {
	if err := validNewFolder(from); err != nil {
		return err
	}
	if err := validNewFolder(to); err != nil {
		return err
	}
	return mutate(func(s *config.Store) error {
		if !hasFolder(*s, from) {
			return fmt.Errorf("folder %q not found", from)
		}
		if core.FolderWithin(to, from) {
			return fmt.Errorf("cannot move folder %q into itself", from)
		}
		if hasFolder(*s, to) {
			return fmt.Errorf("folder %q already exists", to)
		}
		rebase := func(path string) string {
			if core.FolderWithin(path, from) {
				return to + path[len(from):]
			}
			return path
		}
		for i := range s.Folders {
			s.Folders[i] = rebase(s.Folders[i])
		}
		for i := range s.Profiles {
			s.Profiles[i].Folder = rebase(s.Profiles[i].Folder)
		}
		s.Folders = append(s.Folders, to) // the folder itself stays even if only a connection implied it
		return nil
	})
}

// DeleteFolder removes a folder and its empty subfolders. A folder that holds
// connections, directly or in a subfolder, is refused and they are named.
func (ProfilesService) DeleteFolder(path string) error {
	return mutate(func(s *config.Store) error {
		if !hasFolder(*s, path) {
			return fmt.Errorf("folder %q not found", path)
		}
		var inside []string
		for _, pr := range s.Profiles {
			if core.FolderWithin(pr.Folder, path) {
				inside = append(inside, pr.Name)
			}
		}
		if len(inside) > 0 {
			return fmt.Errorf("folder %q still holds %s; move them out first", path, strings.Join(inside, ", "))
		}
		kept := s.Folders[:0]
		for _, f := range s.Folders {
			if !core.FolderWithin(f, path) {
				kept = append(kept, f)
			}
		}
		s.Folders = kept
		return nil
	})
}
