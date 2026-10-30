package cmd

import (
	"fmt"
	"sort"

	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/paths"
)

// ListProfileAddresses returns every profile's addressable name, sorted --
// "folder/name" for a profile in a folder, or just "name" otherwise. This is
// exactly the string a completion script should offer and exactly what
// resolveProfile accepts, so a tab-completed name is always immediately
// usable (`relay-ssh <TAB>` -> `relay-ssh Vascon/dev-app-server`, not something that
// then fails with "profile not found").
func ListProfileAddresses(p paths.Paths) ([]string, error) {
	s, err := config.Load(p.StoreFile)
	if err != nil {
		return nil, err
	}
	addrs := make([]string, 0, len(s.Profiles))
	for _, prof := range s.Profiles {
		if prof.Folder == "" {
			addrs = append(addrs, prof.Name)
		} else {
			addrs = append(addrs, prof.Folder+"/"+prof.Name)
		}
	}
	sort.Strings(addrs)
	return addrs, nil
}

// CompletionScript returns a shell completion script for the given shell.
// The script calls back into `relay-ssh __complete-profiles` at completion time
// (not at generation time), so profiles added or removed after the script is
// installed still complete correctly -- nothing needs regenerating when
// connections.toml changes.
func CompletionScript(shell string) (string, error) {
	switch shell {
	case "zsh":
		return zshCompletion, nil
	case "bash":
		return bashCompletion, nil
	default:
		return "", fmt.Errorf("unsupported shell %q (want \"bash\" or \"zsh\")", shell)
	}
}

const zshCompletion = `#compdef relay-ssh

_relay-ssh() {
  local -a profiles
  profiles=(${(f)"$(relay-ssh __complete-profiles 2>/dev/null)"})
  _describe 'profile' profiles
}

_relay-ssh "$@"
`

const bashCompletion = `_relay-ssh_complete() {
  local cur
  cur="${COMP_WORDS[COMP_CWORD]}"
  COMPREPLY=( $(compgen -W "$(relay-ssh __complete-profiles 2>/dev/null)" -- "$cur") )
}
complete -F _relay-ssh_complete relay-ssh
`
