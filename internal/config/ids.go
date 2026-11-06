package config

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// NewID returns a fresh profile id: '@' plus 12 random hex digits.
func NewID() string {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		panic(err) // the system random source failing is not recoverable
	}
	return "@" + hex.EncodeToString(b)
}

// legacyID is the id a profile written before ids existed gets. It is derived from
// the folder and name, which were unique then, so every load agrees on it until
// the next Save writes it into the file.
func legacyID(folder, name string) string {
	h := sha256.Sum256([]byte(folder + "\x00" + name))
	return "@" + hex.EncodeToString(h[:6])
}

func validID(id string) bool {
	if len(id) != 13 || id[0] != '@' {
		return false
	}
	_, err := hex.DecodeString(id[1:])
	return err == nil
}

// Address is the profile's "Folder/name", or just the name at the top level.
func (p Profile) Address() string {
	if p.Folder == "" {
		return p.Name
	}
	return p.Folder + "/" + p.Name
}

// index finds a profile by id, then by address, then by a bare name only one
// profile has. It returns -1 when nothing matches or the name is ambiguous.
func (s Store) index(key string) int {
	if key == "" {
		return -1
	}
	for i, p := range s.Profiles {
		if p.ID == key {
			return i
		}
	}
	for i, p := range s.Profiles {
		if p.Address() == key {
			return i
		}
	}
	found := -1
	for i, p := range s.Profiles {
		if p.Name == key {
			if found >= 0 {
				return -1
			}
			found = i
		}
	}
	return found
}

// Resolve is Find with an explanation: an unknown key, or a bare name that several
// profiles share (listing each "Folder/name" that would be exact).
func (s Store) Resolve(key string) (Profile, error) {
	if i := s.index(key); i >= 0 {
		return s.Profiles[i], nil
	}
	var all []string
	for _, p := range s.Profiles {
		if p.Name == key {
			all = append(all, p.Address())
		}
	}
	if len(all) > 1 {
		return Profile{}, fmt.Errorf("%q matches %d connections; use one of: %s", key, len(all), strings.Join(all, ", "))
	}
	return Profile{}, fmt.Errorf("profile %q not found", key)
}

// Normalize gives every profile an id and rewrites jumps to point at ids, so a
// rename or move never breaks a jump. With random false (loading) a profile
// without an id gets its stable legacy id; with true (saving) it gets a fresh one.
func (s *Store) Normalize(random bool) {
	for i := range s.Profiles {
		p := &s.Profiles[i]
		if p.ID == "" {
			p.ID = legacyID(p.Folder, p.Name)
			if random {
				p.ID = NewID()
			}
		}
	}
	for i := range s.Profiles {
		if j := s.index(s.Profiles[i].Jump); j >= 0 {
			s.Profiles[i].Jump = s.Profiles[j].ID
		}
	}
}
