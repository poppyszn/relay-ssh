package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/poppyszn/super-ssh-v2/internal/config"
	"github.com/poppyszn/super-ssh-v2/internal/paths"
	"github.com/poppyszn/super-ssh-v2/internal/style"
)

// Pick uses fzf if available, else a numbered menu. Returns selected profile name.
func Pick(p paths.Paths) (string, error) {
	s, err := config.Load(p.StoreFile)
	if err != nil {
		return "", err
	}
	if len(s.Profiles) == 0 {
		return "", fmt.Errorf("no profiles to pick from")
	}
	names := s.Names()
	if len(names) == 1 {
		return names[0], nil
	}
	if name, err := tryFzf(names); err == nil {
		return name, nil
	}
	in := bufio.NewScanner(os.Stdin)
	return menuPick(in, names)
}

// disableFzfForTest forces tryFzf to fail immediately, so tests exercise the
// menu fallback deterministically regardless of whether fzf happens to be
// installed on the machine running them.
var disableFzfForTest = false

// tryFzf attempts to use fzf. Returns error if not available or user cancelled.
func tryFzf(names []string) (string, error) {
	if disableFzfForTest {
		return "", fmt.Errorf("fzf disabled for test")
	}
	if _, err := exec.LookPath("fzf"); err != nil {
		return "", err
	}
	cmd := exec.Command("fzf", "--preview-window=hidden")
	cmd.Stdin = strings.NewReader(strings.Join(names, "\n"))
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// menuPick presents a numbered list and prompts for selection on the given
// scanner. Taking the scanner as a parameter (rather than opening a fresh one
// on os.Stdin) matters here: a second bufio.Scanner wrapping the same stdin fd
// buffers independently and can swallow bytes the caller's scanner already
// read ahead.
func menuPick(in *bufio.Scanner, names []string) (string, error) {
	for i, name := range names {
		fmt.Printf("  %s %s\n", style.Dim(fmt.Sprintf("%d)", i+1)), name)
	}
	fmt.Print(style.Bold("Select: "))
	if !in.Scan() {
		return "", fmt.Errorf("cancelled")
	}
	idx, err := strconv.Atoi(strings.TrimSpace(in.Text()))
	if err != nil || idx < 1 || idx > len(names) {
		return "", fmt.Errorf("invalid selection")
	}
	return names[idx-1], nil
}
