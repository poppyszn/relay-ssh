// Package core holds the UI-free operations shared by the relay-ssh CLI and the
// Relay desktop app: profile lookup and removal, ssh_config import, backup and
// restore, and health checks. Nothing here prints or prompts.
package core

import (
	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/paths"
	"github.com/poppyszn/relay-ssh/internal/sshconfig"
)

// ResolveProfile finds a profile by id, "Folder/name" (what `relay-ssh ls --tree`
// prints), or a bare name that only one profile has.
func ResolveProfile(s config.Store, target string) (config.Profile, bool) {
	return s.Find(target)
}

// Regenerate writes the generated SSH config and ensures the managed
// Include block exists. Every operation that changes the store calls this
// immediately after saving, so `relay-ssh doctor` and the real
// ~/.ssh/relay-ssh.d/config never drift from connections.toml waiting on a
// separate, easy-to-forget `relay-ssh sync`.
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
	target, err := s.Resolve(name)
	if err != nil {
		return config.Profile{}, err
	}
	var kept []config.Profile
	for _, prof := range s.Profiles {
		if prof.ID != target.ID {
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

// SaveProfile adds a profile (key == "") or replaces the one key names (an id,
// "Folder/name" or unique name), keeping its id so jumps and saved passwords follow a
// rename or move. Two profiles may share a name in different folders. The store is
// validated before anything is written, so a rejected change leaves connections.toml
// untouched.
func SaveProfile(p paths.Paths, key string, prof config.Profile) error {
	s, err := config.Load(p.StoreFile)
	if err != nil {
		return err
	}
	if key == "" {
		prof.ID = ""
		s.Profiles = append(s.Profiles, prof)
	} else {
		old, err := s.Resolve(key)
		if err != nil {
			return err
		}
		prof.ID = old.ID
		for i := range s.Profiles {
			if s.Profiles[i].ID == old.ID {
				s.Profiles[i] = prof
			}
		}
		if old.Folder != prof.Folder {
			KeepFolder(&s, old.Folder)
		}
	}
	if err := config.Save(p.StoreFile, s); err != nil {
		return err
	}
	return Regenerate(p, s)
}
