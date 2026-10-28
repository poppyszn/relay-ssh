package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/poppyszn/super-ssh-v2/internal/core"
	"github.com/poppyszn/super-ssh-v2/internal/paths"
)

// ImportCmd parses ~/.ssh/config and creates profiles from Host entries.
func ImportCmd(p paths.Paths, args []string) error {
	sshConfigPath := filepath.Join(os.Getenv("HOME"), ".ssh", "config")
	file, err := os.Open(sshConfigPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", sshConfigPath, err)
	}
	defer file.Close()

	added, err := core.ImportSSHConfig(p, file)
	if err != nil {
		return err
	}
	if added == 0 {
		fmt.Println("No new profiles found.")
		return nil
	}
	fmt.Printf("Imported %d profile(s).\n", added)
	return nil
}
