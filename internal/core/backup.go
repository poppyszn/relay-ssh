package core

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"filippo.io/age"
	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/fsutil"
	"github.com/poppyszn/relay-ssh/internal/paths"
)

// restoreEntryLimit caps how much of one archive entry Restore will read (a variable so tests can lower it).
var restoreEntryLimit int64 = 64 << 20

// ErrIncorrectPassphrase is returned by Restore when decryption or
// decompression fails, which in practice means a wrong passphrase.
var ErrIncorrectPassphrase = errors.New("incorrect passphrase (or the file is corrupted)")

// Backup writes connections.toml and the keys directory to w as a gzip tarball
// encrypted with an age scrypt recipient derived from passphrase.
func Backup(p paths.Paths, w io.Writer, passphrase string) error {
	recipient, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		return fmt.Errorf("creating encryption recipient: %w", err)
	}
	ageWriter, err := age.Encrypt(w, recipient)
	if err != nil {
		return fmt.Errorf("initializing encryption: %w", err)
	}

	gz := gzip.NewWriter(ageWriter)
	tw := tar.NewWriter(gz)

	if err := AddFileToTar(tw, p.StoreFile, "connections.toml"); err != nil {
		return err
	}
	if err := addDirToTar(tw, p.KeysDir, "keys"); err != nil && !os.IsNotExist(err) {
		return err
	}

	if err := tw.Close(); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	return ageWriter.Close()
}

// AddFileToTar appends the file at srcPath to tw under tarName.
func AddFileToTar(tw *tar.Writer, srcPath, tarName string) error {
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
		if err := AddFileToTar(tw, srcPath, tarPath); err != nil {
			return err
		}
	}
	return nil
}

// Restore unpacks a backup read from r. When encrypted is true the stream is
// first decrypted with passphrase; otherwise it is a plain gzip tarball
// (legacy backups). The SSH config is regenerated afterwards.
func Restore(p paths.Paths, r io.Reader, passphrase string, encrypted bool) error {
	tarStream := r
	if encrypted {
		identity, err := age.NewScryptIdentity(passphrase)
		if err != nil {
			return ErrIncorrectPassphrase
		}
		ageReader, err := age.Decrypt(r, identity)
		if err != nil {
			return ErrIncorrectPassphrase
		}
		// Read entire decrypted data into memory buffer to verify integrity and scrypt MAC
		decrypted, err := io.ReadAll(ageReader)
		if err != nil {
			return ErrIncorrectPassphrase
		}
		tarStream = bytes.NewReader(decrypted)
	}

	gz, err := gzip.NewReader(tarStream)
	if err != nil {
		if encrypted {
			return ErrIncorrectPassphrase
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
			if encrypted {
				return ErrIncorrectPassphrase
			}
			return err
		}

		if header.Name == "connections.toml" {
			data, err := readEntry(tr)
			if err != nil {
				return err
			}
			// The archive is untrusted until proven otherwise: never install connections that would not load.
			if _, err := config.Parse(data); err != nil {
				return fmt.Errorf("the backup's connections are not valid: %w", err)
			}
			if err := fsutil.WriteFile(p.StoreFile, data, 0o600); err != nil {
				return fmt.Errorf("restoring connections.toml: %w", err)
			}
		} else if filepath.Dir(header.Name) == "keys" {
			if err := os.MkdirAll(p.KeysDir, 0o700); err != nil {
				return err
			}
			keyPath := filepath.Join(p.KeysDir, filepath.Base(header.Name))
			data, err := readEntry(tr)
			if err != nil {
				return err
			}
			if err := fsutil.WriteFile(keyPath, data, 0o600); err != nil {
				return err
			}
		}
	}

	if s, err := config.Load(p.StoreFile); err == nil {
		return Regenerate(p, s)
	}
	return nil
}

// readEntry reads one archive entry, refusing one that expands past restoreEntryLimit.
func readEntry(r io.Reader) ([]byte, error) {
	data, err := io.ReadAll(io.LimitReader(r, restoreEntryLimit+1))
	if err == nil && int64(len(data)) > restoreEntryLimit {
		err = fmt.Errorf("an entry in the backup is larger than %d MB: refusing to read it", restoreEntryLimit>>20)
	}
	return data, err
}
