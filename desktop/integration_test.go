package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// readUntil collects PTY output until it contains all of want or time runs out.
func readUntil(t *testing.T, s *session, want ...string) string {
	t.Helper()
	var mu sync.Mutex
	var out strings.Builder
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := s.pty.Read(buf)
			mu.Lock()
			out.Write(buf[:n])
			mu.Unlock()
			if err != nil {
				return
			}
		}
	}()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		got := out.String()
		mu.Unlock()
		ok := true
		for _, w := range want {
			ok = ok && strings.Contains(got, w)
		}
		if ok {
			return got
		}
		time.Sleep(50 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	return out.String()
}

// The local shell must emit OSC 133 prompt markers without touching the
// user's dotfiles: A before each prompt, B after it, D;<exit> after a command.
func TestShellIntegrationEmitsPromptMarkers(t *testing.T) {
	for _, shell := range []string{"zsh", "bash"} {
		t.Run(shell, func(t *testing.T) {
			path, err := exec.LookPath(shell)
			if err != nil {
				t.Skipf("%s not installed", shell)
			}
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("ZDOTDIR", "")
			argv, env := shellCommand(path, filepath.Join(t.TempDir(), "integration"))
			s, err := startSession(argv, env, 80, 24)
			if err != nil {
				t.Fatal(err)
			}
			defer s.close()
			if _, err := s.pty.Write([]byte("echo hi\nfalse\n")); err != nil {
				t.Fatal(err)
			}
			got := readUntil(t, s, "\x1b]133;D;0\a", "\x1b]133;D;1\a")
			for _, m := range []string{"\x1b]133;A\a", "\x1b]133;B\a", "\x1b]133;D;0\a", "\x1b]133;D;1\a"} {
				if !strings.Contains(got, m) {
					t.Errorf("%s output missing %q:\n%q", shell, m, got)
				}
			}
		})
	}
}
