package cmd

import (
	"os"
	"testing"

	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/core"
	"github.com/poppyszn/relay-ssh/internal/paths"
)

func TestResolveProfileByBareName(t *testing.T) {
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "dev-app-server", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent, Folder: "Vascon"},
	}}
	prof, ok := core.ResolveProfile(s, "dev-app-server")
	if !ok || prof.Name != "dev-app-server" {
		t.Errorf("core.ResolveProfile(bare name) = %+v, %v", prof, ok)
	}
}

func TestResolveProfileByFolderQualifiedName(t *testing.T) {
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "dev-app-server", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent, Folder: "Vascon"},
	}}
	prof, ok := core.ResolveProfile(s, "Vascon/dev-app-server")
	if !ok || prof.Name != "dev-app-server" {
		t.Errorf("core.ResolveProfile(folder/name) = %+v, %v, want dev-app-server found", prof, ok)
	}
}

func TestResolveProfileRejectsWrongFolder(t *testing.T) {
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "dev-app-server", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent, Folder: "Vascon"},
	}}
	if _, ok := core.ResolveProfile(s, "WrongFolder/dev-app-server"); ok {
		t.Error("resolveProfile should reject a name with the wrong folder prefix")
	}
}

func TestResolveProfileRejectsUnknownName(t *testing.T) {
	s := config.Store{Version: config.Version}
	if _, ok := core.ResolveProfile(s, "nonexistent"); ok {
		t.Error("resolveProfile should reject a name that doesn't exist")
	}
	if _, ok := core.ResolveProfile(s, "Vascon/nonexistent"); ok {
		t.Error("resolveProfile should reject a folder/name pair that doesn't exist")
	}
}

func TestConnectRejectsUnknownProfile(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(p.StoreFile, config.Store{Version: config.Version}); err != nil {
		t.Fatal(err)
	}
	err := Connect(p, "unknown")
	if err == nil || err.Error() != `profile "unknown" not found` {
		t.Errorf("Connect: got %v, want not found", err)
	}
}

func TestRmRemovesProfile(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "to-remove", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent},
		{Name: "to-keep", Host: "h2", User: "u", Port: 22, Auth: config.AuthAgent},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}
	if err := Rm(p, []string{"to-remove"}); err != nil {
		t.Fatalf("Rm: %v", err)
	}
	s, _ = config.Load(p.StoreFile)
	if len(s.Profiles) != 1 || s.Profiles[0].Name != "to-keep" {
		t.Errorf("Rm didn't remove correctly: %v", s.Names())
	}
}

func TestRmAcceptsFolderQualifiedName(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "dev-app-server", Host: "h", User: "u", Port: 22, Auth: config.AuthAgent, Folder: "Vascon"},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}
	if err := Rm(p, []string{"Vascon/dev-app-server"}); err != nil {
		t.Fatalf("Rm with folder-qualified name: %v", err)
	}
	got, _ := config.Load(p.StoreFile)
	if len(got.Profiles) != 0 {
		t.Errorf("Rm with folder-qualified name did not remove the profile: %v", got.Names())
	}
}

func TestRmRejectsUnknown(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(p.StoreFile, config.Store{Version: config.Version}); err != nil {
		t.Fatal(err)
	}
	err := Rm(p, []string{"unknown"})
	if err == nil {
		t.Fatal("Rm should reject unknown profile")
	}
}

func TestSyncRegeneratesConfig(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	s := config.Store{Version: config.Version, Profiles: []config.Profile{
		{Name: "test", Host: "example.com", User: "u", Port: 22, Auth: config.AuthAgent},
	}}
	if err := config.Save(p.StoreFile, s); err != nil {
		t.Fatal(err)
	}
	if err := Sync(p); err != nil {
		t.Fatalf("Sync: %v", err)
	}
	if _, err := os.Stat(p.GenConfig); err != nil {
		t.Errorf("Sync did not create gen config: %v", err)
	}
}

func TestEditRejectsUnknownProfile(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	if err := config.Save(p.StoreFile, config.Store{Version: config.Version}); err != nil {
		t.Fatal(err)
	}
	err := EditCmd(p, []string{"unknown"})
	if err == nil {
		t.Fatal("EditCmd should reject unknown profile")
	}
}

func TestCpRejectsInvalidFormat(t *testing.T) {
	p := paths.New(t.TempDir())
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	err := CpCmd(p, []string{"invalid", "dst"})
	if err == nil || err.Error() != "invalid format: use profile:remote-path" {
		t.Errorf("CpCmd should reject invalid format, got: %v", err)
	}
}
