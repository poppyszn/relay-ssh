package core

import (
	"fmt"
	"os"

	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/keys"
	"github.com/poppyszn/relay-ssh/internal/paths"
	"github.com/poppyszn/relay-ssh/internal/sshconfig"
)

// Status is the outcome of a single health check.
type Status int

const (
	OK Status = iota
	Warn
	Fail
)

// Check is one line of the health report.
type Check struct {
	Name   string
	Status Status
	Detail string
}

// Diagnose inspects the installation and returns one Check per area. It never
// modifies anything.
func Diagnose(p paths.Paths) []Check {
	var out []Check
	add := func(name string, st Status, detail string) {
		out = append(out, Check{name, st, detail})
	}

	s, storeErr := config.Load(p.StoreFile)
	if storeErr != nil {
		add("connections.toml", Fail, storeErr.Error())
	} else {
		add("connections.toml", OK, fmt.Sprintf("%d profiles", len(s.Profiles)))
	}

	sshConfigData, sshConfigErr := os.ReadFile(p.SSHConfig)
	if sshConfigErr != nil {
		add("~/.ssh/config", Fail, "not found")
	} else {
		content := string(sshConfigData)
		switch {
		case !sshconfig.HasBlock(content):
			add("~/.ssh/config", Warn, "no managed block (run `sssh sync`)")
		case !sshconfig.BlockIsFirst(content):
			add("~/.ssh/config", Fail, "managed block exists but is not at top")
		default:
			add("~/.ssh/config", OK, "managed block present and at top")
		}
	}

	if storeErr != nil || len(s.Profiles) == 0 {
		return out
	}

	expected := sshconfig.Generate(s)
	genData, genErr := os.ReadFile(p.GenConfig)
	switch {
	case genErr != nil:
		add("generated config", Fail, "not found (run `sssh sync`)")
	case string(genData) != expected:
		add("generated config", Warn, "out of date (run `sssh sync`)")
	default:
		add("generated config", OK, "up to date")
	}

	var keyNames []string
	for _, prof := range s.Profiles {
		if prof.Auth == config.AuthKey && prof.Key != "" && !isAbsolutePath(prof.Key) {
			keyNames = append(keyNames, prof.Key)
		}
	}
	if len(keyNames) > 0 {
		if missing := keys.ValidateAdopted(p, keyNames); len(missing) == 0 {
			add("adopted keys", OK, fmt.Sprintf("%d keys present", len(keyNames)))
		} else {
			add("adopted keys", Fail, fmt.Sprintf("missing %v", missing))
		}
	}
	return out
}

// Fix applies the two deterministic remediations (regenerate the generated
// config, restore the Include block) and returns any failures -- silently
// discarding an error here would leave the recheck failing again with no
// indication of why the "fix" didn't help.
func Fix(p paths.Paths) []error {
	var errs []error
	if s, err := config.Load(p.StoreFile); err == nil && len(s.Profiles) > 0 {
		if err := sshconfig.Write(p, s); err != nil {
			errs = append(errs, fmt.Errorf("regenerating config failed: %w", err))
		}
	}
	if err := sshconfig.EnsureInclude(p); err != nil {
		errs = append(errs, fmt.Errorf("updating ~/.ssh/config failed: %w", err))
	}
	return errs
}

func isAbsolutePath(p string) bool {
	return len(p) > 0 && (p[0] == '/' || p[0] == '~')
}
