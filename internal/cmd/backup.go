package cmd

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"filippo.io/age"
	"github.com/poppyszn/super-ssh-v2/internal/config"
	"github.com/poppyszn/super-ssh-v2/internal/fsutil"
	"github.com/poppyszn/super-ssh-v2/internal/paths"
	"github.com/poppyszn/super-ssh-v2/internal/style"
)

var errIncorrectPassphrase = errors.New("incorrect passphrase (or the file is corrupted)")

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

	recipient, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		return fmt.Errorf("creating encryption recipient: %w", err)
	}

	f, err := os.Create(outFile)
	if err != nil {
		return fmt.Errorf("creating backup file: %w", err)
	}
	defer f.Close()

	ageWriter, err := age.Encrypt(f, recipient)
	if err != nil {
		return fmt.Errorf("initializing encryption: %w", err)
	}

	gz := gzip.NewWriter(ageWriter)
	tw := tar.NewWriter(gz)

	// Add connections.toml
	if err := addFileToTar(tw, p.StoreFile, "connections.toml"); err != nil {
		return err
	}

	// Add keys directory
	if err := addDirToTar(tw, p.KeysDir, "keys"); err != nil && !os.IsNotExist(err) {
		return err
	}

	if err := tw.Close(); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	if err := ageWriter.Close(); err != nil {
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

func addFileToTar(tw *tar.Writer, srcPath, tarName string) error {
	file, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", srcPath, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}

	header := &tar.Header{
		Name:    tarName,
		Size:    info.Size(),
		Mode:    int64(info.Mode()),
		ModTime: time.Now(),
	}
	if err := tw.WriteHeader(header); err != nil {
		return err
	}
	_, err = io.Copy(tw, file)
	return err
}

func addDirToTar(tw *tar.Writer, srcDir, tarPrefix string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		srcPath := filepath.Join(srcDir, entry.Name())
		tarPath := filepath.Join(tarPrefix, entry.Name())
		if err := addFileToTar(tw, srcPath, tarPath); err != nil {
			return err
		}
	}
	return nil
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

	var tarStream io.Reader = f
	if strings.HasSuffix(backupPath, ".age") {
		passphrase := os.Getenv("SSSH_BACKUP_PASSPHRASE")
		if passphrase == "" {
			in := bufio.NewScanner(os.Stdin)
			val, ok := scanHiddenLine(in, "Enter backup passphrase")
			if !ok {
				return errCancelled
			}
			passphrase = val
		}

		identity, err := age.NewScryptIdentity(passphrase)
		if err != nil {
			return errIncorrectPassphrase
		}
		ageReader, err := age.Decrypt(f, identity)
		if err != nil {
			return errIncorrectPassphrase
		}
		// Read entire decrypted data into memory buffer to verify integrity and scrypt MAC
		decrypted, err := io.ReadAll(ageReader)
		if err != nil {
			return errIncorrectPassphrase
		}
		tarStream = bytes.NewReader(decrypted)
	}

	gz, err := gzip.NewReader(tarStream)
	if err != nil {
		if strings.HasSuffix(backupPath, ".age") {
			return errIncorrectPassphrase
		}
		return fmt.Errorf("decompressing backup: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			if strings.HasSuffix(backupPath, ".age") {
				return errIncorrectPassphrase
			}
			return err
		}

		if header.Name == "connections.toml" {
			data, err := io.ReadAll(tr)
			if err != nil {
				return err
			}
			if err := fsutil.WriteFile(p.StoreFile, data, 0o600); err != nil {
				return fmt.Errorf("restoring connections.toml: %w", err)
			}
		} else if filepath.Dir(header.Name) == "keys" {
			if err := os.MkdirAll(p.KeysDir, 0o700); err != nil {
				return err
			}
			keyPath := filepath.Join(p.KeysDir, filepath.Base(header.Name))
			data, err := io.ReadAll(tr)
			if err != nil {
				return err
			}
			if err := fsutil.WriteFile(keyPath, data, 0o600); err != nil {
				return err
			}
		}
	}

	if s, err := config.Load(p.StoreFile); err == nil {
		if err := regenerateConfig(p, s); err != nil {
			return err
		}
	}
	fmt.Printf("%s Backup restored.\n", style.Success("✓"))
	return nil
}
