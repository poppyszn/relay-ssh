package style

import (
	"strings"
	"testing"
)

func TestPaintingRespectsEnabled(t *testing.T) {
	old := Enabled
	defer func() { Enabled = old }()

	Enabled = true
	got := Success("ok")
	if got == "ok" || !strings.Contains(got, "ok") {
		t.Errorf("Success with Enabled=true should wrap text, got %q", got)
	}
	if !strings.HasPrefix(got, "\033[") {
		t.Errorf("Success with Enabled=true should start with an escape code, got %q", got)
	}

	Enabled = false
	if got := Success("ok"); got != "ok" {
		t.Errorf("Success with Enabled=false should return plain text, got %q", got)
	}
}

func TestAllPaintersPassThroughWhenDisabled(t *testing.T) {
	old := Enabled
	Enabled = false
	defer func() { Enabled = old }()

	fns := map[string]func(string) string{
		"Success": Success, "Fail": Fail, "Warn": Warn,
		"Bold": Bold, "Dim": Dim, "Header": Header, "Accent": Accent,
	}
	for name, fn := range fns {
		if got := fn("text"); got != "text" {
			t.Errorf("%s with Enabled=false = %q, want unmodified 'text'", name, got)
		}
	}
}

func TestAllPaintersWrapWhenEnabled(t *testing.T) {
	old := Enabled
	Enabled = true
	defer func() { Enabled = old }()

	fns := map[string]func(string) string{
		"Success": Success, "Fail": Fail, "Warn": Warn,
		"Bold": Bold, "Dim": Dim, "Header": Header, "Accent": Accent,
	}
	for name, fn := range fns {
		got := fn("text")
		if !strings.Contains(got, "text") {
			t.Errorf("%s lost the original text: %q", name, got)
		}
		if !strings.HasSuffix(got, reset) {
			t.Errorf("%s did not terminate with reset code: %q", name, got)
		}
		if got == "text" {
			t.Errorf("%s with Enabled=true should wrap, got unmodified text", name)
		}
	}
}
