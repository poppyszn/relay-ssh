package main

import (
	"os"
	"path/filepath"
)

// Relay draws command "blocks" in the terminal from OSC 133 prompt markers the
// shell prints: A before the prompt, B after it, D;<exit> after a command.
// Go never reads them; the frontend does. For local shells Relay injects the
// hooks itself, sourcing the user's real startup files first and changing none
// of them.

const zshenv = `[ -f "${RELAY_ORIG_ZDOTDIR:-$HOME}/.zshenv" ] && source "${RELAY_ORIG_ZDOTDIR:-$HOME}/.zshenv"
`
const zprofile = `[ -f "${RELAY_ORIG_ZDOTDIR:-$HOME}/.zprofile" ] && source "${RELAY_ORIG_ZDOTDIR:-$HOME}/.zprofile"
`
const zlogin = `[ -f "${RELAY_ORIG_ZDOTDIR:-$HOME}/.zlogin" ] && source "${RELAY_ORIG_ZDOTDIR:-$HOME}/.zlogin"
`
const zshrc = `# Relay shell integration. Sources the real .zshrc first.
[ -f "${RELAY_ORIG_ZDOTDIR:-$HOME}/.zshrc" ] && source "${RELAY_ORIG_ZDOTDIR:-$HOME}/.zshrc"
__relay_precmd() { local e=$?; print -n $'\e]133;D;'"$e"$'\a\e]133;A\a'; }
__relay_ps1() { [[ "$PS1" == *$'\e]133;B\a'* ]] || PS1="${PS1}%{"$'\e]133;B\a'"%}"; }
precmd_functions=(__relay_precmd $precmd_functions __relay_ps1)
if [ -n "$RELAY_ORIG_ZDOTDIR" ]; then ZDOTDIR="$RELAY_ORIG_ZDOTDIR"; else unset ZDOTDIR; fi
unset RELAY_ORIG_ZDOTDIR
`

const bashrc = `# Relay shell integration. Behaves like a login shell, then adds the hooks.
[ -r /etc/profile ] && . /etc/profile
__relay_found=
for f in "$HOME/.bash_profile" "$HOME/.bash_login" "$HOME/.profile"; do
  if [ -r "$f" ]; then . "$f"; __relay_found=1; break; fi
done
[ -z "$__relay_found" ] && [ -r "$HOME/.bashrc" ] && . "$HOME/.bashrc"
unset __relay_found
__relay_prompt() { local e=$?; printf '\033]133;D;%s\007\033]133;A\007' "$e"; }
case "$PROMPT_COMMAND" in *__relay_prompt*) ;; *) PROMPT_COMMAND="__relay_prompt${PROMPT_COMMAND:+;$PROMPT_COMMAND}";; esac
PS1="${PS1}\[\033]133;B\007\]"
`

// remoteSnippet turns the same hooks on in an interactive bash or zsh on a
// remote host. Relay never types it on its own: the user asks for it, and it
// appears in the terminal as an ordinary command. The leading space keeps it
// out of history where HISTCONTROL=ignorespace is set.
const remoteSnippet = ` if [ -n "$ZSH_VERSION" ]; then __r(){ local e=$?; printf '\033]133;D;%s\007\033]133;A\007' "$e"; }; __rp(){ case $PS1 in *133\;B*) ;; *) PS1="$PS1%{"$'\033]133;B\007'"%}";; esac; }; precmd_functions=(__r $precmd_functions __rp); elif [ -n "$BASH_VERSION" ]; then __r(){ local e=$?; printf '\033]133;D;%s\007\033]133;A\007' "$e"; }; PROMPT_COMMAND="__r${PROMPT_COMMAND:+;$PROMPT_COMMAND}"; PS1="$PS1\[\033]133;B\007\]"; else echo "relay: command blocks need bash or zsh"; fi; clear
`

// IntegrationSnippet returns the one-liner for remote shells. Bound to the frontend.
func (t *TerminalService) IntegrationSnippet() string { return remoteSnippet }

// shellCommand returns how to start the user's local shell with block markers
// enabled: argv and extra environment. dir holds the generated startup files.
// Shells other than zsh and bash start plain, without blocks.
func shellCommand(shell, dir string) (argv, env []string) {
	plain := []string{shell}
	write := func(files map[string]string) bool {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return false
		}
		for name, body := range files {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o600); err != nil {
				return false
			}
		}
		return true
	}
	switch filepath.Base(shell) {
	case "zsh":
		if !write(map[string]string{".zshenv": zshenv, ".zprofile": zprofile, ".zshrc": zshrc, ".zlogin": zlogin}) {
			return plain, nil
		}
		return []string{shell, "-l"}, []string{"ZDOTDIR=" + dir, "RELAY_ORIG_ZDOTDIR=" + os.Getenv("ZDOTDIR")}
	case "bash":
		if !write(map[string]string{"bashrc": bashrc}) {
			return plain, nil
		}
		return []string{shell, "--rcfile", filepath.Join(dir, "bashrc"), "-i"}, nil
	}
	return plain, nil
}

// integrationDir is where the generated startup files live.
func integrationDir() string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "relay", "shell-integration")
}
