package core

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/user"
	"strconv"
	"strings"

	"github.com/poppyszn/super-ssh-v2/internal/config"
	"github.com/poppyszn/super-ssh-v2/internal/paths"
)

// ImportResult says which Host entries became profiles and which were skipped
// (with the reason), so a UI can report both.
type ImportResult struct {
	Added   []string
	Skipped []string
}

// ImportSSHConfig reads ssh_config text from r and adds a profile for every
// concrete Host entry not already in the store. It returns how many were added.
func ImportSSHConfig(p paths.Paths, r io.Reader) (int, error) {
	res, err := ImportSSHConfigDetailed(p, r)
	return len(res.Added), err
}

// ImportSSHConfigDetailed is ImportSSHConfig with a report. Entries that cannot
// become a valid profile are skipped instead of aborting the whole import.
func ImportSSHConfigDetailed(p paths.Paths, r io.Reader) (ImportResult, error) {
	var res ImportResult
	s, err := config.Load(p.StoreFile)
	if err != nil {
		return res, err
	}
	existing := make(map[string]bool)
	for _, prof := range s.Profiles {
		existing[prof.Name] = true
	}

	for _, prof := range parseHostBlocks(r) {
		if existing[prof.Name] {
			continue
		}
		if err := prof.Validate(); err != nil {
			res.Skipped = append(res.Skipped, fmt.Sprintf("%s: %v", prof.Name, err))
			continue
		}
		s.Profiles = append(s.Profiles, prof)
		existing[prof.Name] = true
		res.Added = append(res.Added, prof.Name)
	}
	if len(res.Added) == 0 {
		return res, nil
	}
	if err := config.Save(p.StoreFile, s); err != nil {
		return ImportResult{}, err
	}
	return res, Regenerate(p, s)
}

// parseHostBlocks splits ssh_config into one profile per concrete Host entry.
// A block runs until the next Host or Match line, so adjacent blocks with no
// blank line between them are all seen. Wildcard hosts are ignored. Fields ssh
// would default (HostName, User) get ssh's own defaults: the alias and the
// current user.
func parseHostBlocks(r io.Reader) []config.Profile {
	var out []config.Profile
	var cur *config.Profile
	flush := func() {
		if cur == nil {
			return
		}
		if cur.Host == "" {
			cur.Host = cur.Name
		}
		if cur.User == "" {
			cur.User = defaultUser()
		}
		out = append(out, *cur)
		cur = nil
	}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		fields := strings.Fields(line)
		if len(fields) < 2 || strings.HasPrefix(line, "#") {
			continue
		}
		key := strings.ToLower(fields[0])
		val := strings.Join(fields[1:], " ")
		switch key {
		case "host":
			flush()
			if name := fields[1]; !strings.ContainsAny(name, "*?!") {
				cur = &config.Profile{Name: name, Port: 22, Auth: config.AuthAgent}
			}
		case "match":
			flush()
		case "hostname":
			if cur != nil {
				cur.Host = val
			}
		case "user":
			if cur != nil {
				cur.User = val
			}
		case "port":
			if n, err := strconv.Atoi(val); err == nil && cur != nil {
				cur.Port = n
			}
		case "identityfile":
			if cur != nil {
				cur.Auth = config.AuthKey
				cur.Key = val
			}
		}
	}
	flush()
	return out
}

// defaultUser is who ssh logs in as when a Host entry names no User. Empty if
// unknown, so validation skips the entry rather than guessing (root, say).
func defaultUser() string {
	if u, err := user.Current(); err == nil && u.Username != "" {
		return u.Username
	}
	return os.Getenv("USER")
}
