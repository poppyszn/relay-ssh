// Package core holds the UI-free operations shared by the sssh CLI and the
// Relay desktop app: profile lookup and removal, ssh_config import, backup and
// restore, and health checks. Nothing here prints or prompts.
package core

import (
	"fmt"
	"strings"

	"github.com/poppyszn/super-ssh-v2/internal/config"
	"github.com/poppyszn/super-ssh-v2/internal/paths"
	"github.com/poppyszn/super-ssh-v2/internal/sshconfig"
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
	if err := config.Save(p.StoreFile, s); err != nil {
		return config.Profile{}, err
	}
	return target, Regenerate(p, s)
}
