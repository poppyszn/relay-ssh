package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/core"
	"github.com/poppyszn/relay-ssh/internal/keys"
	"github.com/poppyszn/relay-ssh/internal/paths"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// ToolsService is the everyday-tools surface: health checks, backup and
// restore, ssh_config import and key management. It is a thin layer over
// internal/core, so the CLI behaves identically.
type ToolsService struct{}

type HealthCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"` // ok | warn | fail
	Detail string `json:"detail"`
}

type FixReport struct {
	Errors []string      `json:"errors"`
	Checks []HealthCheck `json:"checks"`
}

type ImportReport struct {
	Added   []string `json:"added"`
	Skipped []string `json:"skipped"`
}

type KeyInfo struct {
	Name    string   `json:"name"`
	Path    string   `json:"path"`
	Adopted bool     `json:"adopted"`
	UsedBy  []string `json:"usedBy"`
}

// setup resolves the paths and makes sure the directories exist, for methods
// that write. The CLI does this at startup; the app has no single entry point.
func setup() (paths.Paths, error) {
	p, err := paths.Default()
	if err != nil {
		return p, err
	}
	return p, paths.EnsureDirs(p)
}

func toChecks(cs []core.Check) []HealthCheck {
	out := make([]HealthCheck, 0, len(cs))
	for _, c := range cs {
		st := "ok"
		switch c.Status {
		case core.Warn:
			st = "warn"
		case core.Fail:
			st = "fail"
		}
		out = append(out, HealthCheck{c.Name, st, c.Detail})
	}
	return out
}

// Health runs every doctor check without changing anything.
func (ToolsService) Health() ([]HealthCheck, error) {
	p, err := paths.Default()
	if err != nil {
		return nil, err
	}
	return toChecks(core.Diagnose(p)), nil
}

// FixHealth applies the deterministic fixes, then re-checks.
func (ToolsService) FixHealth() (FixReport, error) {
	p, err := setup()
	if err != nil {
		return FixReport{}, err
	}
	var rep FixReport
	for _, e := range core.Fix(p) {
		rep.Errors = append(rep.Errors, e.Error())
	}
	rep.Checks = toChecks(core.Diagnose(p))
	return rep, nil
}

// DefaultBackupName suggests a dated file name.
func (ToolsService) DefaultBackupName() string {
	return "relay-backup-" + time.Now().Format("2006-01-02") + ".tar.gz.age"
}

// Backup writes an encrypted backup of the connections and adopted keys to path.
// The passphrase is used once and never stored.
func (ToolsService) Backup(path, passphrase string) error {
	if passphrase == "" {
		return errors.New("a passphrase is required")
	}
	if path == "" {
		return errors.New("no destination chosen")
	}
	if !strings.HasSuffix(path, ".age") {
		path += ".age"
	}
	p, err := paths.Default()
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("creating backup file: %w", err)
	}
	if err := core.Backup(p, f, passphrase); err != nil {
		f.Close()
		os.Remove(path) // never leave a half-written backup behind
		return err
	}
	return f.Close()
}

// NeedsPassphrase reports whether a backup file is encrypted.
func (ToolsService) NeedsPassphrase(path string) bool { return strings.HasSuffix(path, ".age") }

// Restore replaces the current connections and adopted keys with the backup's.
func (ToolsService) Restore(path, passphrase string) error {
	p, err := paths.Default()
	if err != nil {
		return err
	}
	if err := paths.EnsureDirs(p); err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening backup: %w", err)
	}
	defer f.Close()
	enc := strings.HasSuffix(path, ".age")
	if enc && passphrase == "" {
		return errors.New("this backup is encrypted: enter its passphrase")
	}
	return core.Restore(p, f, passphrase, enc)
}

// ImportSSHConfig turns the Host entries of ~/.ssh/config into profiles.
func (ToolsService) ImportSSHConfig() (ImportReport, error) {
	p, err := setup()
	if err != nil {
		return ImportReport{}, err
	}
	f, err := os.Open(p.SSHConfig)
	if err != nil {
		return ImportReport{}, fmt.Errorf("reading %s: %w", p.SSHConfig, err)
	}
	defer f.Close()
	res, err := core.ImportSSHConfigDetailed(p, f)
	return ImportReport{Added: nonNil(res.Added), Skipped: nonNil(res.Skipped)}, err
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// Keys lists the keys Relay has adopted into the vault (covered by backups)
// and the private keys found in ~/.ssh, with the profiles that use each.
func (ToolsService) Keys() ([]KeyInfo, error) {
	p, err := paths.Default()
	if err != nil {
		return nil, err
	}
	s, err := config.Load(p.StoreFile)
	if err != nil {
		return nil, err
	}
	home := filepath.Dir(p.SSHConfig)
	usedBy := func(refs ...string) []string {
		out := []string{}
		for _, prof := range s.Profiles {
			for _, r := range refs {
				if prof.Auth == config.AuthKey && prof.Key == r {
					out = append(out, prof.Name)
					break
				}
			}
		}
		return out
	}

	var out []KeyInfo
	adopted, err := keys.List(p)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, name := range adopted {
		seen[name] = true
		out = append(out, KeyInfo{Name: name, Path: filepath.Join(p.KeysDir, name), Adopted: true, UsedBy: usedBy(name)})
	}
	for _, path := range core.FindKeyFiles(home) {
		name := filepath.Base(path)
		if seen[name] {
			continue // already adopted under this name
		}
		tilde := "~/.ssh/" + name
		out = append(out, KeyInfo{Name: name, Path: path, UsedBy: usedBy(path, tilde)})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Adopted != out[j].Adopted {
			return out[i].Adopted
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// AdoptKey copies a private key into the vault so backups include it, and
// points every profile that referenced the original path at the adopted copy.
func (ToolsService) AdoptKey(path string) error {
	p, err := paths.Default()
	if err != nil {
		return err
	}
	if err := paths.EnsureDirs(p); err != nil {
		return err
	}
	name := filepath.Base(path)
	if _, err := keys.Adopt(p, path, name); err != nil {
		return err
	}
	s, err := config.Load(p.StoreFile)
	if err != nil {
		return err
	}
	home, _ := os.UserHomeDir()
	refs := map[string]bool{path: true, "~/.ssh/" + name: filepath.Dir(path) == filepath.Join(home, ".ssh")}
	changed := false
	for i, prof := range s.Profiles {
		if prof.Auth == config.AuthKey && refs[prof.Key] {
			s.Profiles[i].Key = name
			changed = true
		}
	}
	if !changed {
		return nil
	}
	if err := config.Save(p.StoreFile, s); err != nil {
		return err
	}
	return core.Regenerate(p, s)
}

// PickBackupPath opens the native save dialog. Empty if cancelled.
func (t ToolsService) PickBackupPath() (string, error) {
	return application.Get().Dialog.SaveFile().
		SetMessage("Save encrypted backup").
		SetFilename(t.DefaultBackupName()).
		CanCreateDirectories(true).
		PromptForSingleSelection()
}

// PickRestorePath opens the native open dialog. Empty if cancelled.
func (ToolsService) PickRestorePath() (string, error) {
	return application.Get().Dialog.OpenFile().
		SetTitle("Choose a backup to restore").
		CanChooseFiles(true).
		PromptForSingleSelection()
}

// PickKeyToAdopt opens the native open dialog on ~/.ssh. Empty if cancelled.
func (ToolsService) PickKeyToAdopt() (string, error) {
	home, _ := os.UserHomeDir()
	return application.Get().Dialog.OpenFile().
		SetTitle("Choose a private key to adopt").
		SetDirectory(filepath.Join(home, ".ssh")).
		CanChooseFiles(true).ShowHiddenFiles(true).
		PromptForSingleSelection()
}
