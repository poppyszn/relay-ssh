package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/poppyszn/super-ssh-v2/internal/config"
	"github.com/poppyszn/super-ssh-v2/internal/paths"
	"github.com/poppyszn/super-ssh-v2/internal/style"
)

// EditCmd interactively edits an existing profile.
func EditCmd(p paths.Paths, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("edit requires a profile name")
	}
	s, err := config.Load(p.StoreFile)
	if err != nil {
		return err
	}
	name := args[0]
	prof, ok := resolveProfile(s, name)
	if !ok {
		return fmt.Errorf("profile %q not found", name)
	}

	in := bufio.NewScanner(os.Stdin)
	fmt.Println(style.Header(fmt.Sprintf("── Edit profile %s ──", prof.Name)))

	// 1. Name
	newName, err := promptEditName(in, s, prof.Name)
	if err != nil {
		return err
	}

	// 2. Host
	newHost, err := promptEditField(in, "Hostname or IP", prof.Host)
	if err != nil {
		return err
	}

	// 3. User
	newUser, err := promptEditField(in, "SSH user", prof.User)
	if err != nil {
		return err
	}

	// 4. Port
	newPort, err := promptEditPort(in, prof.Port)
	if err != nil {
		return err
	}

	// 5. Auth & Key
	newAuth, newKey, err := promptEditAuth(in, prof.Auth, prof.Key)
	if err != nil {
		return err
	}

	// 6. Jump
	// Exclude prof.Name (the profile's current name in s), not newName: if
	// the user is renaming, newName doesn't exist as an entry in s yet, so
	// excluding it would fail to exclude the profile actually being edited --
	// its old name would slip through as a valid (but wrong) jump option.
	newJump, err := PickJump(in, s, prof.Name, prof.Jump)
	if err != nil {
		return err
	}

	// 7. ForwardAgent
	newForwardAgent := promptEditForwardAgent(in, prof.ForwardAgent)

	// 8. Folder
	newFolder, err := promptEditFolder(in, s, prof.Folder)
	if err != nil {
		return err
	}

	// 9. Tags
	newTags := promptEditTags(in, prof.Tags)

	// 10. Tunnels
	newTunnels, err := promptEditTunnels(in, prof.Tunnels)
	if err != nil {
		return err
	}

	edited := prof
	edited.Name, edited.Host, edited.User = newName, newHost, newUser
	edited.Port, edited.Auth, edited.Key = newPort, newAuth, newKey
	edited.Jump, edited.ForwardAgent = newJump, newForwardAgent
	edited.Folder, edited.Tags, edited.Tunnels = newFolder, newTags, newTunnels

	// Cycle detection loop on whole Store.Validate():
	for {
		candidateStore := s
		var updatedProfiles []config.Profile
		for _, p := range candidateStore.Profiles {
			if p.Name == prof.Name {
				updatedProfiles = append(updatedProfiles, edited)
			} else {
				updatedProfiles = append(updatedProfiles, p)
			}
		}
		candidateStore.Profiles = updatedProfiles

		if err := candidateStore.Validate(); err != nil {
			fmt.Println(style.Fail("✗ " + err.Error()))
			// current="" here, not edited.Jump: the value that just failed
			// shouldn't be re-offered as "Keep current" -- that would put a
			// menu option right next to the failure message inviting the
			// user to re-select the exact thing that was just rejected.
			reJump, err := PickJump(in, s, prof.Name, "")
			if err != nil {
				return err
			}
			edited.Jump = reJump
			continue
		}

		s = candidateStore
		break
	}

	if err := config.Save(p.StoreFile, s); err != nil {
		return fmt.Errorf("saving profile: %w", err)
	}
	if err := regenerateConfig(p, s); err != nil {
		return err
	}
	fmt.Printf("%s Profile %s updated.\n", style.Success("✓"), style.Bold(edited.Name))
	return nil
}

func promptEditName(in *bufio.Scanner, s config.Store, current string) (string, error) {
	for {
		name, ok := scanLine(in, fmt.Sprintf("Profile name [%s]", current))
		if !ok {
			return "", errCancelled
		}
		if name == "" || name == current {
			return current, nil
		}
		if err := config.ValidateName(name); err != nil {
			fmt.Println(style.Fail("✗ " + err.Error()))
			continue
		}
		if _, exists := s.Find(name); exists {
			fmt.Println(style.Fail(fmt.Sprintf("✗ profile %q already exists", name)))
			continue
		}
		return name, nil
	}
}

func promptEditField(in *bufio.Scanner, label, current string) (string, error) {
	val, ok := scanLine(in, fmt.Sprintf("%s [%s]", label, current))
	if !ok {
		return "", errCancelled
	}
	if val == "" {
		return current, nil
	}
	return val, nil
}

func promptEditPort(in *bufio.Scanner, current int) (int, error) {
	for {
		portStr, ok := scanLine(in, fmt.Sprintf("Port [%d]", current))
		if !ok {
			return 0, errCancelled
		}
		if portStr == "" {
			return current, nil
		}
		n, err := strconv.Atoi(portStr)
		if err != nil {
			fmt.Println(style.Fail("✗ port must be a number"))
			continue
		}
		if err := config.ValidatePort(n); err != nil {
			fmt.Println(style.Fail("✗ " + err.Error()))
			continue
		}
		return n, nil
	}
}

func promptEditAuth(in *bufio.Scanner, currentAuth config.AuthType, currentKey string) (config.AuthType, string, error) {
	fmt.Println(style.Header(fmt.Sprintf("Auth type [%s]:", currentAuth)))
	fmt.Printf("  %s Keep current %s\n", style.Dim("0)"), style.Dim(fmt.Sprintf("(%s)", currentAuth)))
	fmt.Printf("  %s Agent %s\n", style.Dim("1)"), style.Dim("(uses ssh-agent)"))
	fmt.Printf("  %s Key %s\n", style.Dim("2)"), style.Dim("(a private key file)"))
	fmt.Printf("  %s Password %s\n", style.Dim("3)"), style.Dim("(prompted by ssh itself)"))
	choice := prompt(in, style.Bold("Select"))
	var auth config.AuthType
	switch strings.TrimSpace(choice) {
	case "1", "a", "agent":
		auth = config.AuthAgent
	case "2", "k", "key":
		auth = config.AuthKey
	case "3", "p", "password":
		auth = config.AuthPassword
	default:
		auth = currentAuth
	}

	if auth == config.AuthKey {
		if currentKey != "" {
			key, ok := scanLine(in, fmt.Sprintf("Key file [%s] (press Enter to keep, or type path)", currentKey))
			if !ok {
				return "", "", errCancelled
			}
			if key == "" {
				return auth, currentKey, nil
			}
			return auth, key, nil
		}
		key, err := promptKey(in)
		if err != nil {
			return "", "", err
		}
		return auth, key, nil
	}

	return auth, "", nil
}

func promptEditFolder(in *bufio.Scanner, s config.Store, current string) (string, error) {
	currentLabel := current
	if currentLabel == "" {
		currentLabel = "(no folder)"
	}
	change, ok := scanLine(in, fmt.Sprintf("Folder [%s] — change folder? (y/N)", currentLabel))
	if !ok {
		return "", errCancelled
	}
	if !strings.HasPrefix(strings.ToLower(change), "y") {
		return current, nil
	}
	return PickFolder(in, s)
}

func promptEditTags(in *bufio.Scanner, current []string) []string {
	currentStr := strings.Join(current, ", ")
	label := "Tags (comma-separated, optional)"
	if currentStr != "" {
		label = fmt.Sprintf("Tags (comma-separated, optional) [%s] ('-' to clear)", currentStr)
	}
	val := prompt(in, label)
	if val == "" {
		return current
	}
	if val == "-" {
		return nil
	}
	return parseTags(val)
}

func promptEditForwardAgent(in *bufio.Scanner, current bool) bool {
	defaultStr := "y/N"
	if current {
		defaultStr = "Y/n"
	}
	val := prompt(in, fmt.Sprintf("Forward the SSH agent to this host? (%s)", defaultStr))
	val = strings.ToLower(strings.TrimSpace(val))
	if val == "" {
		return current
	}
	return strings.HasPrefix(val, "y")
}
