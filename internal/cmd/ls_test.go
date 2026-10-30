package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/paths"
)

func TestLsEmptyStore(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	out := capture(func() error {
		return Ls(p, nil)
	})
	if !strings.Contains(out, "(no profiles)") {
		t.Errorf("empty store output = %q, want '(no profiles)'", out)
	}
}

func TestLsFlatListsAllProfiles(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "dev", Host: "dev.local", User: "u", Port: 22, Auth: config.AuthAgent},
		{Name: "prod", Host: "prod.example.com", User: "u", Port: 22, Auth: config.AuthAgent, Folder: "work"},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}
	out := capture(func() error {
		return Ls(p, nil)
	})
	if !strings.Contains(out, "dev") || !strings.Contains(out, "work/prod") {
		t.Errorf("flat list output = %q, want to contain 'dev' and 'work/prod'", out)
	}
}

func TestLsTreeGroupsByFolder(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "local", Host: "localhost", User: "u", Port: 22, Auth: config.AuthAgent},
		{Name: "web1", Host: "web1.example.com", User: "u", Port: 22, Auth: config.AuthAgent, Folder: "prod"},
		{Name: "web2", Host: "web2.example.com", User: "u", Port: 22, Auth: config.AuthAgent, Folder: "prod"},
		{Name: "db", Host: "db.internal", User: "u", Port: 22, Auth: config.AuthAgent, Folder: "staging"},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}
	out := capture(func() error {
		return Ls(p, []string{"--tree"})
	})
	lines := strings.Split(out, "\n")
	prodIdx, stagingIdx := -1, -1
	for i, line := range lines {
		if strings.TrimSpace(line) == "prod" {
			prodIdx = i
		}
		if strings.TrimSpace(line) == "staging" {
			stagingIdx = i
		}
	}
	if prodIdx < 0 || stagingIdx < 0 {
		t.Errorf("tree output missing folder headers: %q", out)
	}
	if !strings.Contains(out, "web1") || !strings.Contains(out, "web2") {
		t.Errorf("tree output missing prod profiles: %q", out)
	}
}

func TestLsFlatIsSorted(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "zebra", Host: "z.local", User: "u", Port: 22, Auth: config.AuthAgent},
		{Name: "alpha", Host: "a.local", User: "u", Port: 22, Auth: config.AuthAgent},
		{Name: "bravo", Host: "b.local", User: "u", Port: 22, Auth: config.AuthAgent},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}
	out := capture(func() error {
		return Ls(p, nil)
	})
	lines := strings.Fields(strings.TrimSpace(out))
	if len(lines) != 3 || lines[0] != "alpha" || lines[1] != "bravo" || lines[2] != "zebra" {
		t.Errorf("flat list not sorted: %v", lines)
	}
}

func TestLsTreeFoldersAreSortedDeterministically(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	// Folder names chosen so insertion order (via Profiles slice, itself
	// sorted by profile name) differs from alphabetical folder order.
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "a-profile", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent, Folder: "zulu"},
		{Name: "b-profile", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent, Folder: "alpha"},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}
	// Run several times: map iteration order is randomized per-process-run
	// in Go, so a single pass wouldn't catch a regression back to map order.
	for i := 0; i < 5; i++ {
		out := capture(func() error {
			return Ls(p, []string{"--tree"})
		})
		alphaIdx := strings.Index(out, "alpha")
		zuluIdx := strings.Index(out, "zulu")
		if alphaIdx < 0 || zuluIdx < 0 || alphaIdx > zuluIdx {
			t.Fatalf("iteration %d: folders not in sorted order: %q", i, out)
		}
	}
}

func TestLsFilterByTagFlat(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "web-prod", Host: "10.0.0.1", User: "u", Port: 22, Auth: config.AuthAgent, Tags: []string{"web", "prod"}},
		{Name: "web-dev", Host: "10.0.0.2", User: "u", Port: 22, Auth: config.AuthAgent, Tags: []string{"web", "dev"}},
		{Name: "db-prod", Host: "10.0.0.3", User: "u", Port: 22, Auth: config.AuthAgent, Tags: []string{"db", "prod"}},
		{Name: "untagged", Host: "10.0.0.4", User: "u", Port: 22, Auth: config.AuthAgent},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}

	out := capture(func() error {
		return Ls(p, []string{"--tag", "prod"})
	})
	if !strings.Contains(out, "web-prod") || !strings.Contains(out, "db-prod") {
		t.Errorf("expected web-prod and db-prod in output, got: %q", out)
	}
	if strings.Contains(out, "web-dev") || strings.Contains(out, "untagged") {
		t.Errorf("web-dev or untagged should not be in filtered output, got: %q", out)
	}
}

func TestLsFilterByTagTree(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "api", Host: "10.0.0.1", User: "u", Port: 22, Auth: config.AuthAgent, Folder: "backend", Tags: []string{"prod"}},
		{Name: "worker", Host: "10.0.0.2", User: "u", Port: 22, Auth: config.AuthAgent, Folder: "backend", Tags: []string{"staging"}},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}

	out := capture(func() error {
		return Ls(p, []string{"--tree", "--tag", "prod"})
	})
	if !strings.Contains(out, "backend") || !strings.Contains(out, "api") {
		t.Errorf("expected backend tree with api, got: %q", out)
	}
	if strings.Contains(out, "worker") {
		t.Errorf("worker should not be in output, got: %q", out)
	}
}

func TestLsFilterByTagEmptyResults(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "web", Host: "10.0.0.1", User: "u", Port: 22, Auth: config.AuthAgent, Tags: []string{"prod"}},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}

	out := capture(func() error {
		return Ls(p, []string{"--tag", "nonexistent"})
	})
	if !strings.Contains(out, "(no profiles)") {
		t.Errorf("expected '(no profiles)', got: %q", out)
	}
}

func TestLsSkipsRdpProfiles(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "ssh-box", Host: "ssh.local", User: "u", Port: 22, Auth: config.AuthAgent},
		{Protocol: config.ProtocolRDP, Name: "win-rdp", Host: "win.local", User: "admin", Port: 3389},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}

	// Flat view
	flat := capture(func() error {
		return Ls(p, nil)
	})
	if strings.Contains(flat, "win-rdp") {
		t.Errorf("flat Ls should not show RDP profile, got:\n%s", flat)
	}
	if !strings.Contains(flat, "ssh-box") {
		t.Errorf("flat Ls should show SSH profile, got:\n%s", flat)
	}

	// Tree view
	tree := capture(func() error {
		return Ls(p, []string{"--tree"})
	})
	if strings.Contains(tree, "win-rdp") {
		t.Errorf("tree Ls should not show RDP profile, got:\n%s", tree)
	}
	if !strings.Contains(tree, "ssh-box") {
		t.Errorf("tree Ls should show SSH profile, got:\n%s", tree)
	}

	// Store with only RDP profiles should report "(no profiles)"
	pOnlyRDP := paths.New(t.TempDir())
	if err := paths.EnsureDirs(pOnlyRDP); err != nil {
		t.Fatal(err)
	}
	sOnlyRDP := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Protocol: config.ProtocolRDP, Name: "win-only", Host: "win.local", User: "admin", Port: 3389},
	}}
	if err := config.Save(pOnlyRDP.StoreFile, sOnlyRDP); err != nil {
		t.Fatal(err)
	}
	onlyRdpOut := capture(func() error {
		return Ls(pOnlyRDP, nil)
	})
	if !strings.Contains(onlyRdpOut, "(no profiles)") {
		t.Errorf("expected '(no profiles)' when only RDP profiles exist, got: %q", onlyRdpOut)
	}
}

// capture redirects stdout and runs fn, returning the captured output.
func capture(fn func() error) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	err := fn()
	w.Close()
	os.Stdout = old
	var b bytes.Buffer
	io.Copy(&b, r)
	r.Close()
	if err != nil {
		b.WriteString("ERROR: " + err.Error())
	}
	return b.String()
}
