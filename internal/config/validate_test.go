package config

import "testing"

// valid returns a profile that passes validation, so each test can mutate one
// field and assert on that field alone.
func valid() Profile {
	return Profile{
		Name: "prod-web", Host: "10.0.0.5", User: "deploy",
		Port: 22, Auth: AuthAgent,
	}
}

func TestProfileValidateAcceptsAGoodProfile(t *testing.T) {
	if err := valid().Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
}

func TestProfileValidateRejectsBadFields(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Profile)
	}{
		{"empty name", func(p *Profile) { p.Name = "" }},
		{"name with a space", func(p *Profile) { p.Name = "prod web" }},
		{"name with a star", func(p *Profile) { p.Name = "web*" }},
		{"name with a question mark", func(p *Profile) { p.Name = "web?" }},
		{"name with a bang", func(p *Profile) { p.Name = "!web" }},
		{"name with a hash", func(p *Profile) { p.Name = "web#1" }},
		{"reserved name ls", func(p *Profile) { p.Name = "ls" }},
		{"reserved name backup", func(p *Profile) { p.Name = "backup" }},
		{"empty host", func(p *Profile) { p.Host = "" }},
		{"empty user", func(p *Profile) { p.User = "" }},
		{"port zero", func(p *Profile) { p.Port = 0 }},
		{"port negative", func(p *Profile) { p.Port = -1 }},
		{"port too large", func(p *Profile) { p.Port = 65536 }},
		{"unknown auth", func(p *Profile) { p.Auth = "magic" }},
		{"empty auth", func(p *Profile) { p.Auth = "" }},
		{"key auth without a key", func(p *Profile) { p.Auth = AuthKey; p.Key = "" }},
		{"agent auth with a key", func(p *Profile) { p.Auth = AuthAgent; p.Key = "id_ed25519" }},
		{"folder with a leading slash", func(p *Profile) { p.Folder = "/work" }},
		{"folder with a trailing slash", func(p *Profile) { p.Folder = "work/" }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := valid()
			tt.mutate(&p)
			if err := p.Validate(); err == nil {
				t.Fatalf("Validate() = nil, want an error for %s", tt.name)
			}
		})
	}
}

func TestProfileValidateAcceptsGoodNames(t *testing.T) {
	for _, name := range []string{"web", "prod-web", "web.example.com", "db_1", "A1"} {
		p := valid()
		p.Name = name
		if err := p.Validate(); err != nil {
			t.Errorf("name %q rejected: %v", name, err)
		}
	}
}

func TestValidateNameStandalone(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty", "", true},
		{"reserved", "ls", true},
		{"space", "prod web", true},
		{"star", "web*", true},
		{"good simple", "web", false},
		{"good with dots and dashes", "prod-web.1", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName(%q) = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestValidatePortStandalone(t *testing.T) {
	tests := []struct {
		port    int
		wantErr bool
	}{
		{0, true}, {-1, true}, {65536, true},
		{1, false}, {22, false}, {65535, false},
	}
	for _, tt := range tests {
		if err := ValidatePort(tt.port); (err != nil) != tt.wantErr {
			t.Errorf("ValidatePort(%d) = %v, wantErr %v", tt.port, err, tt.wantErr)
		}
	}
}

func TestValidateFolderStandalone(t *testing.T) {
	tests := []struct {
		folder  string
		wantErr bool
	}{
		{"", false}, {"work", false}, {"work/prod", false},
		{"/work", true}, {"work/", true},
	}
	for _, tt := range tests {
		if err := ValidateFolder(tt.folder); (err != nil) != tt.wantErr {
			t.Errorf("ValidateFolder(%q) = %v, wantErr %v", tt.folder, err, tt.wantErr)
		}
	}
}

func TestTunnelValidate(t *testing.T) {
	tests := []struct {
		name    string
		tunnel  Tunnel
		wantErr bool
	}{
		{"good local", Tunnel{TunnelLocal, "5432", "db:5432"}, false},
		{"good remote", Tunnel{TunnelRemote, "8080", "localhost:80"}, false},
		{"good dynamic", Tunnel{TunnelDynamic, "1080", ""}, false},
		{"local without target", Tunnel{TunnelLocal, "5432", ""}, true},
		{"remote without target", Tunnel{TunnelRemote, "8080", ""}, true},
		{"dynamic with target", Tunnel{TunnelDynamic, "1080", "db:5432"}, true},
		{"no listen", Tunnel{TunnelLocal, "", "db:5432"}, true},
		{"unknown type", Tunnel{"sideways", "1", "a:1"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.tunnel.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStoreValidateRejectsDuplicateNames(t *testing.T) {
	a, b := valid(), valid()
	s := Store{Version: Version, Profiles: []Profile{a, b}}
	if err := s.Validate(); err == nil {
		t.Fatal("expected an error for duplicate names, got nil")
	}
}

func TestStoreValidateRejectsUnknownJumpTarget(t *testing.T) {
	p := valid()
	p.Jump = "ghost"
	s := Store{Version: Version, Profiles: []Profile{p}}
	if err := s.Validate(); err == nil {
		t.Fatal("expected an error for an unknown jump target, got nil")
	}
}

func TestStoreValidateAcceptsAJumpChain(t *testing.T) {
	a, b, c := valid(), valid(), valid()
	a.Name, b.Name, c.Name = "a", "b", "c"
	a.Jump, b.Jump = "b", "c"
	s := Store{Version: Version, Profiles: []Profile{a, b, c}}
	if err := s.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil", err)
	}
}

func TestStoreValidateRejectsJumpCycles(t *testing.T) {
	t.Run("self", func(t *testing.T) {
		a := valid()
		a.Name, a.Jump = "a", "a"
		s := Store{Version: Version, Profiles: []Profile{a}}
		if err := s.Validate(); err == nil {
			t.Fatal("expected an error for a self-referencing jump, got nil")
		}
	})
	t.Run("two hop", func(t *testing.T) {
		a, b := valid(), valid()
		a.Name, b.Name = "a", "b"
		a.Jump, b.Jump = "b", "a"
		s := Store{Version: Version, Profiles: []Profile{a, b}}
		if err := s.Validate(); err == nil {
			t.Fatal("expected an error for a two-hop jump cycle, got nil")
		}
	})
	t.Run("three hop", func(t *testing.T) {
		a, b, c := valid(), valid(), valid()
		a.Name, b.Name, c.Name = "a", "b", "c"
		a.Jump, b.Jump, c.Jump = "b", "c", "a"
		s := Store{Version: Version, Profiles: []Profile{a, b, c}}
		if err := s.Validate(); err == nil {
			t.Fatal("expected an error for a three-hop jump cycle, got nil")
		}
	})
}

func TestStoreValidateRejectsWrongVersion(t *testing.T) {
	s := Store{Version: 99, Profiles: []Profile{valid()}}
	if err := s.Validate(); err == nil {
		t.Fatal("expected an error for an unsupported version, got nil")
	}
}

func TestStoreFindAndNames(t *testing.T) {
	a, b := valid(), valid()
	a.Name, b.Name = "alpha", "beta"
	s := Store{Version: Version, Profiles: []Profile{a, b}}

	got, ok := s.Find("beta")
	if !ok {
		t.Fatal("Find(beta) not found")
	}
	if got.Name != "beta" {
		t.Errorf("Find(beta).Name = %q", got.Name)
	}
	if _, ok := s.Find("ghost"); ok {
		t.Error("Find(ghost) = found, want not found")
	}

	names := s.Names()
	if len(names) != 2 || names[0] != "alpha" || names[1] != "beta" {
		t.Errorf("Names() = %v, want [alpha beta]", names)
	}
}
