package main

import (
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
func (ProfilesService) Save(oldName string, prof Profile) error {
	p, err := paths.Default()
	if err != nil {
		return err
	}
	if err := paths.EnsureDirs(p); err != nil {
		return err
	}
	if err := core.SaveProfile(p, oldName, fromProfile(prof)); err != nil {
		return err
	}

	// Keep vault in step with connection changes.
	// These calls must not fail the save if the vault is locked.
	v := getVault()
	if oldName != "" && oldName != prof.Name {
		_ = v.Rename(oldName, prof.Name)
	}
	if prof.Auth != "password" {
		_ = v.Forget(prof.Name)
		if oldName != "" {
			_ = v.Forget(oldName)
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
	}
	return res
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
