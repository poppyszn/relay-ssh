// Package launch replaces the sssh process with SSH.
//
// Instead of spawning a child ssh process, sssh builds an argv and execs it,
// replacing itself entirely. This means sssh never consumes memory during the
// connection and never slows down terminal I/O.
package launch

import (
	"fmt"
	"syscall"

	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/paths"
)

// execFunc wraps the system exec call, overridable in tests.
var execFunc = func(name string, argv []string) error {
	if err := syscall.Exec("/usr/bin/"+name, argv, nil); err != nil {
		return fmt.Errorf("exec %s: %w", name, err)
	}
	return nil
}

// SSH builds an argv and execs the SSH binary.
//
// Profile and environment are final; ssh receives exactly what the profile
// specifies, nothing more, nothing less. The user can modify ~/.ssh/config
// outside sssh's managed block if they need SSH options sssh doesn't surface.
func SSH(p paths.Paths, profile config.Profile) error {
	argv := []string{"ssh", "-F", p.GenConfig, profile.Name}
	return execFunc("ssh", argv)
}

// SCP builds an argv for SSH copy and execs it.
//
// The src and dst arguments follow scp syntax: [user@]host:[path] or local path.
// The recursive flag sets -r.
func SCP(p paths.Paths, profile config.Profile, src string, dst string, recursive bool) error {
	argv := []string{"scp", "-F", p.GenConfig}
	if recursive {
		argv = append(argv, "-r")
	}
	argv = append(argv, src, dst)
	return execFunc("scp", argv)
}
