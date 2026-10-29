package main

import (
	"os"
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
			// A blank row after the command (before the next prompt marker) and a
			// leading blank row in the prompt give each block vertical padding.
			if !strings.Contains(got, "\x1b]133;D;0\a\r\n\x1b]133;A;") {
				t.Errorf("%s: no blank row between the finish marker and the next prompt:\n%q", shell, got)
			}
			// Everything between A and the end of the prompt (B) must include a newline.
			if i, j := strings.Index(got, "\x1b]133;A;"), strings.Index(got, "\x1b]133;B\a"); i < 0 || j < i || !strings.Contains(got[i:j], "\n") {
				t.Errorf("%s: prompt does not start with a blank row:\n%q", shell, got)
			}
			for _, m := range []string{"\x1b]133;A;", "\x1b]133;B\a", "\x1b]133;D;0\a", "\x1b]133;D;1\a"} {
				if !strings.Contains(got, m) {
					t.Errorf("%s output missing %q:\n%q", shell, m, got)
				}
			}
		})
	}
}

// The prompt marker carries where the next command will run, as "A;<branch>;<cwd>",
// for the block header. The cwd is shown as ~ under the home directory (and only
// under it: a sibling that merely starts with the same text is not abbreviated),
// and nothing a directory is named can inject terminal sequences.
func TestShellIntegrationReportsBranchAndDirectory(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	clearGitEnv(t)
	for _, shell := range []string{"zsh", "bash"} {
		t.Run(shell, func(t *testing.T) {
			path, err := exec.LookPath(shell)
			if err != nil {
				t.Skipf("%s not installed", shell)
			}
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("ZDOTDIR", "")
			other := home + "-other"
			for _, d := range []string{home + "/proj", other, home + "/ev\x1bil;x"} {
				if err := os.MkdirAll(d, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			t.Cleanup(func() { os.RemoveAll(other) })
			argv, env := shellCommand(path, filepath.Join(t.TempDir(), "integration"))
			s, err := startSession(argv, env, 80, 24)
			if err != nil {
				t.Fatal(err)
			}
			defer s.close()
			cmds := "cd ~/proj\ngit init -q -b trunk .\ncd '" + other + "'\ncd ~/$'ev\\033il;x'\n"
			if _, err := s.pty.Write([]byte(cmds)); err != nil {
				t.Fatal(err)
			}
			// zsh already prints a control character as ^[ itself; bash gets it stripped.
			evil := "~/evil;x"
			if shell == "zsh" {
				evil = "~/ev^[il;x"
			}
			want := []string{"\x1b]133;A;;~/proj\a", "\x1b]133;A;trunk;~/proj\a", "\x1b]133;A;;" + other + "\a", "\x1b]133;A;;" + evil + "\a"}
			got := readUntil(t, s, want...)
			for _, w := range want {
				if !strings.Contains(got, w) {
					t.Errorf("%s: missing %q in:\n%q", shell, w, got)
				}
			}
		})
	}
}

// The snippet a user pastes into a remote shell defines the same hooks.
func TestRemoteSnippetReportsDirectory(t *testing.T) {
	clearGitEnv(t)
	for _, shell := range []string{"zsh", "bash"} {
		t.Run(shell, func(t *testing.T) {
			path, err := exec.LookPath(shell)
			if err != nil {
				t.Skipf("%s not installed", shell)
			}
			home := t.TempDir()
			if err := os.Mkdir(home+"/work", 0o755); err != nil {
				t.Fatal(err)
			}
			snippet := strings.TrimSuffix(remoteSnippet, "; clear\n")
			cmd := exec.Command(path, "-c", snippet+"; cd ~/work; __r")
			cmd.Env = append(os.Environ(), "HOME="+home, "PS1=$ ")
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("%v: %s", err, out)
			}
			if want := "\x1b]133;D;0\a\n\x1b]133;A;;~/work\a"; !strings.Contains(string(out), want) {
				t.Errorf("%s: want %q in %q", shell, want, out)
			}
		})
	}
}

// clearGitEnv removes the variables git sets for its hooks (GIT_DIR and friends), which would
// point the test's own git commands at the repository that is running the tests, for example when
// `git push` runs the pre-push hook from a linked worktree.
func clearGitEnv(t *testing.T) {
	t.Helper()
	for _, kv := range os.Environ() {
		name, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(name, "GIT_") {
			old, _ := os.LookupEnv(name)
			os.Unsetenv(name)
			t.Cleanup(func() { os.Setenv(name, old) })
		}
	}
}
