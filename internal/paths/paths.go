// Package paths derives every filesystem location sssh touches from a single
// root directory. Nothing else in the codebase spells out a path, which is what
// lets tests run against t.TempDir() and never see the real home directory.
package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

// Paths is the complete set of locations sssh reads or writes.
//
// Everything from Sandbox down is owned by sssh and may be rewritten freely.
// SSHConfig is shared with the user: only the marked Include block inside it
// belongs to sssh. KnownHosts is shared too, and is only ever merged.
type Paths struct {
	Root       string // the home directory this tree hangs off
	ConfigDir  string // <root>/.config/sssh
	StoreFile  string // <root>/.config/sssh/connections.toml -- the source of truth
	SSHDir     string // <root>/.ssh
	SSHConfig  string // <root>/.ssh/config       -- shared; only the marked block is ours
	Sandbox    string // <root>/.ssh/sssh.d       -- entirely ours
	GenConfig  string // <root>/.ssh/sssh.d/config
	KeysDir    string // <root>/.ssh/sssh.d/keys
	KnownHosts string // <root>/.ssh/known_hosts  -- shared; merged, never overwritten
}

// New builds the path set for a given home directory.
//
// The config directory is $HOME/.config/sssh on every platform. This
// deliberately does not use os.UserConfigDir, which on macOS returns
// ~/Library/Application Support -- a location that is awkward to back up by
// hand and surprising to anyone who expects dotfile conventions.
func New(root string) Paths {
	cfg := filepath.Join(root, ".config", "sssh")
	ssh := filepath.Join(root, ".ssh")
	sandbox := filepath.Join(ssh, "sssh.d")
	return Paths{
		Root:       root,
		ConfigDir:  cfg,
		StoreFile:  filepath.Join(cfg, "connections.toml"),
		SSHDir:     ssh,
		SSHConfig:  filepath.Join(ssh, "config"),
		Sandbox:    sandbox,
		GenConfig:  filepath.Join(sandbox, "config"),
		KeysDir:    filepath.Join(sandbox, "keys"),
		KnownHosts: filepath.Join(ssh, "known_hosts"),
	}
}

// Default uses $SSSH_HOME when set, otherwise the user's home directory.
// SSSH_HOME exists so the tool can be exercised end to end against a throwaway
// tree without risking the real ~/.ssh.
func Default() (Paths, error) {
	if root := os.Getenv("SSSH_HOME"); root != "" {
		return New(root), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, fmt.Errorf("locating home directory: %w", err)
	}
	return New(home), nil
}

// EnsureDirs creates every directory sssh writes into, at 0700. It is safe to
// call repeatedly.
func EnsureDirs(p Paths) error {
	for _, dir := range []string{p.ConfigDir, p.SSHDir, p.Sandbox, p.KeysDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return fmt.Errorf("creating %s: %w", dir, err)
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			return fmt.Errorf("setting permissions on %s: %w", dir, err)
		}
	}
	return nil
}
