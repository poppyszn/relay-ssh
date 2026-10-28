package cmd

import (
	"testing"

	"github.com/poppyszn/super-ssh-v2/internal/config"
	"github.com/poppyszn/super-ssh-v2/internal/paths"
)

func TestAddWithLocalTunnelRoundTrip(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(p.StoreFile, config.Store{Version: config.Version}); err != nil {
		t.Fatal(err)
	}
	// Name: web, Host: 10.0.0.1, User: u, Port: 22, Auth: agent, Jump: none, ForwardAgent: n,
	// Folder: 1, Tags: "",
	// Add port forward? y, Type: 1 (local), Port: 8080, Target: localhost:80, Add another? n
	input := "web\n10.0.0.1\nu\n\n\n\n1\n\ny\n1\n8080\nlocalhost:80\nn\n"
	feedStdin(t, input)

	if err := Add(p, nil); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, err := config.Load(p.StoreFile)
	if err != nil {
		t.Fatal(err)
	}
	prof, ok := got.Find("web")
	if !ok {
		t.Fatal("web profile not found")
	}
	if len(prof.Tunnels) != 1 {
		t.Fatalf("expected 1 tunnel, got %d", len(prof.Tunnels))
	}
	tun := prof.Tunnels[0]
	if tun.Type != config.TunnelLocal || tun.Listen != "8080" || tun.Target != "localhost:80" {
		t.Errorf("tunnel = %+v, want local 8080 -> localhost:80", tun)
	}
}

func TestAddMultipleTunnelsPreservesOrder(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(p.StoreFile, config.Store{Version: config.Version}); err != nil {
		t.Fatal(err)
	}
	// Tunnel 1: local 8080 -> localhost:80
	// Add another? y
	// Tunnel 2: dynamic 1080 (no target prompted)
	// Add another? n
	input := "multi\n10.0.0.1\nu\n\n\n\n1\n\ny\n1\n8080\nlocalhost:80\ny\n3\n1080\nn\n"
	feedStdin(t, input)

	if err := Add(p, nil); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, _ := config.Load(p.StoreFile)
	prof, _ := got.Find("multi")
	if len(prof.Tunnels) != 2 {
		t.Fatalf("expected 2 tunnels, got %d", len(prof.Tunnels))
	}
	if prof.Tunnels[0].Type != config.TunnelLocal || prof.Tunnels[0].Listen != "8080" {
		t.Errorf("tunnel 0 = %+v, want local 8080", prof.Tunnels[0])
	}
	if prof.Tunnels[1].Type != config.TunnelDynamic || prof.Tunnels[1].Listen != "1080" || prof.Tunnels[1].Target != "" {
		t.Errorf("tunnel 1 = %+v, want dynamic 1080 with empty target", prof.Tunnels[1])
	}
}

func TestAddDynamicTunnelNeverPromptsForTarget(t *testing.T) {
	disableFzfForTest = true
	defer func() { disableFzfForTest = false }()

	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(p.StoreFile, config.Store{Version: config.Version}); err != nil {
		t.Fatal(err)
	}
	// Tunnel: dynamic 1080, next answer is "n" for "Add another?". If target was prompted, "n" would be consumed as target.
	input := "socks\n10.0.0.1\nu\n\n\n\n1\n\ny\n3\n1080\nn\n"
	feedStdin(t, input)

	if err := Add(p, nil); err != nil {
		t.Fatalf("Add: %v", err)
	}
	got, _ := config.Load(p.StoreFile)
	prof, _ := got.Find("socks")
	if len(prof.Tunnels) != 1 {
		t.Fatalf("expected 1 tunnel, got %d", len(prof.Tunnels))
	}
	if prof.Tunnels[0].Type != config.TunnelDynamic || prof.Tunnels[0].Target != "" {
		t.Errorf("dynamic tunnel has target: %q", prof.Tunnels[0].Target)
	}
}
