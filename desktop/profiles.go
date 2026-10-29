package main

import (
	"errors"
	"fmt"
	"github.com/poppyszn/relay-ssh/desktop/vault"
	"os"
	"path/filepath"
	"strings"

	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/core"
	"github.com/poppyszn/relay-ssh/internal/paths"
	"github.com/poppyszn/relay-ssh/internal/sshconfig"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// TunnelDTO and Profile are the JSON shapes the frontend sees. They mirror
// config.Profile, which carries TOML tags only.
type TunnelDTO struct {
	Type   string `json:"type"`
	Listen string `json:"listen"`
	Target string `json:"target"`
}

type Profile struct {
	Name         string      `json:"name"`
	Host         string      `json:"host"`
	User         string      `json:"user"`
	Port         int         `json:"port"`
	Auth         string      `json:"auth"`
	Key          string      `json:"key"`
	Folder       string      `json:"folder"`
	Tags         []string    `json:"tags"`
	Jump         string      `json:"jump"`
	ForwardAgent bool        `json:"forwardAgent"`
	Blocks       string      `json:"blocks"`
	Tunnels      []TunnelDTO `json:"tunnels"`
}

// ProfilesService exposes the connection store to the frontend. Every change
// goes through internal/core, so the CLI sees it immediately.
type ProfilesService struct{}

func toProfile(c config.Profile) Profile {
	p := Profile{
		Name: c.Name, Host: c.Host, User: c.User, Port: c.Port, Auth: string(c.Auth),
		Key: c.Key, Folder: c.Folder, Tags: c.Tags, Jump: c.Jump, ForwardAgent: c.ForwardAgent, Blocks: c.Blocks,
		Tunnels: []TunnelDTO{},
	}
	if p.Tags == nil {
		p.Tags = []string{}
	}
	for _, t := range c.Tunnels {
		p.Tunnels = append(p.Tunnels, TunnelDTO{string(t.Type), t.Listen, t.Target})
	}
	return p
}

func fromProfile(p Profile) config.Profile {
	c := config.Profile{
		Name: p.Name, Host: p.Host, User: p.User, Port: p.Port, Auth: config.AuthType(p.Auth),
		Key: p.Key, Folder: p.Folder, Tags: p.Tags, Jump: p.Jump, ForwardAgent: p.ForwardAgent, Blocks: p.Blocks,
	}
	for _, t := range p.Tunnels {
		c.Tunnels = append(c.Tunnels, config.Tunnel{Type: config.TunnelType(t.Type), Listen: t.Listen, Target: t.Target})
	}
	return c
}

// List returns every saved profile, sorted by name.
func (ProfilesService) List() ([]Profile, error) {
	p, err := paths.Default()
	if err != nil {
		return nil, err
	}
	s, err := config.Load(p.StoreFile)
	if err != nil {
		return nil, err
	}
	out := make([]Profile, 0, len(s.Profiles))
	for _, c := range s.Profiles {
		out = append(out, toProfile(c))
	}
	return out, nil
}

// Save adds a profile (oldName == "") or updates and possibly renames one.
func (ps ProfilesService) Save(oldName string, prof Profile) error {
	return ps.save(oldName, prof, "")
}

// SaveWithPassword saves the profile and keeps its password in the vault, bound to the
// connection's host, user and port. It checks the password and the vault first, so a refusal
// leaves nothing half-saved.
func (ps ProfilesService) SaveWithPassword(oldName string, prof Profile, password string) error {
	if prof.Auth != "password" {
		return errors.New("passwords are only kept for connections that use password auth")
	}
	if !vault.ValidSecret(password) {
		return vault.ErrBadSecret
	}
	if !getVault().Status().Unlocked {
		return vault.ErrVaultLocked
	}
	return ps.save(oldName, prof, password)
}

func (ProfilesService) save(oldName string, prof Profile, password string) error {
	p, err := paths.Default()
	if err != nil {
		return err
	}
	if err := paths.EnsureDirs(p); err != nil {
		return err
	}
	if msg := keyProblem(p, oldName, prof); msg != "" {
		return errors.New(msg)
	}
	oldBinding := ""
	if oldName != "" {
		if store, err := config.Load(p.StoreFile); err == nil {
			if old, ok := core.ResolveProfile(store, oldName); ok {
				oldBinding = bindingOf(old)
			}
		}
	}
	if err := core.SaveProfile(p, oldName, fromProfile(prof)); err != nil {
		return err
	}

	// Keep the vault in step. A locked vault cannot be touched, so these are best effort
	// (the form asks to unlock before a rename); an entry left behind is inert, because it is
	// only released for the connection details it was saved with.
	v := getVault()
	newBinding := bindingOf(fromProfile(prof))
	if prof.Auth != "password" {
		_ = v.Forget(prof.Name)
		if oldName != "" {
			_ = v.Forget(oldName)
		}
	} else if oldName != "" {
		// An edit made here moves the binding along only if it was intact; see Vault.Rename.
		_ = v.Rename(oldName, prof.Name, oldBinding, newBinding)
	}
	if password != "" {
		if err := v.Set(prof.Name, password, newBinding); err != nil {
			return fmt.Errorf("the connection was saved but its password was not stored: %w", err)
		}
	}
	return nil
}

// Delete removes a profile by name.
func (ProfilesService) Delete(name string) error {
	p, err := paths.Default()
	if err != nil {
		return err
	}
	if _, err := core.RemoveProfile(p, name); err != nil {
		return err
	}
	_ = getVault().Forget(name)
	return nil
}

// PreviewResult is the form's live preview: the exact Host block the profile
// would generate, and the reason it cannot be saved yet, if any.
type PreviewResult struct {
	Config  string `json:"config"`
	Problem string `json:"problem"`
}

// Preview renders the generated SSH config for prof (replacing oldName, or
// added when oldName is empty) without writing anything.
func (ProfilesService) Preview(oldName string, prof Profile) PreviewResult {
	cand := fromProfile(prof)
	// Generate has no validation of its own, so the preview renders even
	// while the form is half filled in; Problem says what blocks saving.
	block := sshconfig.Generate(config.Store{Version: config.Version, Profiles: []config.Profile{cand}})
	if i := strings.Index(block, "Host "); i >= 0 {
		block = block[i:]
	}
	res := PreviewResult{Config: strings.TrimRight(block, "\n")}

	p, err := paths.Default()
	if err != nil {
		res.Problem = err.Error()
		return res
	}
	s, err := config.Load(p.StoreFile)
	if err != nil {
		res.Problem = err.Error()
		return res
	}
	var profiles []config.Profile
	for _, cur := range s.Profiles {
		if cur.Name != oldName {
			profiles = append(profiles, cur)
		}
	}
	s.Profiles = append(profiles, cand)
	if err := s.Validate(); err != nil {
		res.Problem = err.Error()
	} else if msg := keyProblem(p, oldName, prof); msg != "" {
		res.Problem = msg
	}
	return res
}

// keyProblem explains a key setting that ssh cannot use. A bare key name means a key adopted into
// Relay's vault; if that file is not there ssh fails with "no such identity". Only a new profile or a
// changed key is checked, so a connection already saved that way can still be edited.
func keyProblem(p paths.Paths, oldName string, prof Profile) string {
	key := prof.Key
	if prof.Auth != "key" || key == "" || filepath.IsAbs(key) || strings.HasPrefix(key, "~") {
		return ""
	}
	if s, err := config.Load(p.StoreFile); err == nil {
		if old, ok := s.Find(oldName); ok && oldName != "" && old.Key == key {
			return ""
		}
	}
	if _, err := os.Stat(filepath.Join(p.KeysDir, filepath.Base(key))); err == nil {
		return ""
	}
	hint := "Pick it from the Key list, or adopt it first in Tools → Keys."
	if _, err := os.Stat(filepath.Join(filepath.Dir(p.SSHConfig), filepath.Base(key))); err == nil {
		hint = fmt.Sprintf("It is in ~/.ssh: choose it from the Key list, enter ~/.ssh/%s, or adopt it in Tools → Keys.", filepath.Base(key))
	}
	return fmt.Sprintf("%q is not an adopted key. %s", key, hint)
}

// PickKeyFile opens the native file dialog on ~/.ssh and returns the chosen
// path, shortened to ~/ when it is under the home directory. Empty if cancelled.
func (ProfilesService) PickKeyFile() (string, error) {
	home, _ := os.UserHomeDir()
	path, err := application.Get().Dialog.OpenFile().
		SetTitle("Choose a private key").
		SetDirectory(filepath.Join(home, ".ssh")).
		CanChooseFiles(true).ShowHiddenFiles(true).
		PromptForSingleSelection()
	if err != nil || path == "" {
		return "", err
	}
	if home != "" && strings.HasPrefix(path, home+string(filepath.Separator)) {
		path = "~" + path[len(home):]
	}
	return path, nil
}
