// Command relay-ssh manages SSH connection profiles and launches the real ssh
// binary. It is never running while a session is open: connecting replaces
// this process via exec.
package main

import (
	"fmt"
	"os"

	"github.com/poppyszn/relay-ssh/internal/cmd"
	"github.com/poppyszn/relay-ssh/internal/core"
	"github.com/poppyszn/relay-ssh/internal/paths"
)

// version is the tool version, overridden at release time with
// -ldflags "-X main.version=v1.2.3".
var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "relay-ssh:", err)
		os.Exit(1)
	}
}

// migratedPaths is paths.Default, after moving anything left under the tool's old name (sssh).
func migratedPaths() (paths.Paths, error) {
	p, err := paths.Default()
	if err != nil {
		return p, err
	}
	return p, core.Migrate(p)
}

func run(args []string) error {
	if len(args) == 0 {
		p, err := migratedPaths()
		if err != nil {
			return err
		}
		name, err := cmd.Pick(p)
		if err != nil {
			return err
		}
		return cmd.Connect(p, name)
	}
	switch args[0] {
	case "version", "--version", "-v":
		fmt.Println("relay-ssh " + version)
		return nil
	case "help", "--help", "-h":
		fmt.Print(usage)
		return nil
	}
	p, err := migratedPaths()
	if err != nil {
		return err
	}
	switch args[0] {
	case "ls":
		return cmd.Ls(p, args[1:])
	case "add":
		return cmd.Add(p, args[1:])
	case "rm":
		return cmd.Rm(p, args[1:])
	case "sync":
		return cmd.Sync(p)
	case "edit":
		if len(args) < 2 {
			return fmt.Errorf("edit requires a profile name")
		}
		return cmd.EditCmd(p, args[1:])
	case "cp":
		if len(args) < 2 {
			return fmt.Errorf("cp requires profile:src local-dst")
		}
		return cmd.CpCmd(p, args[1:])
	case "import":
		return cmd.ImportCmd(p, args[1:])
	case "backup":
		return cmd.BackupCmd(p, args[1:])
	case "restore":
		if len(args) < 2 {
			return fmt.Errorf("restore requires an archive file")
		}
		return cmd.RestoreCmd(p, args[1:])
	case "doctor":
		return cmd.DoctorCmd(p, args[1:]...)
	case "completion":
		if len(args) < 2 {
			return fmt.Errorf("completion requires a shell name (bash or zsh)")
		}
		script, err := cmd.CompletionScript(args[1])
		if err != nil {
			return err
		}
		fmt.Print(script)
		return nil
	case "__complete-profiles":
		// Undocumented on purpose: this is plumbing the completion scripts
		// call at completion time, not a command a user runs directly.
		addrs, err := cmd.ListProfileAddresses(p)
		if err != nil {
			return nil // never surface an error mid-completion
		}
		for _, a := range addrs {
			fmt.Println(a)
		}
		return nil
	}
	// Assume it's a profile name, connect to it
	return cmd.Connect(p, args[0])
}

const usage = `relay-ssh — SSH connection manager

Usage:
  relay-ssh                    pick a connection and connect
  relay-ssh <name>             connect to a saved profile
  relay-ssh ls [--tree]        list profiles
  relay-ssh add                add a profile
  relay-ssh edit <name>        edit a profile
  relay-ssh rm <name>          remove a profile
  relay-ssh cp <name>:/p ./l   copy files with scp
  relay-ssh import             adopt hosts from ~/.ssh/config
  relay-ssh backup [-o FILE]   write an encrypted archive
  relay-ssh restore FILE       restore from an encrypted archive
  relay-ssh doctor [--fix]     check the installation (or fix deterministic issues)
  relay-ssh sync               regenerate ~/.ssh/relay-ssh.d/config
  relay-ssh completion <shell> print a completion script (bash or zsh)
  relay-ssh version            print the version

Tab completion (zsh):
  relay-ssh completion zsh > "${fpath[1]}/_relay-ssh" && exec zsh

Tab completion (bash):
  echo 'source <(relay-ssh completion bash)' >> ~/.bashrc
`
