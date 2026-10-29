// Package core holds the UI-free operations shared by the sssh CLI and the
// Relay desktop app: profile lookup and removal, ssh_config import, backup and
// restore, and health checks. Nothing here prints or prompts.
package core

import (
	"fmt"
	"strings"

	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/paths"
	"github.com/poppyszn/relay-ssh/internal/sshconfig"
)

// ResolveProfile finds a profile by name, also accepting the "folder/name"
// form `sssh ls --tree` prints. Profile names are already globally unique
// (Store.Validate enforces it) regardless of folder, so the folder segment
// is a display convention, not part of the stored identifier -- this just
// lets a name pasted straight out of `ls --tree` actually work.
func ResolveProfile(s config.Store, target string) (config.Profile, bool) {
	if prof, ok := s.Find(target); ok {
		return prof, true
	}
	if idx := strings.LastIndex(target, "/"); idx >= 0 {
		folder, name := target[:idx], target[idx+1:]
		if prof, ok := s.Find(name); ok && prof.Folder == folder {
			return prof, true
		}
	}
	return config.Profile{}, false
}

// Regenerate writes the generated SSH config and ensures the managed
// Include block exists. Every operation that changes the store calls this
// immediately after saving, so `sssh doctor` and the real
// ~/.ssh/sssh.d/config never drift from connections.toml waiting on a
// separate, easy-to-forget `sssh sync`.
func Regenerate(p paths.Paths, s config.Store) error {
	if err := sshconfig.Write(p, s); err != nil {
		return err
	}
	return sshconfig.EnsureInclude(p)
}

// RemoveProfile deletes the named profile, saves the store, and regenerates
// the SSH config. It returns the profile that was removed.
func RemoveProfile(p paths.Paths, name string) (config.Profile, error) {
	s, err := config.Load(p.StoreFile)
	if err != nil {
		return config.Profile{}, err
	}
	target, ok := ResolveProfile(s, name)
	if !ok {
		return config.Profile{}, fmt.Errorf("profile %q not found", name)
	}
	var kept []config.Profile
	for _, prof := range s.Profiles {
		if prof.Name != target.Name {
			kept = append(kept, prof)
		}
	}
	s.Profiles = kept
	KeepFolder(&s, target.Folder)
	if err := config.Save(p.StoreFile, s); err != nil {
		return config.Profile{}, err
	}
	return target, Regenerate(p, s)
}

// SaveProfile adds a profile (oldName == "") or replaces the one called
// oldName, renaming it if prof.Name differs. A rename also repoints every other
// profile's Jump at the new name. The store is validated before anything is
// written, so a rejected change leaves connections.toml untouched.
func SaveProfile(p paths.Paths, oldName string, prof config.Profile) error {
	s, err := config.Load(p.StoreFile)
	if err != nil {
		return err
	}
	if oldName == "" {
		if _, taken := s.Find(prof.Name); taken {
			return fmt.Errorf("profile %q already exists", prof.Name)
		}
		s.Profiles = append(s.Profiles, prof)
	} else {
		if _, ok := s.Find(oldName); !ok {
			return fmt.Errorf("profile %q not found", oldName)
		}
		if _, taken := s.Find(prof.Name); taken && prof.Name != oldName {
			return fmt.Errorf("profile %q already exists", prof.Name)
		}
		profiles := make([]config.Profile, len(s.Profiles))
		leftFolder := ""
		for i, cur := range s.Profiles {
			switch {
			case cur.Name == oldName:
				profiles[i] = prof
				leftFolder = cur.Folder
			case cur.Jump == oldName:
				cur.Jump = prof.Name
				profiles[i] = cur
			default:
				profiles[i] = cur
			}
		}
		s.Profiles = profiles
		if leftFolder != prof.Folder {
			KeepFolder(&s, leftFolder)
		}
	}
	if err := config.Save(p.StoreFile, s); err != nil {
		return err
	}
	return Regenerate(p, s)
}
