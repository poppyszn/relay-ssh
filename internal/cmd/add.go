package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/poppyszn/super-ssh-v2/internal/config"
	"github.com/poppyszn/super-ssh-v2/internal/core"
	"github.com/poppyszn/super-ssh-v2/internal/paths"
	"github.com/poppyszn/super-ssh-v2/internal/style"
)

// Add prompts for profile fields and adds a new connection.
//
// Every field is validated as it's entered, not just at Save time. Nothing
// here is worse than filling in a whole form only to be told at the very end
// that the second field was wrong -- and having to retype the rest from
// scratch, since nothing was kept.
func Add(p paths.Paths, args []string) error {
	s, err := config.Load(p.StoreFile)
	if err != nil {
		return err
	}
	in := bufio.NewScanner(os.Stdin)

	fmt.Println(style.Header("── Add a new profile ──"))
	name, err := promptName(in, s)
	if err != nil {
		return err
	}
	host, err := promptRequired(in, "Hostname or IP")
	if err != nil {
		return err
	}
	user, err := promptRequired(in, "SSH user")
	if err != nil {
		return err
	}
	port, err := promptPort(in)
	if err != nil {
		return err
	}
	auth := promptAuthType(in)

	prof := config.Profile{Name: name, Host: host, User: user, Port: port, Auth: auth}
	if auth == config.AuthKey {
		key, err := promptKey(in)
		if err != nil {
			return err
		}
		prof.Key = key
	}
	if prof.Jump, err = PickJump(in, s, prof.Name, ""); err != nil {
		return err
	}
	if prof.ForwardAgent, err = promptForwardAgent(in); err != nil {
		return err
	}
	if prof.Folder, err = PickFolder(in, s); err != nil {
		return err
	}
	prof.Tags = parseTags(prompt(in, "Tags (comma-separated, optional)"))
	if prof.Tunnels, err = promptTunnels(in); err != nil {
		return err
	}

	for {
		candidateStore := s
		candidateStore.Profiles = append(candidateStore.Profiles, prof)
		if err := candidateStore.Validate(); err != nil {
			fmt.Println(style.Fail("✗ " + err.Error()))
			newJump, err := PickJump(in, s, prof.Name, "")
			if err != nil {
				return err
			}
			prof.Jump = newJump
			continue
		}
		s = candidateStore
		break
	}

	if err := config.Save(p.StoreFile, s); err != nil {
		return fmt.Errorf("saving profile: %w", err)
	}
	if err := core.Regenerate(p, s); err != nil {
		return err
	}
	fmt.Printf("%s Profile %s added.\n", style.Success("✓"), style.Bold(name))
	return nil
}

// promptName loops until it has a name that passes ValidateName and isn't
// already taken -- the two ways a name can fail that only show up once you
// actually try to use it.
func promptName(in *bufio.Scanner, s config.Store) (string, error) {
	for {
		name, ok := scanLine(in, "Profile name")
		if !ok {
			return "", errCancelled
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

// promptRequired loops until it gets a non-blank line.
func promptRequired(in *bufio.Scanner, label string) (string, error) {
	for {
		val, ok := scanLine(in, label)
		if !ok {
			return "", errCancelled
		}
		if val == "" {
			fmt.Println(style.Fail("✗ this field is required"))
			continue
		}
		return val, nil
	}
}

// promptPort loops until it gets a blank line (default 22), or a number in
// the valid TCP range.
func promptPort(in *bufio.Scanner) (int, error) {
	for {
		portStr, ok := scanLine(in, "Port (default 22)")
		if !ok {
			return 0, errCancelled
		}
		if portStr == "" {
			return 22, nil
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

// promptAuthType shows a small numbered menu for auth type, defaulting to
// agent on a blank line since that requires nothing else from the user.
func promptAuthType(in *bufio.Scanner) config.AuthType {
	fmt.Println(style.Header("Auth type:"))
	fmt.Printf("  %s Agent %s\n", style.Dim("1)"), style.Dim("(default — uses ssh-agent)"))
	fmt.Printf("  %s Key %s\n", style.Dim("2)"), style.Dim("(a private key file)"))
	fmt.Printf("  %s Password %s\n", style.Dim("3)"), style.Dim("(prompted by ssh itself)"))
	choice := prompt(in, style.Bold("Select"))
	switch strings.TrimSpace(choice) {
	case "2", "k", "key":
		return config.AuthKey
	case "3", "p", "password":
		return config.AuthPassword
	default:
		return config.AuthAgent
	}
}

// promptKey loops PickKeyFile until it returns a non-empty path: a blank
// manual entry is the one way that picker can hand back nothing.
func promptKey(in *bufio.Scanner) (string, error) {
	for {
		key, err := PickKeyFile(in)
		if err != nil {
			return "", fmt.Errorf("selecting key: %w", err)
		}
		if key == "" {
			fmt.Println(style.Fail("✗ a key is required for key auth"))
			continue
		}
		return key, nil
	}
}

// errCancelled is returned by a prompt retry loop when stdin runs out before
// a valid value was ever entered, so a piped or closed input can't spin the
// loop forever re-asking a question nobody is there to answer.
var errCancelled = fmt.Errorf("cancelled: no input")

// scanLine reads one line, trimmed, after printing label as a prompt. ok is
// false when the scanner is exhausted (EOF) or errored.
func scanLine(in *bufio.Scanner, label string) (value string, ok bool) {
	if label != "" {
		fmt.Print(label + ": ")
	}
	if !in.Scan() {
		return "", false
	}
	return strings.TrimSpace(in.Text()), true
}

// prompt reads one line for callers that treat "blank input" and "no more
// input" the same way -- both become "".
func prompt(in *bufio.Scanner, msg string) string {
	val, _ := scanLine(in, msg)
	return val
}

// parseTags splits a comma-separated string, trimming spaces and discarding empties.
func parseTags(input string) []string {
	var tags []string
	for _, part := range strings.Split(input, ",") {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			tags = append(tags, trimmed)
		}
	}
	return tags
}

func promptForwardAgent(in *bufio.Scanner) (bool, error) {
	val, ok := scanLine(in, "Forward the SSH agent to this host? (y/N)")
	if !ok {
		return false, errCancelled
	}
	val = strings.ToLower(val)
	if strings.HasPrefix(val, "y") {
		return true, nil
	}
	return false, nil
}
