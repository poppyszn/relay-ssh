package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"github.com/poppyszn/relay-ssh/internal/core"
	"github.com/poppyszn/relay-ssh/internal/style"
)

const manualEntryOption = "Type a path manually..."

// PickKeyFile browses ~/.ssh for private key files and lets the user choose
// one, with fzf if available, else a numbered menu. If none are found, or the
// user picks "type a path manually", it falls back to a plain prompt.
func PickKeyFile(in *bufio.Scanner) (string, error) {
	candidates := core.FindKeyFiles(sshHomeDir())
	if len(candidates) == 0 {
		val, ok := scanLine(in, "Key file or name")
		if !ok {
			return "", errCancelled
		}
		return val, nil
	}

	options := append(append([]string{}, candidates...), manualEntryOption)
	fmt.Println(style.Header("Select a key:"))
	choice, err := tryFzf(options)
	if err != nil {
		choice, err = menuPick(in, options)
		if err != nil {
			return "", err
		}
	}
	if choice == manualEntryOption {
		val, ok := scanLine(in, "Key file or name")
		if !ok {
			return "", errCancelled
		}
		return val, nil
	}
	return choice, nil
}

func sshHomeDir() string {
	return filepath.Join(os.Getenv("HOME"), ".ssh")
}
