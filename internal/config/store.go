package config

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"

	"github.com/BurntSushi/toml"

	"github.com/poppyszn/super-ssh-v2/internal/fsutil"
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
	var s Store
	if _, err := toml.Decode(string(data), &s); err != nil {
		return Store{}, fmt.Errorf("parsing %s: %w", path, err)
	}
	if err := s.Validate(); err != nil {
		return Store{}, fmt.Errorf("%s: %w", path, err)
	}
	return s, nil
}

// Save validates, sorts, and writes the store atomically at 0600.
//
// Sorting keeps the file's diff stable, so a private git repo of
// connections.toml shows real changes rather than reordering noise. Validation
// happens before any bytes are written: an invalid store must never reach disk,
// because Load would then refuse to read it back.
func Save(path string, s Store) error {
	s.Version = Version
	sorted := make([]Profile, len(s.Profiles))
	copy(sorted, s.Profiles)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })
	s.Profiles = sorted

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
