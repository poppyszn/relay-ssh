package cmd

import (
	"bufio"
	"strings"
	"testing"
)

// TestIsInteractiveStdinFalseUnderTest confirms the premise the rest of this
// file's tests rely on: go test's stdin (and any piped/redirected stdin,
// which is exactly what feedStdin uses everywhere else in this package) is
// never a terminal, so disableEcho's real stty invocation never actually
// runs during the test suite -- these tests exercise the fallback path, not
// a mocked one.
func TestIsInteractiveStdinFalseUnderTest(t *testing.T) {
	if isInteractiveStdin() {
		t.Skip("stdin is an actual terminal in this environment; the fallback-path premise doesn't hold here")
	}
}

func TestDisableEchoNoOpsWhenStdinNotInteractive(t *testing.T) {
	restore, disabled := disableEcho()
	if disabled {
		t.Fatal("disableEcho reported disabled=true with non-interactive stdin; it should never shell out to stty here")
	}
	restore() // must be safe to call even though nothing was disabled
}

func TestScanHiddenLineFallsBackToPlainReadWhenNotATerminal(t *testing.T) {
	in := bufio.NewScanner(strings.NewReader("my-secret\n"))
	val, ok := scanHiddenLine(in, "Passphrase")
	if !ok {
		t.Fatal("scanHiddenLine returned ok=false for valid piped input")
	}
	if val != "my-secret" {
		t.Errorf("scanHiddenLine = %q, want my-secret", val)
	}
}

func TestScanHiddenLineReturnsFalseOnExhaustedInput(t *testing.T) {
	in := bufio.NewScanner(strings.NewReader(""))
	_, ok := scanHiddenLine(in, "Passphrase")
	if ok {
		t.Error("scanHiddenLine should return ok=false on exhausted input")
	}
}
