package core

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// keyFilesToSkip are ~/.ssh entries that are never private keys.
var keyFilesToSkip = map[string]bool{
	"known_hosts":     true,
	"known_hosts.old": true,
	"config":          true,
	"authorized_keys": true,
	"environment":     true,
}

// FindKeyFiles lists files in dir that look like SSH private keys: not a
// directory, not a .pub file, not one of the well-known non-key filenames,
// and containing a "PRIVATE KEY" marker on its first line.
func FindKeyFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var keys []string
	for _, e := range entries {
		if e.IsDir() || strings.HasSuffix(e.Name(), ".pub") || keyFilesToSkip[e.Name()] {
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
