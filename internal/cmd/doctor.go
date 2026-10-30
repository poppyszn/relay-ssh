package cmd

import (
	"fmt"

	"github.com/poppyszn/relay-ssh/internal/core"
	"github.com/poppyszn/relay-ssh/internal/paths"
	"github.com/poppyszn/relay-ssh/internal/style"
)

// DoctorCmd checks installation health, optionally applying deterministic fixes with --fix.
func DoctorCmd(p paths.Paths, args ...string) error {
	for _, arg := range args {
		if arg == "--fix" {
			for _, err := range core.Fix(p) {
				fmt.Println(style.Fail("✗ fix: " + err.Error()))
			}
			break
		}
	}

	fmt.Println(style.Header("relay-ssh health check"))
	fmt.Println(style.Dim("──────────────────"))
	var issues int
	for _, c := range core.Diagnose(p) {
		switch c.Status {
		case core.OK:
			reportOK(c.Name, c.Detail)
		case core.Warn:
			reportWarn(c.Name, c.Detail)
		default:
			reportFail(c.Name, c.Detail)
			issues++
		}
	}

	fmt.Println()
	if issues == 0 {
		fmt.Println(style.Success("Health: OK"))
		return nil
	}
	fmt.Println(style.Fail(fmt.Sprintf("Health: %d issue(s) found", issues)))
	return nil
}

func reportOK(check, detail string) {
	fmt.Printf("  %s %-16s %s\n", style.Success("✓"), check, style.Dim(detail))
}

func reportWarn(check, detail string) {
	fmt.Printf("  %s %-16s %s\n", style.Warn("⚠"), check, detail)
}

func reportFail(check, detail string) {
	fmt.Printf("  %s %-16s %s\n", style.Fail("✗"), check, detail)
}
