package main

import (
	"github.com/poppyszn/super-ssh-v2/internal/config"
	"github.com/poppyszn/super-ssh-v2/internal/core"
	"github.com/poppyszn/super-ssh-v2/internal/paths"
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
	Tunnels      []TunnelDTO `json:"tunnels"`
}

// ProfilesService exposes the connection store to the frontend. Every change
// goes through internal/core, so the CLI sees it immediately.
type ProfilesService struct{}

func toProfile(c config.Profile) Profile {
	p := Profile{
		Name: c.Name, Host: c.Host, User: c.User, Port: c.Port, Auth: string(c.Auth),
		Key: c.Key, Folder: c.Folder, Tags: c.Tags, Jump: c.Jump, ForwardAgent: c.ForwardAgent,
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
		Key: p.Key, Folder: p.Folder, Tags: p.Tags, Jump: p.Jump, ForwardAgent: p.ForwardAgent,
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
	return core.SaveProfile(p, oldName, fromProfile(prof))
}

// Delete removes a profile by name.
func (ProfilesService) Delete(name string) error {
	p, err := paths.Default()
	if err != nil {
		return err
	}
	_, err = core.RemoveProfile(p, name)
	return err
}
