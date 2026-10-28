package main

import (
	"os"
	"strings"
	"testing"
)

func TestProfilesServiceRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv("SSSH_HOME", home)
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
	cfg, err := os.ReadFile(home + "/.ssh/sssh.d/config")
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
