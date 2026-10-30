package cmd

import (
	"fmt"
	"strings"

	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/core"
	"github.com/poppyszn/relay-ssh/internal/launch"
	"github.com/poppyszn/relay-ssh/internal/paths"
	"github.com/poppyszn/relay-ssh/internal/style"
)

// Connect finds a profile and execs the SSH binary.
func Connect(p paths.Paths, name string) error {
	s, err := config.Load(p.StoreFile)
	if err != nil {
		return err
	}
	prof, ok := core.ResolveProfile(s, name)
	if !ok {
		return fmt.Errorf("profile %q not found", name)
	}
	if prof.IsRDP() {
		return fmt.Errorf("profile %q is an rdp connection; open it in the Relay desktop app", name)
	}
	if err := core.Regenerate(p, s); err != nil {
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
	removed, err := core.RemoveProfile(p, args[0])
	if err != nil {
		return err
	}
	fmt.Printf("%s Profile %s removed.\n", style.Success("✓"), style.Bold(removed.Name))
	return nil
}

// Sync regenerates the SSH config.
func Sync(p paths.Paths) error {
	s, err := config.Load(p.StoreFile)
	if err != nil {
		return err
	}
	if err := core.Regenerate(p, s); err != nil {
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
	prof, ok := core.ResolveProfile(s, profName)
	if !ok {
		return fmt.Errorf("profile %q not found", profName)
	}
	if prof.IsRDP() {
		return fmt.Errorf("profile %q is an rdp connection; scp is not supported for rdp", profName)
	}

	if err := core.Regenerate(p, s); err != nil {
		return err
	}

	// Build the scp path: profile:remotePath becomes host:remotePath for scp
	scpSrc := prof.Name + ":" + remotePath
	return launch.SCP(p, prof, scpSrc, localPath, false)
}
