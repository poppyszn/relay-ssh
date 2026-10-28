package core

import (
	"bufio"
	"io"
	"strconv"
	"strings"

	"github.com/poppyszn/super-ssh-v2/internal/config"
	"github.com/poppyszn/super-ssh-v2/internal/paths"
)

// ImportSSHConfig reads ssh_config text from r and adds a profile for every
// concrete Host entry not already in the store. It returns how many were added.
func ImportSSHConfig(p paths.Paths, r io.Reader) (int, error) {
	s, err := config.Load(p.StoreFile)
	if err != nil {
		return 0, err
	}
	existing := make(map[string]bool)
	for _, prof := range s.Profiles {
		existing[prof.Name] = true
	}

	var added int
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "Host ") && !strings.Contains(line, "*") {
			name := strings.Fields(line)[1]
			if !existing[name] {
				s.Profiles = append(s.Profiles, parseHostBlock(scanner, name))
				existing[name] = true
				added++
			}
		}
	}
	if added == 0 {
		return 0, nil
	}
	if err := config.Save(p.StoreFile, s); err != nil {
		return 0, err
	}
	return added, Regenerate(p, s)
}

func parseHostBlock(scanner *bufio.Scanner, name string) config.Profile {
	prof := config.Profile{Name: name, Port: 22, Auth: config.AuthAgent}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "Host ") {
			return prof
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := strings.ToLower(fields[0])
		val := strings.Join(fields[1:], " ")
		switch key {
		case "hostname":
			prof.Host = val
		case "user":
			prof.User = val
		case "port":
			if p, err := strconv.Atoi(val); err == nil {
				prof.Port = p
			}
		case "identityfile":
			prof.Auth = config.AuthKey
			prof.Key = val
		}
	}
	return prof
}
