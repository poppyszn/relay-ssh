package sshconfig

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"

	"github.com/poppyszn/relay-ssh/internal/fsutil"
	"github.com/poppyszn/relay-ssh/internal/paths"
)

// The markers delimit the only part of ~/.ssh/config that sssh owns. Every
// byte outside them belongs to the user and is never modified.
const (
	BeginMarker = "# >>> sssh managed block >>>"
	EndMarker   = "# <<< sssh managed block <<<"
	IncludeLine = "Include ~/.ssh/sssh.d/config"
)

// Block is what gets inserted into the user's ssh config. The Include path uses
// a tilde rather than an absolute path so the same line works on any machine
// the config is restored onto.
var Block = BeginMarker + "\n" + IncludeLine + "\n" + EndMarker + "\n"

// HasBlock reports whether the managed block is present.
func HasBlock(content string) bool {
	return strings.Contains(content, BeginMarker) && strings.Contains(content, EndMarker)
}

// AddBlock returns content with the managed block prepended.
//
// Top placement is mandatory. ssh config is first-match-wins: the first value
// seen for a keyword is the one used. An Include placed below a "Host *" block
// would be read too late to affect anything that block already set, and the
// failure would be silent.
func AddBlock(content string) string {
	if HasBlock(content) {
		return content
	}
	if content == "" {
		return Block
	}
	return Block + "\n" + content
}

// RemoveBlock strips the managed block and leaves everything else byte-identical.
func RemoveBlock(content string) string {
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))
	inside := false
	for _, line := range lines {
		switch strings.TrimSpace(line) {
		case BeginMarker:
			inside = true
			continue
		case EndMarker:
			inside = false
			continue
		}
		if inside {
			continue
		}
		out = append(out, line)
	}
	// Drop the single blank line AddBlock inserted between the block and the
	// user's content, so remove(add(x)) == x.
	for len(out) > 0 && strings.TrimSpace(out[0]) == "" {
		out = out[1:]
	}
	return strings.Join(out, "\n")
}

// BlockIsFirst reports whether the managed block appears before any Host or
// Match directive. `sssh doctor` uses this to catch a config where the block was
// moved down, which disables it without any visible error.
func BlockIsFirst(content string) bool {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == BeginMarker {
			return true
		}
		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, "host ") || strings.HasPrefix(lower, "match ") {
			return false
		}
	}
	return false
}

// EnsureInclude adds the managed block to the user's ssh config if it is not
// already there, creating the file if the machine has none. An existing file
// keeps its permissions: sssh has no business tightening or loosening a mode
// the user chose.
func EnsureInclude(p paths.Paths) error {
	data, err := os.ReadFile(p.SSHConfig)
	if errors.Is(err, fs.ErrNotExist) {
		if err := fsutil.WriteFile(p.SSHConfig, []byte(Block), 0o600); err != nil {
			return fmt.Errorf("creating %s: %w", p.SSHConfig, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading %s: %w", p.SSHConfig, err)
	}
	if HasBlock(string(data)) {
		return nil
	}
	mode := os.FileMode(0o600)
	if info, statErr := os.Stat(p.SSHConfig); statErr == nil {
		mode = info.Mode().Perm()
	}
	if err := fsutil.WriteFile(p.SSHConfig, []byte(AddBlock(string(data))), mode); err != nil {
		return fmt.Errorf("updating %s: %w", p.SSHConfig, err)
	}
	return nil
}
