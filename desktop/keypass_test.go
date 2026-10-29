package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func makeKey(t *testing.T, dir, name, pass string, extra ...string) string {
	t.Helper()
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not installed")
	}
	path := filepath.Join(dir, name)
	args := append([]string{"-q", "-t", "ed25519", "-N", pass, "-f", path}, extra...)
	if out, err := exec.Command("ssh-keygen", args...).CombinedOutput(); err != nil {
		t.Fatalf("ssh-keygen: %v: %s", err, out)
	}
	return path
}

func TestKeyEncryptedDetectsAPassphrase(t *testing.T) {
	dir := t.TempDir()
	plain := makeKey(t, dir, "plain", "")
	locked := makeKey(t, dir, "locked", "hunter2")
	// the older PEM format marks encryption in its header instead
	pem := makeKey(t, dir, "pem", "hunter2", "-m", "PEM", "-t", "rsa", "-b", "2048")
	for path, want := range map[string]bool{plain: false, locked: true, pem: true} {
		got, err := keyEncrypted(path)
		if err != nil || got != want {
			t.Errorf("keyEncrypted(%s) = %v, %v; want %v", filepath.Base(path), got, err, want)
		}
	}
	if _, err := keyEncrypted(filepath.Join(dir, "missing")); err == nil {
		t.Error("a missing file should be an error")
	}
	junk := filepath.Join(dir, "junk")
	_ = os.WriteFile(junk, []byte("hello"), 0o600)
	if got, _ := keyEncrypted(junk); got {
		t.Error("a file that is not a key must not read as encrypted")
	}
}

// startAgent runs a private ssh-agent so the test never touches the real one, and returns its socket.
func startAgent(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("ssh-agent"); err != nil {
		t.Skip("ssh-agent not installed")
	}
	sock := fmt.Sprintf("/tmp/relay-ag-%d.sock", os.Getpid()) // short: unix socket paths are limited to ~100 bytes
	_ = os.Remove(sock)
	cmd := exec.Command("ssh-agent", "-D", "-a", sock)
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot start ssh-agent: %v", err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait(); _ = os.Remove(sock) })
	for i := 0; i < 50; i++ {
		if _, err := os.Stat(sock); err == nil {
			return sock
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Skip("ssh-agent did not start")
	return ""
}

func agentKeys(t *testing.T, sock string) string {
	t.Helper()
	cmd := exec.Command("ssh-add", "-l")
	cmd.Env = append(os.Environ(), "SSH_AUTH_SOCK="+sock)
	out, _ := cmd.CombinedOutput()
	return string(out)
}

// The passphrase goes to ssh-add through ssh's askpass hook (an environment variable of that one
// child, never an argument, never a file), and a wrong one is refused without loading anything.
func TestRememberPassphraseLoadsTheKeyIntoTheAgent(t *testing.T) {
	dir := t.TempDir()
	locked := makeKey(t, dir, "locked", "hunter2")
	sock := startAgent(t)

	if err := rememberPassphrase(locked, "wrong", false, sock); err == nil {
		t.Fatal("a wrong passphrase was accepted")
	} else if !strings.Contains(err.Error(), "passphrase") {
		t.Errorf("unhelpful error: %v", err)
	}
	if got := agentKeys(t, sock); strings.Contains(got, "ED25519") {
		t.Fatalf("a rejected passphrase still loaded the key:\n%s", got)
	}
	if err := rememberPassphrase(locked, "hunter2", false, sock); err != nil {
		t.Fatal(err)
	}
	if got := agentKeys(t, sock); !strings.Contains(got, "ED25519") {
		t.Errorf("the key is not in the agent:\n%s", got)
	}
	// the secret is not left behind in the helper script
	if data, err := os.ReadFile(askpassPath()); err != nil || strings.Contains(string(data), "hunter2") {
		t.Errorf("askpass helper = %q, %v", data, err)
	}
}
