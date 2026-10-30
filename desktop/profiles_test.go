package main

import (
	"os"
	"strings"
	"testing"
)

func TestProfilesServiceRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("RELAY_SSH_HOME", home)
	svc := ProfilesService{}

	bastion := Profile{Name: "bastion", Host: "b.example.com", User: "ops", Port: 22, Auth: "agent", Folder: "prod", Tags: []string{"jump"}}
	web := Profile{Name: "web", Host: "10.0.0.5", User: "deploy", Port: 2222, Auth: "key", Key: "~/.ssh/id_ed25519",
		Jump: "bastion", ForwardAgent: true, Tunnels: []TunnelDTO{{Type: "local", Listen: "8080", Target: "localhost:80"}}}
	for _, p := range []Profile{bastion, web} {
		if err := svc.Save("", p); err != nil {
			t.Fatalf("Save %s: %v", p.Name, err)
		}
	}

	got, err := svc.List()
	if err != nil || len(got) != 2 {
		t.Fatalf("List = %v, %v", got, err)
	}
	if w := got[1]; w.Name != "web" || w.Jump != "bastion" || len(w.Tunnels) != 1 || w.Tags == nil {
		t.Errorf("web round-tripped as %+v", w)
	}

	// the generated ssh config is what the terminal's ssh actually reads
	cfg, err := os.ReadFile(home + "/.ssh/relay-ssh.d/config")
	if err != nil || !strings.Contains(string(cfg), "Host web") || !strings.Contains(string(cfg), "ProxyJump bastion") {
		t.Errorf("generated config = %q, %v", cfg, err)
	}

	if err := svc.Save("", Profile{Name: "bad name", Host: "h", User: "u", Port: 22, Auth: "agent"}); err == nil {
		t.Error("invalid name should be rejected")
	}
	// a jump host that another profile uses cannot be deleted out from under it
	if err := svc.Delete("bastion"); err == nil {
		t.Error("deleting a jump target still in use should fail")
	}
	for _, name := range []string{"web", "bastion"} {
		if err := svc.Delete(name); err != nil {
			t.Fatalf("Delete %s: %v", name, err)
		}
	}
	if got, _ := svc.List(); len(got) != 0 {
		t.Errorf("after delete List = %v", got)
	}
}

func TestPreviewUsesRealGeneratorAndReportsProblems(t *testing.T) {
	t.Setenv("RELAY_SSH_HOME", t.TempDir())
	svc := ProfilesService{}
	if err := svc.Save("", Profile{Name: "bastion", Host: "b.example.com", User: "ops", Port: 22, Auth: "agent"}); err != nil {
		t.Fatal(err)
	}

	good := Profile{Name: "web-1", Host: "10.0.0.5", User: "deploy", Port: 2222, Auth: "key", Key: "~/.ssh/id_ed25519",
		Jump: "bastion", ForwardAgent: true, Tunnels: []TunnelDTO{{Type: "local", Listen: "8080", Target: "localhost:80"}}}
	r := svc.Preview("", good)
	if r.Problem != "" {
		t.Errorf("valid profile reported problem %q", r.Problem)
	}
	for _, want := range []string{"Host web-1", "HostName 10.0.0.5", "Port 2222", "IdentityFile", "ProxyJump bastion", "ForwardAgent yes", "LocalForward 8080 localhost:80"} {
		if !strings.Contains(r.Config, want) {
			t.Errorf("preview missing %q:\n%s", want, r.Config)
		}
	}
	if strings.Contains(r.Config, "Host bastion") || strings.Contains(r.Config, "managed by") {
		t.Errorf("preview should show only this profile, got:\n%s", r.Config)
	}

	bad := good
	bad.Jump = "ghost"
	if r := svc.Preview("", bad); r.Problem == "" || r.Config == "" {
		t.Errorf("unknown jump should give a problem but still a preview, got %+v", r)
	}
	// previewing must never write anything
	if got, _ := svc.List(); len(got) != 1 {
		t.Errorf("Preview changed the store: %v", got)
	}
}

// The per-connection command-block mode is saved with the profile, validated, and kept out of
// the generated ssh config (it is an app setting, not an ssh option).
func TestProfileBlocksModeRoundTrips(t *testing.T) {
	home := t.TempDir()
	t.Setenv("RELAY_SSH_HOME", home)
	svc := ProfilesService{}
	if err := svc.Save("", Profile{Name: "box", Host: "h", User: "u", Port: 22, Auth: "agent", Blocks: "on"}); err != nil {
		t.Fatal(err)
	}
	got, err := svc.List()
	if err != nil || len(got) != 1 || got[0].Blocks != "on" {
		t.Fatalf("List = %+v, %v", got, err)
	}
	if err := svc.Save("", Profile{Name: "box2", Host: "h", User: "u", Port: 22, Auth: "agent", Blocks: "sometimes"}); err == nil {
		t.Error("an unknown blocks mode should be rejected")
	}
	cfg, _ := os.ReadFile(home + "/.ssh/relay-ssh.d/config")
	if strings.Contains(strings.ToLower(string(cfg)), "blocks") {
		t.Errorf("blocks leaked into the generated ssh config:\n%s", cfg)
	}
}

// A bare key name means "a key adopted into the vault". Saving one that is not there makes ssh fail with
// "no such identity", so it is refused with a message that says what to do; an existing profile whose stored
// key is already like that can still be edited.
func TestSavingABareKeyNameRequiresAnAdoptedKey(t *testing.T) {
	home := t.TempDir()
	t.Setenv("RELAY_SSH_HOME", home)
	svc := ProfilesService{}
	_ = os.MkdirAll(home+"/.ssh", 0o700)
	_ = os.WriteFile(home+"/.ssh/id_work", []byte("-----BEGIN OPENSSH PRIVATE KEY-----\nx\n"), 0o600)
	base := Profile{Name: "box", Host: "h", User: "u", Port: 22, Auth: "key"}

	missing := base
	missing.Key = "id_work"
	err := svc.Save("", missing)
	if err == nil || !strings.Contains(err.Error(), "Tools") || !strings.Contains(err.Error(), "~/.ssh/id_work") {
		t.Fatalf("a bare name with no adopted key: err = %v", err)
	}
	if got := svc.Preview("", missing).Problem; !strings.Contains(got, "not an adopted key") {
		t.Errorf("Preview problem = %q", got)
	}

	for _, ok := range []string{"~/.ssh/id_work", home + "/.ssh/id_work"} {
		p := base
		p.Key = ok
		if err := svc.Save("", p); err != nil {
			t.Errorf("path %q should be accepted: %v", ok, err)
		}
		_ = svc.Delete("box")
	}

	_ = os.MkdirAll(home+"/.ssh/relay-ssh.d/keys", 0o700)
	_ = os.WriteFile(home+"/.ssh/relay-ssh.d/keys/id_work", []byte("k"), 0o600)
	if err := svc.Save("", missing); err != nil {
		t.Errorf("an adopted key by name should be accepted: %v", err)
	}
	// once it is gone, unrelated edits to the saved connection still save
	_ = os.Remove(home + "/.ssh/relay-ssh.d/keys/id_work")
	edit := missing
	edit.Host = "other-host"
	if err := svc.Save("box", edit); err != nil {
		t.Errorf("editing another field of a stored profile was blocked: %v", err)
	}
}
