package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
)

// scanHiddenLine reads one line like scanLine, but disables terminal echo
// for the duration -- for passphrases, where the input shouldn't appear on
// screen.
//
// This shells out to the standard `stty` utility rather than using
// golang.org/x/term or hand-rolled termios syscalls. golang.org/x/term is on
// the guardrails blocklist alongside the TUI libraries v1 died from, and
// hand-rolling raw ioctl calls means getting a platform-specific magic number
// exactly right or leaving the user's terminal in a broken state (needing
// `reset` to fix). `stty` is a POSIX utility present on effectively every
// Unix system, already handles the platform-specific details correctly, and
// this follows the same pattern already used for fzf elsewhere in this
// codebase: shell out to an optional external tool, degrade gracefully if
// it's unavailable rather than fail.
//
// If stdin isn't an interactive terminal (piped input, a test, a script) or
// `stty` isn't available, this silently falls back to plain, visible input --
// never an error. That fallback is also exactly what makes this hermetically
// testable: tests feed piped input, which is never a terminal, so they never
// touch stty at all.
func scanHiddenLine(in *bufio.Scanner, label string) (value string, ok bool) {
	restore, disabled := disableEcho()
	defer restore()
	value, ok = scanLine(in, label)
	if disabled {
		// The Enter keypress that ended input wasn't echoed either.
		fmt.Println()
	}
	return value, ok
}

// disableEcho best-effort disables terminal echo on stdin. The returned
// restore function must always be called (deferred), even on error -- the
// terminal must never be left in a state where nothing the user types is
// visible after an error path. disabled reports whether echo was actually
// turned off, so the caller knows whether the terminal will have echoed the
// terminating Enter keypress or not.
func disableEcho() (restore func(), disabled bool) {
	noop := func() {}
	if !isInteractiveStdin() {
		return noop, false
	}
	off := exec.Command("stty", "-echo")
	off.Stdin = os.Stdin
	if err := off.Run(); err != nil {
		return noop, false
	}
	return func() {
		on := exec.Command("stty", "echo")
		on.Stdin = os.Stdin
		_ = on.Run()
	}, true
}

func isInteractiveStdin() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
