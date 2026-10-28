package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/poppyszn/super-ssh-v2/internal/config"
	"github.com/poppyszn/super-ssh-v2/internal/paths"
)

// Import parses ~/.ssh/config and creates profiles from Host entries.
func ImportCmd(p paths.Paths, args []string) error {
	sshConfigPath := filepath.Join(os.Getenv("HOME"), ".ssh", "config")
	file, err := os.Open(sshConfigPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", sshConfigPath, err)
	}
	defer file.Close()

	s, err := config.Load(p.StoreFile)
	if err != nil {
		return err
	}
	existing := make(map[string]bool)
	for _, prof := range s.Profiles {
		existing[prof.Name] = true
	}

	var added int
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "Host ") && !strings.Contains(line, "*") {
			name := strings.Fields(line)[1]
			if !existing[name] {
				prof := parseHostBlock(scanner, name)
				s.Profiles = append(s.Profiles, prof)
				existing[name] = true
				added++
			}
		}
	}
	if added == 0 {
		fmt.Println("No new profiles found.")
		return nil
	}
	if err := config.Save(p.StoreFile, s); err != nil {
		return err
	}
	if err := regenerateConfig(p, s); err != nil {
		return err
	}
	fmt.Printf("Imported %d profile(s).\n", added)
	return nil
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
