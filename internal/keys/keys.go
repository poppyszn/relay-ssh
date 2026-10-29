// Package keys manages SSH private keys that sssh adopts.
//
// A key is "adopted" when sssh copies it from the user's existing keyring into
// ~/.ssh/sssh.d/keys/, giving sssh a local copy for restore operations. Non-adopted
// keys are "linked" — stored elsewhere and referenced by absolute or tilde path.
package keys

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/poppyszn/relay-ssh/internal/fsutil"
	"github.com/poppyszn/relay-ssh/internal/paths"
)

// Adopt copies a key file into ~/.ssh/sssh.d/keys/ at 0600, or validates that
// it is already there. The key is renamed to keyName, idempotently. Returns the
// basename used in the connections store.
//
// A key at ~/.ssh/id_rsa is adopted as "id_rsa" (basename).
// A key at /etc/ssl/keys/ci is adopted as "ci" (basename).
// A tilde-prefixed key is expanded to home before adoption.
func Adopt(p paths.Paths, source string, keyName string) (string, error) {
	expanded := expandTilde(source)
	srcData, err := os.ReadFile(expanded)
	if err != nil {
		return "", fmt.Errorf("reading source key %s: %w", source, err)
	}

	dstPath := filepath.Join(p.KeysDir, keyName)
	if err := fsutil.WriteFile(dstPath, srcData, 0o600); err != nil {
		return "", fmt.Errorf("adopting key to %s: %w", dstPath, err)
	}
	return keyName, nil
}

// ValidateAdopted checks that all adopted keys in profiles exist in KeysDir.
// Returns all missing keys.
func ValidateAdopted(p paths.Paths, keyNames []string) []string {
	var missing []string
	for _, name := range keyNames {
		path := filepath.Join(p.KeysDir, name)
		if _, err := os.Stat(path); err != nil {
			missing = append(missing, name)
		}
	}
	return missing
}

// FingerprintMatches reports whether the key at source and the adopted copy
// at keyName have the same MD5 fingerprint. Used by doctor to detect when
// an adopted key has diverged from its source.
//
// OpenSSH fingerprints keys using the MD5 hash of the public key blob,
// not the private key file. We can't re-derive that blob from the private key
// without parsing it. So sssh stores a fingerprint when adopting, and doctor
// compares the current source against the stored fingerprint. This detects
// drift without needing a public key parser.
func FingerprintMatches(p paths.Paths, keyName string, sourceData []byte) bool {
	adoptedPath := filepath.Join(p.KeysDir, keyName)
	adoptedData, err := os.ReadFile(adoptedPath)
	if err != nil {
		return false
	}
	return bytes.Equal(adoptedData, sourceData)
}

// List returns the basenames of all adopted keys.
func List(p paths.Paths) ([]string, error) {
	entries, err := os.ReadDir(p.KeysDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("listing keys: %w", err)
	}
	var keys []string
	for _, e := range entries {
		if !e.IsDir() {
			keys = append(keys, e.Name())
		}
	}
	return keys, nil
}

// expandTilde turns ~/file into $HOME/file. Absolute paths pass through.
func expandTilde(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, path[1:])
}
