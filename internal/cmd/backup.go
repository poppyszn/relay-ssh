package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/poppyszn/super-ssh-v2/internal/core"
	"github.com/poppyszn/super-ssh-v2/internal/paths"
	"github.com/poppyszn/super-ssh-v2/internal/style"
)

// BackupCmd creates an age-encrypted gzip tarball of connections.toml and keys.
func BackupCmd(p paths.Paths, args []string) error {
	outFile := "sssh-backup.tar.gz.age"
	if len(args) > 0 && args[0] == "-o" && len(args) > 1 {
		outFile = args[1]
		if !strings.HasSuffix(outFile, ".age") {
			outFile += ".age"
		}
	}

	passphrase := os.Getenv("SSSH_BACKUP_PASSPHRASE")
	if passphrase == "" {
		in := bufio.NewScanner(os.Stdin)
		p, err := promptBackupPassphrase(in)
		if err != nil {
			return err
		}
		passphrase = p
	}

	f, err := os.Create(outFile)
	if err != nil {
		return fmt.Errorf("creating backup file: %w", err)
	}
	defer f.Close()

	if err := core.Backup(p, f, passphrase); err != nil {
		return err
	}
	fmt.Printf("%s Backup created: %s\n", style.Success("✓"), style.Bold(outFile))
	return nil
}

func promptBackupPassphrase(in *bufio.Scanner) (string, error) {
	for {
		p1, ok := scanHiddenLine(in, "Enter backup passphrase")
		if !ok {
			return "", errCancelled
		}
		if p1 == "" {
			fmt.Println(style.Fail("✗ passphrase cannot be empty"))
			continue
		}
		p2, ok := scanHiddenLine(in, "Confirm passphrase")
		if !ok {
			return "", errCancelled
		}
		if p1 != p2 {
			fmt.Println(style.Fail("✗ passphrases do not match, try again"))
			continue
		}
		return p1, nil
	}
}

// RestoreCmd extracts a backup archive, decrypting if encrypted with age.
func RestoreCmd(p paths.Paths, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("restore requires a backup file")
	}
	backupPath := args[0]

	f, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("opening backup: %w", err)
	}
	defer f.Close()

	encrypted := strings.HasSuffix(backupPath, ".age")
	passphrase := ""
	if encrypted {
		passphrase = os.Getenv("SSSH_BACKUP_PASSPHRASE")
		if passphrase == "" {
			in := bufio.NewScanner(os.Stdin)
			val, ok := scanHiddenLine(in, "Enter backup passphrase")
			if !ok {
				return errCancelled
			}
			passphrase = val
		}
	}

	if err := core.Restore(p, f, passphrase, encrypted); err != nil {
		return err
	}
	fmt.Printf("%s Backup restored.\n", style.Success("✓"))
	return nil
}
