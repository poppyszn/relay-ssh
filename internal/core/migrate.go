package core

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/paths"
	"github.com/poppyszn/relay-ssh/internal/sshconfig"
)

// Migrate brings a machine that used the tool under its old name (sssh) over to the current one:
// it moves ~/.config/sssh and ~/.ssh/sssh.d to their new places, repoints the managed block in
// ~/.ssh/config, and regenerates the SSH config so key paths and the Include line agree.
// A new tree that already exists is never overwritten, and a machine with nothing old is left
// alone. It is safe to call on every start.
func Migrate(p paths.Paths) error {
	moved := false
	for _, m := range [][2]string{{p.LegacyConfigDir(), p.ConfigDir}, {p.LegacySandbox(), p.Sandbox}} {
		if _, err := os.Stat(m[0]); err != nil {
			continue
		}
		if _, err := os.Stat(m[1]); err == nil {
			continue
		} else if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		if err := os.MkdirAll(p.SSHDir, 0o700); err != nil {
			return err
		}
		if err := os.MkdirAll(p.Root+"/.config", 0o700); err != nil {
			return err
		}
		if err := os.Rename(m[0], m[1]); err != nil {
			return fmt.Errorf("moving %s to %s: %w", m[0], m[1], err)
		}
		moved = true
	}
	upgraded, err := sshconfig.UpgradeLegacyBlock(p)
	if err != nil {
		return err
	}
	if !moved && !upgraded {
		return nil
	}
	s, err := config.Load(p.StoreFile)
	if err != nil {
		return nil // nothing to regenerate from; the next save will do it
	}
	return Regenerate(p, s)
}
