package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/BurntSushi/toml"
	"github.com/poppyszn/relay-ssh/internal/fsutil"
	"github.com/poppyszn/relay-ssh/internal/paths"
)

// Snippet is a saved command the user can type into a terminal.
type Snippet struct {
	Name    string `json:"name" toml:"name"`
	Command string `json:"command" toml:"command"`
}

type snippetFile struct {
	Snippets []Snippet `toml:"snippet"`
}

const (
	maxSnippetName    = 80
	maxSnippetCommand = 20000
)

// SnippetsService keeps snippets in snippets.toml next to connections.toml: a file a
// person can read and fix by hand. It belongs to the desktop app (the CLI never reads it).
type SnippetsService struct{}

func snippetsPath() (string, error) {
	p, err := paths.Default()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(p.StoreFile), "snippets.toml"), nil
}

func loadSnippets() ([]Snippet, string, error) {
	path, err := snippetsPath()
	if err != nil {
		return nil, "", err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, path, nil
	}
	if err != nil {
		return nil, path, err
	}
	var f snippetFile
	// A garbled file is an error, never an empty list: the next save would overwrite it.
	if err := toml.Unmarshal(data, &f); err != nil {
		return nil, path, fmt.Errorf("snippets.toml is not valid: %w", err)
	}
	return f.Snippets, path, nil
}

func saveSnippets(path string, list []Snippet) error {
	slices.SortFunc(list, func(a, b Snippet) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) })
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(snippetFile{Snippets: list}); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return fsutil.WriteFile(path, buf.Bytes(), 0o600)
}

func (s Snippet) validate() error {
	name := strings.TrimSpace(s.Name)
	switch {
	case name == "":
		return errors.New("A snippet needs a name.")
	case len(name) > maxSnippetName:
		return fmt.Errorf("A snippet name can be at most %d characters.", maxSnippetName)
	case strings.IndexFunc(name, unicode.IsControl) >= 0:
		return errors.New("A snippet name cannot contain control characters.")
	case strings.TrimSpace(s.Command) == "":
		return errors.New("A snippet needs a command.")
	case len(s.Command) > maxSnippetCommand:
		return fmt.Errorf("A snippet command can be at most %d characters.", maxSnippetCommand)
	case strings.ContainsRune(s.Command, 0):
		return errors.New("A snippet command cannot contain a NUL character.")
	}
	return nil
}

// List returns every snippet, sorted by name.
func (SnippetsService) List() ([]Snippet, error) {
	list, _, err := loadSnippets()
	if list == nil {
		list = []Snippet{}
	}
	slices.SortFunc(list, func(a, b Snippet) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) })
	return list, err
}

// Save adds a snippet, or replaces the one called oldName ("" for a new one).
func (SnippetsService) Save(oldName string, s Snippet) error {
	s.Name = strings.TrimSpace(s.Name)
	if err := s.validate(); err != nil {
		return err
	}
	list, path, err := loadSnippets()
	if err != nil {
		return err
	}
	for _, o := range list {
		if o.Name == s.Name && o.Name != oldName {
			return fmt.Errorf("There is already a snippet called %q.", s.Name)
		}
	}
	if i := slices.IndexFunc(list, func(o Snippet) bool { return o.Name == oldName }); oldName != "" && i >= 0 {
		list[i] = s
	} else {
		list = append(list, s)
	}
	return saveSnippets(path, list)
}

// Delete removes the snippet called name.
func (SnippetsService) Delete(name string) error {
	list, path, err := loadSnippets()
	if err != nil {
		return err
	}
	i := slices.IndexFunc(list, func(o Snippet) bool { return o.Name == name })
	if i < 0 {
		return fmt.Errorf("There is no snippet called %q.", name)
	}
	return saveSnippets(path, slices.Delete(list, i, i+1))
}
