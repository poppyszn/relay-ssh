package config

import "testing"

func TestRdpProfileValidate(t *testing.T) {
	validRDP := func() Profile {
		return Profile{
			Protocol: ProtocolRDP,
			Name:     "win-box",
			Host:     "192.168.1.100",
			User:     "Administrator",
			Domain:   "CORP",
			Port:     3389,
		}
	}

	t.Run("accepts good RDP profile", func(t *testing.T) {
		p := validRDP()
		if err := p.Validate(); err != nil {
			t.Fatalf("Validate() = %v, want nil", err)
		}
	})

	t.Run("accepts RDP profile without domain", func(t *testing.T) {
		p := validRDP()
		p.Domain = ""
		if err := p.Validate(); err != nil {
			t.Fatalf("Validate() = %v, want nil", err)
		}
	})

	t.Run("accepts RDP profile with port 0 (defaults to 3389)", func(t *testing.T) {
		p := validRDP()
		p.Port = 0
		if err := p.Validate(); err != nil {
			t.Fatalf("Validate() = %v, want nil", err)
		}
	})

	t.Run("ignores auth for RDP", func(t *testing.T) {
		p := validRDP()
		p.Auth = "any-string"
		if err := p.Validate(); err != nil {
			t.Fatalf("Validate() = %v, want nil", err)
		}
	})

	t.Run("rejects bad fields for RDP", func(t *testing.T) {
		cases := []struct {
			name   string
			mutate func(*Profile)
		}{
			{"empty host", func(p *Profile) { p.Host = "" }},
			{"empty user", func(p *Profile) { p.User = "" }},
			{"host with leading dash", func(p *Profile) { p.Host = "-bad" }},
			{"host with newline", func(p *Profile) { p.Host = "h\nProxy" }},
			{"user with space", func(p *Profile) { p.User = "u x" }},
			{"port negative", func(p *Profile) { p.Port = -1 }},
			{"port too large", func(p *Profile) { p.Port = 65536 }},
			{"domain with leading dash", func(p *Profile) { p.Domain = "-CORP" }},
			{"domain with control char", func(p *Profile) { p.Domain = "CORP\x1b" }},
			{"domain with space", func(p *Profile) { p.Domain = "CORP DOMAIN" }},
			{"key rejected", func(p *Profile) { p.Key = "id_rsa" }},
			{"tunnel rejected", func(p *Profile) {
				p.Tunnels = []Tunnel{{Type: TunnelLocal, Listen: "8080", Target: "80"}}
			}},
			{"forward_agent rejected", func(p *Profile) { p.ForwardAgent = true }},
			{"blocks rejected", func(p *Profile) { p.Blocks = "on" }},
			{"unknown protocol", func(p *Profile) { p.Protocol = "vnc" }},
		}
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				p := validRDP()
				tc.mutate(&p)
				if err := p.Validate(); err == nil {
					t.Fatalf("%s: expected validation error, got nil", tc.name)
				}
			})
		}
	})
}

func TestRdpJumpRules(t *testing.T) {
	sshJump := Profile{
		Name: "bastion", Host: "bastion.example.com", User: "admin",
		Port: 22, Auth: AuthAgent,
	}
	rdpProfile := Profile{
		Protocol: ProtocolRDP,
		Name:     "win-target",
		Host:     "10.0.0.5",
		User:     "admin",
		Port:     3389,
		Jump:     "bastion",
	}

	t.Run("RDP jumping through SSH profile is valid", func(t *testing.T) {
		s := Store{Version: Version, Profiles: []Profile{sshJump, rdpProfile}}
		if err := s.Validate(); err != nil {
			t.Fatalf("Validate() = %v, want nil", err)
		}
	})

	t.Run("RDP jumping through another RDP profile is rejected", func(t *testing.T) {
		rdpJump := Profile{
			Protocol: ProtocolRDP,
			Name:     "win-jump",
			Host:     "10.0.0.4",
			User:     "admin",
			Port:     3389,
		}
		target := rdpProfile
		target.Jump = "win-jump"
		s := Store{Version: Version, Profiles: []Profile{rdpJump, target}}
		if err := s.Validate(); err == nil {
			t.Fatal("expected error when RDP jumps through RDP, got nil")
		}
	})

	t.Run("SSH jumping through an RDP profile is rejected", func(t *testing.T) {
		rdpJump := Profile{
			Protocol: ProtocolRDP,
			Name:     "win-jump",
			Host:     "10.0.0.4",
			User:     "admin",
			Port:     3389,
		}
		sshTarget := Profile{
			Name: "linux-target", Host: "10.0.0.6", User: "ubuntu",
			Port: 22, Auth: AuthAgent, Jump: "win-jump",
		}
		s := Store{Version: Version, Profiles: []Profile{rdpJump, sshTarget}}
		if err := s.Validate(); err == nil {
			t.Fatal("expected error when SSH jumps through RDP, got nil")
		}
	})
}
