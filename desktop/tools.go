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
	Name    string `json:"name"`
	Path    string `json:"path"`
	Adopted bool   `json:"adopted"`
	// Encrypted means the key file is protected by a passphrase.
	Encrypted bool     `json:"encrypted"`
	UsedBy    []string `json:"usedBy"`
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

// prepareDirs tightens the folders Relay writes into (0700) at startup, including ones an older
// version made world-readable, instead of waiting for the first write.
func prepareDirs() {
	if p, err := paths.Default(); err == nil {
		_ = paths.EnsureDirs(p)
	}
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
		vault := filepath.Join(p.KeysDir, name)
		enc, _ := keyEncrypted(vault)
		out = append(out, KeyInfo{Name: name, Path: vault, Adopted: true, Encrypted: enc, UsedBy: usedBy(name)})
	}
	for _, path := range core.FindKeyFiles(home) {
		name := filepath.Base(path)
		if seen[name] {
			continue // already adopted under this name
		}
		tilde := "~/.ssh/" + name
		enc, _ := keyEncrypted(path)
		out = append(out, KeyInfo{Name: name, Path: path, Encrypted: enc, UsedBy: usedBy(path, tilde)})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Adopted != out[j].Adopted {
			return out[i].Adopted
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// KeyNeedsPassphrase reports whether the private key at path is protected by a passphrase.
func (ToolsService) KeyNeedsPassphrase(path string) bool {
	enc, _ := keyEncrypted(path)
	return enc
}

// RememberPassphrase loads an adopted key into the ssh-agent (and, on macOS, has the Keychain
// remember its passphrase) so ssh stops asking for it. Needed again on a new machine after a restore.
func (ToolsService) RememberPassphrase(name, passphrase string) error {
	p, err := paths.Default()
	if err != nil {
		return err
	}
	return rememberPassphrase(filepath.Join(p.KeysDir, filepath.Base(name)), passphrase, keychainEnabled, "")
}

// AdoptKey copies a private key into the vault so backups include it, and
// points every profile that referenced the original path at the adopted copy.
// A key with a passphrase needs it: it is loaded into the ssh-agent (and the macOS Keychain) so ssh
// does not ask each time. Relay itself never stores the passphrase. If the passphrase is wrong,
// nothing is adopted.
func (ToolsService) AdoptKey(path, passphrase string) error {
	p, err := paths.Default()
	if err != nil {
		return err
	}
	if err := paths.EnsureDirs(p); err != nil {
		return err
	}
	name := filepath.Base(path)
	encrypted, err := keyEncrypted(path)
	if err != nil {
		return err
	}
	if encrypted && passphrase == "" {
		return errors.New("This key has a passphrase. Enter it so ssh does not have to ask for it each time.")
	}
	if _, err := keys.Adopt(p, path, name); err != nil {
		return err
	}
	if encrypted {
		vault := filepath.Join(p.KeysDir, name)
		if err := rememberPassphrase(vault, passphrase, keychainEnabled, ""); err != nil {
			os.Remove(vault) // nothing is adopted on a wrong passphrase
			return err
		}
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
