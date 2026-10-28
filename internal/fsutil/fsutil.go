// Package fsutil provides the atomic file write used for every piece of state
// whose partial write would lock the user out of their own hosts.
package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteFile writes data to path atomically.
//
// It creates a temp file in the target's own directory -- not /tmp, so the
// final step is a same-filesystem rename rather than a copy -- sets the
// requested mode, fsyncs, and renames over the target. A crash at any point
// leaves either the old file or the new one, never a truncated mixture.
func WriteFile(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("creating temp file in %s: %w", dir, err)
	}
	tmp := f.Name()
	defer os.Remove(tmp) // no-op once the rename has succeeded

	if err := f.Chmod(mode); err != nil {
		f.Close()
		return fmt.Errorf("setting permissions on %s: %w", tmp, err)
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return fmt.Errorf("writing %s: %w", tmp, err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return fmt.Errorf("syncing %s: %w", tmp, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("closing %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("renaming %s to %s: %w", tmp, path, err)
	}
	return nil
}
