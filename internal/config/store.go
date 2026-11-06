package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"sort"

	"github.com/BurntSushi/toml"

	"github.com/poppyszn/relay-ssh/internal/fsutil"
)

// Load reads connections.toml.
//
// A missing file yields an empty store rather than an error, so a first run
// needs no bootstrap step. Anything else -- unreadable, unparseable, or
// semantically invalid -- is reported with the path in the message, because the
// user is expected to be able to open this file and fix it by hand.
func Load(path string) (Store, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return Store{Version: Version}, nil
	}
	if err != nil {
		return Store{}, fmt.Errorf("reading %s: %w", path, err)
	}
	s, err := Parse(data)
	if err != nil {
		return Store{}, fmt.Errorf("%s: %w", path, err)
	}
	return s, nil
}

// Parse decodes and validates connections.toml text that has not been trusted yet.
func Parse(data []byte) (Store, error) {
	var s Store
	if _, err := toml.Decode(string(data), &s); err != nil {
		return Store{}, fmt.Errorf("parsing: %w", err)
	}
	s.Normalize(false)
	return s, s.Validate()
}

// Save validates, sorts, and writes the store atomically at 0600.
//
// Sorting keeps the file's diff stable, so a private git repo of
// connections.toml shows real changes rather than reordering noise. Validation
// happens before any bytes are written: an invalid store must never reach disk,
// because Load would then refuse to read it back.
func Save(path string, s Store) error {
	s.Version = Version
	s.Profiles = slices.Clone(s.Profiles)
	s.Normalize(true)
	sorted := make([]Profile, len(s.Profiles))
	copy(sorted, s.Profiles)
	sort.Slice(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		if a.Folder != b.Folder {
			return a.Folder < b.Folder
		}
		return a.ID < b.ID
	})
	s.Profiles = sorted
	s.Folders = tidyFolders(s.Folders)

	if err := s.Validate(); err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(s); err != nil {
		return fmt.Errorf("encoding the connection store: %w", err)
	}
	if err := fsutil.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		return fmt.Errorf("saving %s: %w", path, err)
	}
	return nil
}

// tidyFolders sorts and deduplicates the explicit folder list so the file's diff
// stays stable.
func tidyFolders(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	out := slices.Clone(in)
	slices.Sort(out)
	return slices.Compact(out)
}
