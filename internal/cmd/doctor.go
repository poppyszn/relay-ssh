package cmd

import (
	"fmt"
	"os"

	"github.com/poppyszn/super-ssh-v2/internal/config"
	"github.com/poppyszn/super-ssh-v2/internal/keys"
	"github.com/poppyszn/super-ssh-v2/internal/paths"
	"github.com/poppyszn/super-ssh-v2/internal/sshconfig"
	"github.com/poppyszn/super-ssh-v2/internal/style"
)

// DoctorCmd checks installation health, optionally applying deterministic fixes with --fix.
func DoctorCmd(p paths.Paths, args ...string) error {
	for _, arg := range args {
		if arg == "--fix" {
			applyFixes(p)
			break
		}
	}

	fmt.Println(style.Header("sssh health check"))
	fmt.Println(style.Dim("──────────────────"))
	var issues int

	s, storeErr := config.Load(p.StoreFile)
	if storeErr != nil {
		reportFail("connections.toml", storeErr.Error())
		issues++
	} else {
		reportOK("connections.toml", fmt.Sprintf("%d profiles", len(s.Profiles)))
	}

	sshConfigData, sshConfigErr := os.ReadFile(p.SSHConfig)
	if sshConfigErr != nil {
		reportFail("~/.ssh/config", "not found")
		issues++
	} else {
		content := string(sshConfigData)
		switch {
		case !sshconfig.HasBlock(content):
			reportWarn("~/.ssh/config", "no managed block (run `sssh sync`)")
		case !sshconfig.BlockIsFirst(content):
			reportFail("~/.ssh/config", "managed block exists but is not at top")
			issues++
		default:
			reportOK("~/.ssh/config", "managed block present and at top")
		}
	}

	if storeErr == nil && len(s.Profiles) > 0 {
		expected := sshconfig.Generate(s)
		genData, genErr := os.ReadFile(p.GenConfig)
		switch {
		case genErr != nil:
			reportFail("generated config", "not found (run `sssh sync`)")
			issues++
		case string(genData) != expected:
			reportWarn("generated config", "out of date (run `sssh sync`)")
		default:
			reportOK("generated config", "up to date")
		}
	}

	if storeErr == nil && len(s.Profiles) > 0 {
		var keyNames []string
		for _, prof := range s.Profiles {
			if prof.Auth == config.AuthKey && prof.Key != "" && !isAbsolutePath(prof.Key) {
				keyNames = append(keyNames, prof.Key)
			}
		}
		if len(keyNames) > 0 {
			missing := keys.ValidateAdopted(p, keyNames)
			if len(missing) == 0 {
				reportOK("adopted keys", fmt.Sprintf("%d keys present", len(keyNames)))
			} else {
				reportFail("adopted keys", fmt.Sprintf("missing %v", missing))
				issues++
			}
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

// applyFixes attempts the two deterministic fixes and reports any failure --
// silently discarding an error here would leave the recheck below failing
// again with no indication of why the "fix" didn't help.
func applyFixes(p paths.Paths) {
	if s, err := config.Load(p.StoreFile); err == nil && len(s.Profiles) > 0 {
		if err := sshconfig.Write(p, s); err != nil {
			fmt.Println(style.Fail("✗ fix: regenerating config failed: " + err.Error()))
		}
	}
	if err := sshconfig.EnsureInclude(p); err != nil {
		fmt.Println(style.Fail("✗ fix: updating ~/.ssh/config failed: " + err.Error()))
	}
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

func isAbsolutePath(p string) bool {
	return len(p) > 0 && (p[0] == '/' || p[0] == '~')
}
