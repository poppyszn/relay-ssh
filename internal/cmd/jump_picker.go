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
func PickJump(in *bufio.Scanner, s config.Store, self, current string) (string, error) {
	var options []string
	cur, _ := s.Find(current)
	me, _ := s.Find(self)
	keepLabel := fmt.Sprintf("Keep current (%s)", cur.Address())
	if cur.ID == "" {
		keepLabel = fmt.Sprintf("Keep current (%s)", current)
	}
	if current != "" {
		options = append(options, keepLabel)
	}
	options = append(options, noJumpOption)
	for _, prof := range s.Profiles {
		if prof.IsRDP() || (me.ID != "" && prof.ID == me.ID) || prof.Name == self && me.ID == "" || (cur.ID != "" && prof.ID == cur.ID) || (cur.ID == "" && prof.Name == current) {
			continue
		}
		options = append(options, prof.Address())
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
		if picked, ok := s.Find(choice); ok && picked.ID != "" {
			return picked.ID, nil
		}
		return choice, nil
	}
}
