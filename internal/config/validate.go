package config

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// reservedNames are the subcommands. A profile may not take one of these,
// because `sssh <name>` dispatches on the same argument position: a profile
// called "backup" would be unreachable, and would shadow the backup command.
var reservedNames = map[string]bool{
	"ls": true, "add": true, "edit": true, "rm": true, "cp": true,
	"import": true, "backup": true, "restore": true, "doctor": true,
	"sync": true, "version": true, "help": true,
}

// validNameChar allows letters, digits, '.', '_' and '-'.
//
// This is stricter than ssh itself requires, on purpose. ssh treats '*', '?'
// and '!' as pattern metacharacters, so a profile named "web*" would generate a
// Host block silently matching hosts the user never configured -- including,
// potentially, every host they have. '#' would start a comment mid-line. A
// space would split the alias in two. Rejecting the whole class is safer than
// enumerating the dangerous ones.
func validNameChar(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		return true
	case r == '.', r == '_', r == '-':
		return true
	}
	return false
}

// ValidateName checks a profile name in isolation: non-empty, not a reserved
// subcommand, and made only of characters ssh's config format treats literally
// (see validNameChar). Exported so callers can validate a name field-by-field
// -- e.g. the `add` command, which re-prompts on the spot instead of
// collecting every field and only discovering a bad one at save time.
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	if reservedNames[name] {
		return fmt.Errorf("name %q is reserved for the sssh subcommand of the same name", name)
	}
	for _, r := range name {
		if !validNameChar(r) {
			return fmt.Errorf("name contains %q; only letters, digits, '.', '_' and '-' are allowed", string(r))
		}
	}
	return nil
}

// ValidatePort checks a port falls within the valid TCP range. Exported for
// the same field-by-field reason as ValidateName.
func ValidatePort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}
	return nil
}

// ValidateFolder checks a folder path has no leading or trailing slash and no
// empty segment ("a//b"). The empty string means "no folder" and is allowed.
// Exported for the same field-by-field reason as ValidateName.
func ValidateFolder(folder string) error {
	if strings.HasPrefix(folder, "/") || strings.HasSuffix(folder, "/") {
		return fmt.Errorf("folder %q must not begin or end with '/'", folder)
	}
	if strings.Contains(folder, "//") {
		return fmt.Errorf("folder %q must not contain an empty segment", folder)
	}
	return nil
}

// checkValue rejects a value that could add a line or an option once it is written into the
// generated ssh config (a line break or other control character; with spaces=false also a space).
func checkValue(field, v string, spaces bool) error {
	for _, r := range v {
		if unicode.IsControl(r) || (!spaces && unicode.IsSpace(r)) {
			return fmt.Errorf("%s must not contain whitespace or control characters", field)
		}
	}
	return nil
}

// Validate checks a single profile in isolation. Cross-profile rules -- unique
// names, jump targets resolving, jump chains terminating -- live on Store.
func (p Profile) Validate() error {
	if err := ValidateName(p.Name); err != nil {
		return err
	}
	if p.Host == "" {
		return fmt.Errorf("profile %q: host is required", p.Name)
	}
	if p.User == "" {
		return fmt.Errorf("profile %q: user is required", p.Name)
	}
	if err := errors.Join(checkValue("host", p.Host, false), checkValue("user", p.User, false), checkValue("key", p.Key, true)); err != nil {
		return fmt.Errorf("profile %q: %w", p.Name, err)
	}
	if strings.HasPrefix(p.Host, "-") {
		return fmt.Errorf("profile %q: host must not start with '-'", p.Name)
	}
	if err := ValidatePort(p.Port); err != nil {
		return fmt.Errorf("profile %q: %w", p.Name, err)
	}
	switch p.Auth {
	case AuthAgent, AuthPassword:
		if p.Key != "" {
			return fmt.Errorf("profile %q: key is set but auth is %q; use auth = \"key\" or clear the key", p.Name, p.Auth)
		}
	case AuthKey:
		if p.Key == "" {
			return fmt.Errorf("profile %q: key is required when auth is %q", p.Name, AuthKey)
		}
	default:
		return fmt.Errorf("profile %q: unknown auth %q (want \"agent\", \"key\" or \"password\")", p.Name, p.Auth)
	}
	if err := ValidateFolder(p.Folder); err != nil {
		return fmt.Errorf("profile %q: %w", p.Name, err)
	}
	if p.Blocks != "" && p.Blocks != "on" && p.Blocks != "off" {
		return fmt.Errorf("profile %q: blocks must be \"on\", \"off\" or empty, not %q", p.Name, p.Blocks)
	}
	for i, t := range p.Tunnels {
		if err := t.Validate(); err != nil {
			return fmt.Errorf("profile %q, tunnel %d: %w", p.Name, i+1, err)
		}
	}
	return nil
}

// Validate checks one forwarding rule.
func (t Tunnel) Validate() error {
	if err := errors.Join(checkValue("listen", t.Listen, false), checkValue("target", t.Target, false)); err != nil {
		return err
	}
	switch t.Type {
	case TunnelLocal, TunnelRemote:
		if t.Listen == "" {
			return fmt.Errorf("listen is required")
		}
		if t.Target == "" {
			return fmt.Errorf("target is required for a %s tunnel", t.Type)
		}
	case TunnelDynamic:
		if t.Listen == "" {
			return fmt.Errorf("listen is required")
		}
		if t.Target != "" {
			return fmt.Errorf("a dynamic tunnel must not set target (it is a SOCKS proxy, not a fixed route)")
		}
	default:
		return fmt.Errorf("unknown tunnel type %q (want \"local\", \"remote\" or \"dynamic\")", t.Type)
	}
	return nil
}

// Validate checks every profile, then the rules that span profiles.
func (s Store) Validate() error {
	if s.Version != Version {
		return fmt.Errorf("unsupported store version %d (this sssh understands version %d)", s.Version, Version)
	}
	for _, f := range s.Folders {
		if f == "" {
			return fmt.Errorf("a folder in the folders list is empty")
		}
		if err := ValidateFolder(f); err != nil {
			return err
		}
	}
	seen := make(map[string]bool, len(s.Profiles))
	for _, p := range s.Profiles {
		if err := p.Validate(); err != nil {
			return err
		}
		if seen[p.Name] {
			return fmt.Errorf("duplicate profile name %q", p.Name)
		}
		seen[p.Name] = true
	}
	for _, p := range s.Profiles {
		if p.Jump != "" && !seen[p.Jump] {
			return fmt.Errorf("profile %q: jump target %q is not a saved profile", p.Name, p.Jump)
		}
	}
	return s.checkJumpCycles()
}

// checkJumpCycles walks each profile's jump chain. Without this, a cycle makes
// ssh recurse until it exhausts file descriptors, with an error that points
// nowhere near the real cause.
func (s Store) checkJumpCycles() error {
	jump := make(map[string]string, len(s.Profiles))
	for _, p := range s.Profiles {
		jump[p.Name] = p.Jump
	}
	for _, p := range s.Profiles {
		visited := map[string]bool{p.Name: true}
		for cur := p.Jump; cur != ""; cur = jump[cur] {
			if visited[cur] {
				return fmt.Errorf("profile %q: jump chain forms a cycle through %q", p.Name, cur)
			}
			visited[cur] = true
		}
	}
	return nil
}
