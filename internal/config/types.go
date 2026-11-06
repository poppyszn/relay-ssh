// Package config defines the connection store: the TOML file that is the single
// source of truth for every profile relay-ssh knows about.
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

const (
	ProtocolSSH = "ssh"
	ProtocolRDP = "rdp"
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
// Name doubles as the ssh Host alias and as the argument to `relay-ssh <name>`,
// which is why Validate constrains it so tightly.
//
// Key is either a bare filename, meaning a key adopted into
// ~/.ssh/relay-ssh.d/keys/, or an absolute or ~-prefixed path, meaning a key linked
// in place and therefore not covered by backups.
//
// Jump names another profile, never a free-text host, so a jump host gets the
// same key and port handling as any other connection.
type Profile struct {
	// ID is permanent and unique across the store ("@" plus 12 hex digits; '@' cannot
	// appear in a name). Names only need to be unique within a folder; the id is what
	// jumps, the vault and the desktop app use to refer to a profile.
	ID           string   `toml:"id,omitempty"`
	Protocol     string   `toml:"protocol,omitempty"`
	Name         string   `toml:"name"`
	Host         string   `toml:"host"`
	User         string   `toml:"user"`
	Domain       string   `toml:"domain,omitempty"`
	Port         int      `toml:"port"`
	Auth         AuthType `toml:"auth,omitempty"`
	Key          string   `toml:"key,omitempty"`
	Folder       string   `toml:"folder,omitempty"`
	Tags         []string `toml:"tags,omitempty"`
	Jump         string   `toml:"jump,omitempty"`
	ForwardAgent bool     `toml:"forward_agent,omitempty"`
	// Blocks is the Relay desktop app's command-block mode for this host: "on",
	// "off", or empty to follow the app setting. The CLI ignores it.
	Blocks  string   `toml:"blocks,omitempty"`
	Tunnels []Tunnel `toml:"tunnel,omitempty"`
}

// IsRDP reports whether the profile is an RDP connection.
func (p Profile) IsRDP() bool {
	return p.Protocol == ProtocolRDP
}

// Store is the whole of connections.toml.
//
// Folders lists folders that exist even when no connection is in them, so an
// empty folder or subfolder can be created ahead of time. A folder that holds
// connections exists without being listed; Folders only needs to name the empty
// ones. Files written before this field existed simply omit it.
type Store struct {
	Version  int       `toml:"version"`
	Folders  []string  `toml:"folders,omitempty"`
	Profiles []Profile `toml:"profile"`
}

// Find returns the profile a key names: an id, "Folder/name", or a bare name that
// only one profile has. A bare name shared by several profiles finds none; use
// Resolve to learn why.
func (s Store) Find(key string) (Profile, bool) {
	if i := s.index(key); i >= 0 {
		return s.Profiles[i], true
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
