package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/poppyszn/super-ssh-v2/internal/style"
)

const manualEntryOption = "Type a path manually..."

// filesToSkip are ~/.ssh entries that are never private keys.
var filesToSkip = map[string]bool{
	"known_hosts":     true,
	"known_hosts.old": true,
	"config":          true,
	"authorized_keys": true,
	"environment":     true,
}

// PickKeyFile browses ~/.ssh for private key files and lets the user choose
// one, with fzf if available, else a numbered menu. If none are found, or the
// user picks "type a path manually", it falls back to a plain prompt.
func PickKeyFile(in *bufio.Scanner) (string, error) {
	candidates := findKeyFiles(sshHomeDir())
	if len(candidates) == 0 {
		val, ok := scanLine(in, "Key file or name")
		if !ok {
			return "", errCancelled
		}
		return val, nil
	}

	options := append(append([]string{}, candidates...), manualEntryOption)
	fmt.Println(style.Header("Select a key:"))
	choice, err := tryFzf(options)
	if err != nil {
		choice, err = menuPick(in, options)
		if err != nil {
			return "", err
		}
	}
	if choice == manualEntryOption {
		val, ok := scanLine(in, "Key file or name")
		if !ok {
			return "", errCancelled
		}
		return val, nil
	}
	return choice, nil
}

// findKeyFiles lists files in dir that look like SSH private keys: not a
// directory, not a .pub file, not one of the well-known non-key filenames,
// and containing a "PRIVATE KEY" marker on its first line.
func findKeyFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var keys []string
	for _, e := range entries {
		if e.IsDir() || strings.HasSuffix(e.Name(), ".pub") || filesToSkip[e.Name()] {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if looksLikePrivateKey(path) {
			keys = append(keys, path)
		}
	}
	sort.Strings(keys)
	return keys
}

// looksLikePrivateKey reads the first line of a file and checks for the
// "PRIVATE KEY" marker common to OpenSSH, RSA, DSA, and EC key formats.
func looksLikePrivateKey(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 64)
	n, _ := f.Read(buf)
	return strings.Contains(string(buf[:n]), "PRIVATE KEY")
}

func sshHomeDir() string {
	return filepath.Join(os.Getenv("HOME"), ".ssh")
}
