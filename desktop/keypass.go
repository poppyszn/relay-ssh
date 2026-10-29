package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/poppyszn/relay-ssh/internal/paths"
)

// keyEncrypted reports whether a private key file is protected by a passphrase, from its
// header alone: the passphrase is never needed to know.
func keyEncrypted(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	text := string(data)
	switch {
	case strings.Contains(text, "BEGIN OPENSSH PRIVATE KEY"):
		return openSSHCipher(text) != "none", nil
	case strings.Contains(text, "BEGIN ENCRYPTED PRIVATE KEY"), strings.Contains(text, "ENCRYPTED"):
		return true, nil // PKCS#8 and the older PEM "Proc-Type: 4,ENCRYPTED"
	}
	return false, nil
}

// openSSHCipher reads the cipher name from an OpenSSH-format key ("none" when unprotected).
func openSSHCipher(text string) string {
	var b64 strings.Builder
	for _, line := range strings.Split(text, "\n") {
		if line = strings.TrimSpace(line); line != "" && !strings.HasPrefix(line, "-----") {
			b64.WriteString(line)
		}
	}
	raw, err := base64.StdEncoding.DecodeString(b64.String())
	const magic = "openssh-key-v1\x00"
	if err != nil || !bytes.HasPrefix(raw, []byte(magic)) || len(raw) < len(magic)+4 {
		return "none"
	}
	rest := raw[len(magic):]
	n := binary.BigEndian.Uint32(rest)
	if int(n) > len(rest)-4 {
		return "none"
	}
	return string(rest[4 : 4+n])
}

// askpassPath is a tiny script ssh runs to ask for a passphrase. It holds no secret: it prints
// the one in the RELAY_PASSPHRASE variable of the single ssh-add process that runs it.
func askpassPath() string {
	base, err := os.UserCacheDir()
	if err != nil {
		base = os.TempDir()
	}
	return filepath.Join(base, "relay", "askpass.sh")
}

// It answers once: ssh-add asks again after a wrong passphrase, forever if the answer never
// changes, so a second question is refused (the marker file is a per-run path of the caller).
const askpassScript = `#!/bin/sh
[ -e "$RELAY_ASKPASS_USED" ] && exit 1
: > "$RELAY_ASKPASS_USED"
printf '%s\n' "$RELAY_PASSPHRASE"
`

// rememberPassphrase loads an encrypted key into the ssh-agent so ssh stops asking for its
// passphrase. On macOS it also asks the system Keychain to remember the passphrase (the OS
// stores it; Relay does not), which lets ssh unlock the key again after a restart. The
// passphrase reaches ssh-add only through ssh's askpass hook, as an environment variable of
// that one child process. `agent` overrides SSH_AUTH_SOCK (for tests).
func rememberPassphrase(keyPath, passphrase string, keychain bool, agent string) error {
	script := askpassPath()
	if err := os.MkdirAll(filepath.Dir(script), 0o700); err != nil {
		return err
	}
	if err := os.WriteFile(script, []byte(askpassScript), 0o700); err != nil {
		return err
	}
	args := []string{keyPath}
	if keychain {
		args = append([]string{"--apple-use-keychain"}, args...)
	}
	used, err := os.CreateTemp("", "relay-askpass-*")
	if err != nil {
		return err
	}
	used.Close()
	os.Remove(used.Name()) // the script creates it when it answers
	defer os.Remove(used.Name())
	ctx, cancel := contextWithTimeout(20 * time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ssh-add", args...)
	cmd.Env = append(os.Environ(), "SSH_ASKPASS="+script, "SSH_ASKPASS_REQUIRE=force", "RELAY_PASSPHRASE="+passphrase, "RELAY_ASKPASS_USED="+used.Name())
	if agent != "" {
		cmd.Env = append(cmd.Env, "SSH_AUTH_SOCK="+agent)
	}
	cmd.Stdin = nil
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	msg := strings.TrimSpace(string(out))
	_, asked := os.Stat(used.Name()) // the script ran, so ssh-add got as far as unlocking the key
	switch {
	case strings.Contains(msg, "Could not open a connection to your authentication agent"):
		return errors.New("There is no ssh-agent to hold the key. Start one, or sign out and back in.")
	case asked == nil, strings.Contains(msg, "passphrase"):
		return errors.New("That passphrase is not right for this key.")
	case runtime.GOOS == "darwin" && keychain && strings.Contains(msg, "illegal option"):
		return errors.New("This macOS cannot save the passphrase in the Keychain.")
	}
	return fmt.Errorf("ssh-add failed: %s", msg)
}

func contextWithTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}

// keychainEnabled says whether to ask the macOS Keychain to keep passphrases (a variable so tests can switch it off).
var keychainEnabled = runtime.GOOS == "darwin"

// keyIsEncrypted resolves a profile's key setting (a vault key name, a ~/ path or a full path) and
// reports whether that key has a passphrase.
func keyIsEncrypted(p paths.Paths, key string) bool {
	path := key
	switch {
	case strings.HasPrefix(key, "~/"):
		path = filepath.Join(filepath.Dir(p.SSHConfig), strings.TrimPrefix(key, "~/.ssh/"))
	case !filepath.IsAbs(key):
		path = filepath.Join(p.KeysDir, key)
	}
	enc, _ := keyEncrypted(path)
	return enc
}
