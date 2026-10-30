package cmd

import (
	"bufio"
	"fmt"

	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/style"
)

const noJumpOption = "(no jump)"

// PickJump offers a menu of existing profiles to jump through. Every valid
// jump target is, by definition, an existing profile, so a picker can
// structurally rule out the two ways a typed answer used to go wrong:
// profileName is never offered (no self-jump), and every other option is a
// real, already-saved profile (no "not a saved profile" rejection needed --
// there is nothing else to offer).
//
// current is the profile's existing Jump value; when non-empty it is offered
// first as "Keep current (X)" so editing a profile doesn't force reselecting
// an unchanged target. Pass "" for current when there is none yet (`add`, or
// clearing during a cycle-retry).
func PickJump(in *bufio.Scanner, s config.Store, profileName, current string) (string, error) {
	var options []string
	keepLabel := fmt.Sprintf("Keep current (%s)", current)
	if current != "" {
		options = append(options, keepLabel)
	}
	options = append(options, noJumpOption)
	for _, prof := range s.Profiles {
		if prof.IsRDP() {
			continue
		}
		if prof.Name != profileName && prof.Name != current {
			options = append(options, prof.Name)
		}
	}
	if len(options) == 1 {
		// Only "(no jump)" available: no other profile exists yet to jump
		// through, so there's nothing to pick from.
		return "", nil
	}

	fmt.Println(style.Header("Jump through another profile first?"))
	choice, err := tryFzf(options)
	if err != nil {
		choice, err = menuPick(in, options)
		if err != nil {
			return "", err
		}
	}
	switch choice {
	case keepLabel:
		return current, nil
	case noJumpOption:
		return "", nil
	default:
		return choice, nil
	}
}
