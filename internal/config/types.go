// Package config defines the connection store: the TOML file that is the single
// source of truth for every profile sssh knows about.
package config

// Version is the schema version written into connections.toml. Load refuses a
// file with a different version rather than guessing at its meaning.
const Version = 1

// AuthType selects which authentication directives get generated.
//
// No secret is ever stored. A profile records only which method it uses; ssh
// prompts for passwords, key passphrases, and 2FA codes itself.
type AuthType string

const (
	// AuthAgent relies on the running ssh-agent. Generates no directives.
	AuthAgent AuthType = "agent"
	// AuthKey uses a specific private key file.
	AuthKey AuthType = "key"
	// AuthPassword asks ssh to prefer interactive password authentication.
	AuthPassword AuthType = "password"
)

// TunnelType selects which port-forwarding directive gets generated.
type TunnelType string

const (
	TunnelLocal   TunnelType = "local"   // LocalForward
	TunnelRemote  TunnelType = "remote"  // RemoteForward
	TunnelDynamic TunnelType = "dynamic" // DynamicForward (SOCKS proxy)
)

// Tunnel becomes one forwarding line in the generated config. Target is unused
// for dynamic tunnels, which only need something to listen on.
type Tunnel struct {
	Type   TunnelType `toml:"type"`
	Listen string     `toml:"listen"`
	Target string     `toml:"target,omitempty"`
}

// Profile is one saved connection.
//
// Name doubles as the ssh Host alias and as the argument to `sssh <name>`,
// which is why Validate constrains it so tightly.
//
// Key is either a bare filename, meaning a key adopted into
// ~/.ssh/sssh.d/keys/, or an absolute or ~-prefixed path, meaning a key linked
// in place and therefore not covered by backups.
//
// Jump names another profile, never a free-text host, so a jump host gets the
// same key and port handling as any other connection.
type Profile struct {
	Name         string   `toml:"name"`
	Host         string   `toml:"host"`
	User         string   `toml:"user"`
	Port         int      `toml:"port"`
	Auth         AuthType `toml:"auth"`
	Key          string   `toml:"key,omitempty"`
	Folder       string   `toml:"folder,omitempty"`
	Tags         []string `toml:"tags,omitempty"`
	Jump         string   `toml:"jump,omitempty"`
	ForwardAgent bool     `toml:"forward_agent,omitempty"`
	Tunnels      []Tunnel `toml:"tunnel,omitempty"`
}

// Store is the whole of connections.toml.
type Store struct {
	Version  int       `toml:"version"`
	Profiles []Profile `toml:"profile"`
}

// Find returns the profile with the given name.
func (s Store) Find(name string) (Profile, bool) {
	for _, p := range s.Profiles {
		if p.Name == name {
			return p, true
		}
	}
	return Profile{}, false
}

// Names returns every profile name in the order the profiles are stored, which
// Save keeps sorted.
func (s Store) Names() []string {
	out := make([]string, 0, len(s.Profiles))
	for _, p := range s.Profiles {
		out = append(out, p.Name)
	}
	return out
}
