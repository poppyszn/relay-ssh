package cmd

import (
	"fmt"
	"strings"

	"github.com/poppyszn/super-ssh-v2/internal/config"
	"github.com/poppyszn/super-ssh-v2/internal/launch"
	"github.com/poppyszn/super-ssh-v2/internal/paths"
	"github.com/poppyszn/super-ssh-v2/internal/sshconfig"
	"github.com/poppyszn/super-ssh-v2/internal/style"
)

// resolveProfile finds a profile by name, also accepting the "folder/name"
// form `sssh ls --tree` prints. Profile names are already globally unique
// (Store.Validate enforces it) regardless of folder, so the folder segment
// is a display convention, not part of the stored identifier -- this just
// lets a name pasted straight out of `ls --tree` actually work.
func resolveProfile(s config.Store, target string) (config.Profile, bool) {
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

// regenerateConfig writes the generated SSH config and ensures the managed
// Include block exists. Every command that changes the store calls this
// immediately after saving, so `sssh doctor` and the real
// ~/.ssh/sssh.d/config never drift from connections.toml waiting on a
// separate, easy-to-forget `sssh sync`.
func regenerateConfig(p paths.Paths, s config.Store) error {
	if err := sshconfig.Write(p, s); err != nil {
		return err
	}
	return sshconfig.EnsureInclude(p)
}

// Connect finds a profile and execs the SSH binary.
func Connect(p paths.Paths, name string) error {
	s, err := config.Load(p.StoreFile)
	if err != nil {
		return err
	}
	prof, ok := resolveProfile(s, name)
	if !ok {
		return fmt.Errorf("profile %q not found", name)
	}
	if err := regenerateConfig(p, s); err != nil {
		return err
	}
	fmt.Printf("%s %s (%s@%s)\n", style.Dim("→ connecting to"), style.Bold(prof.Name), prof.User, prof.Host)
	return launch.SSH(p, prof)
}

// Rm removes a profile.
func Rm(p paths.Paths, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("rm requires a profile name")
	}
	s, err := config.Load(p.StoreFile)
	if err != nil {
		return err
	}
	target, ok := resolveProfile(s, args[0])
	if !ok {
		return fmt.Errorf("profile %q not found", args[0])
	}
	var filtered []config.Profile
	for _, prof := range s.Profiles {
		if prof.Name != target.Name {
			filtered = append(filtered, prof)
		}
	}
	s.Profiles = filtered
	if err := config.Save(p.StoreFile, s); err != nil {
		return err
	}
	if err := regenerateConfig(p, s); err != nil {
		return err
	}
	fmt.Printf("%s Profile %s removed.\n", style.Success("✓"), style.Bold(target.Name))
	return nil
}

// Sync regenerates the SSH config.
func Sync(p paths.Paths) error {
	s, err := config.Load(p.StoreFile)
	if err != nil {
		return err
	}
	if err := regenerateConfig(p, s); err != nil {
		return err
	}
	fmt.Printf("%s SSH config synchronized.\n", style.Success("✓"))
	return nil
}

// CpCmd copies files via scp using a profile.
func CpCmd(p paths.Paths, args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("cp requires: profile:src local-dst")
	}
	parts := strings.Split(args[0], ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid format: use profile:remote-path")
	}
	profName, remotePath := parts[0], parts[1]
	localPath := args[1]

	s, err := config.Load(p.StoreFile)
	if err != nil {
		return err
	}
	prof, ok := resolveProfile(s, profName)
	if !ok {
		return fmt.Errorf("profile %q not found", profName)
	}

	if err := regenerateConfig(p, s); err != nil {
		return err
	}

	// Build the scp path: profile:remotePath becomes host:remotePath for scp
	scpSrc := prof.Name + ":" + remotePath
	return launch.SCP(p, prof, scpSrc, localPath, false)
}
