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
	ID           string      `json:"id"`
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
	Protocol     string      `json:"protocol,omitempty"`
	Domain       string      `json:"domain,omitempty"`
}

// ProfilesService exposes the connection store to the frontend. Every change
// goes through internal/core, so the CLI sees it immediately.
type ProfilesService struct{}

func toProfile(c config.Profile) Profile {
	p := Profile{
		ID: c.ID, Protocol: c.Protocol, Domain: c.Domain,
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
		ID: p.ID, Protocol: p.Protocol, Domain: p.Domain,
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

// Save adds a profile (id == "") or updates the one with that id, possibly renaming or moving it.
func (ps ProfilesService) Save(id string, prof Profile) error {
	return ps.save(id, prof, "")
}

// SaveWithPassword saves the profile and keeps its password in the vault, bound to the
// connection's host, user and port. It checks the password and the vault first, so a refusal
// leaves nothing half-saved.
func (ps ProfilesService) SaveWithPassword(id string, prof Profile, password string) error {
	if prof.Protocol != config.ProtocolRDP && prof.Auth != "password" {
		return errors.New("passwords are only kept for connections that use password auth")
	}
	if !vault.ValidSecret(password) {
		return vault.ErrBadSecret
	}
	if !getVault().Status().Unlocked {
		return vault.ErrVaultLocked
	}
	return ps.save(id, prof, password)
}

func (ProfilesService) save(id string, prof Profile, password string) error {
	p, err := paths.Default()
	if err != nil {
		return err
	}
	if err := paths.EnsureDirs(p); err != nil {
		return err
	}
	if msg := keyProblem(p, id, prof); msg != "" {
		return errors.New(msg)
	}
	oldBinding := ""
	if id != "" {
		if store, err := config.Load(p.StoreFile); err == nil {
			if old, ok := core.ResolveProfile(store, id); ok {
				id, oldBinding = old.ID, bindingOf(old)
			}
		}
	}
	if err := core.SaveProfile(p, id, fromProfile(prof)); err != nil {
		return err
	}
	if id == "" { // a new connection: its id was made on save
		if store, err := config.Load(p.StoreFile); err == nil {
			if saved, ok := store.Find(fromProfile(prof).Address()); ok {
				id = saved.ID
			}
		}
	}

	// Keep the vault in step. A locked vault cannot be touched, so these are best effort
	// (the form asks to unlock before a change); an entry left behind is inert, because it is
	// only released for the connection details it was saved with.
	v := getVault()
	newBinding := bindingOf(fromProfile(prof))
	if prof.Protocol != config.ProtocolRDP && prof.Auth != "password" {
		_ = v.Forget(id)
	} else {
		// An edit made here moves the binding along only if it was intact; see Vault.Rename.
		_ = v.Rename(id, id, oldBinding, newBinding)
	}
	if password != "" {
		if err := v.Set(id, password, newBinding); err != nil {
			return fmt.Errorf("the connection was saved but its password was not stored: %w", err)
		}
	}
	return nil
}

// Delete removes a profile by id.
func (ProfilesService) Delete(id string) error {
	p, err := paths.Default()
	if err != nil {
		return err
	}
	removed, err := core.RemoveProfile(p, id)
	if err != nil {
		return err
	}
	_ = getVault().Forget(removed.ID)
	return nil
}

// PreviewResult is the form's live preview: the exact Host block the profile
// would generate, and the reason it cannot be saved yet, if any.
type PreviewResult struct {
	Config  string `json:"config"`
	Problem string `json:"problem"`
}

// Preview renders the generated SSH config for prof (replacing oldName, or
// added when id is empty) without writing anything.
func (ProfilesService) Preview(id string, prof Profile) PreviewResult {
	cand := fromProfile(prof)
	// Generate has no validation of its own, so the preview renders even
	// while the form is half filled in; Problem says what blocks saving.
	var res PreviewResult
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
	old, _ := s.Find(id)
	for _, cur := range s.Profiles {
		if id == "" || cur.ID != old.ID {
			profiles = append(profiles, cur)
		}
	}
	cand.ID = old.ID
	if cand.ID == "" {
		cand.ID = "@0000000000ff" // only so the preview can tell the candidate from its neighbours
	}
	s.Profiles = append(profiles, cand)
	if !cand.IsRDP() {
		res.Config = hostBlock(sshconfig.Generate(s), sshconfig.Alias(s, cand))
	}
	if err := s.Validate(); err != nil {
		res.Problem = err.Error()
	} else if msg := keyProblem(p, id, prof); msg != "" {
		res.Problem = msg
	}
	return res
}

// keyProblem explains a key setting that ssh cannot use. A bare key name means a key adopted into
// Relay's vault; if that file is not there ssh fails with "no such identity". Only a new profile or a
// changed key is checked, so a connection already saved that way can still be edited.
func keyProblem(p paths.Paths, id string, prof Profile) string {
	key := prof.Key
	if prof.Auth != "key" || key == "" || filepath.IsAbs(key) || strings.HasPrefix(key, "~") {
		return ""
	}
	if s, err := config.Load(p.StoreFile); err == nil {
		if old, ok := s.Find(id); ok && id != "" && old.Key == key {
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

// hostBlock cuts one Host block out of generated ssh config text.
func hostBlock(all, alias string) string {
	start := strings.Index(all, "Host "+alias+"\n")
	if start < 0 {
		return ""
	}
	block := all[start:]
	if end := strings.Index(block[1:], "\nHost "); end >= 0 {
		block = block[:end+1]
	}
	return strings.TrimRight(block, "\n")
}
