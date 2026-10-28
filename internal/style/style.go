// Package style adds ANSI color and box-drawing to terminal output.
//
// No dependency is used: golang.org/x/term is on the guardrails blocklist
// (banned alongside the other TUI/terminal libraries v1 died from), so
// "is this a terminal" is answered by hand via os.ModeCharDevice.
package style

import "os"

// Enabled controls whether Paint functions emit escape codes. Computed once
// from NO_COLOR and whether stdout is a terminal; tests may override it.
var Enabled = computeEnabled()

func computeEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

const (
	reset   = "\033[0m"
	boldC   = "\033[1m"
	dimC    = "\033[2m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	cyan    = "\033[36m"
	magenta = "\033[35m"
)

func paint(code, s string) string {
	if !Enabled {
		return s
	}
	return code + s + reset
}

// Success paints text green, for confirmations.
func Success(s string) string { return paint(green, s) }

// Fail paints text red, for errors and failed checks.
func Fail(s string) string { return paint(red, s) }

// Warn paints text yellow, for things that work but need attention.
func Warn(s string) string { return paint(yellow, s) }

// Bold paints text bold with no color, for emphasis.
func Bold(s string) string { return paint(boldC, s) }

// Dim paints text dim, for secondary detail (counts, hints).
func Dim(s string) string { return paint(dimC, s) }

// Header paints text bold cyan, for section titles.
func Header(s string) string { return paint(boldC+cyan, s) }

// Accent paints text magenta, for folder names and groupings.
func Accent(s string) string { return paint(magenta, s) }
