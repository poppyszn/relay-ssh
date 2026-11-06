package config

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestNewIDHasTheDocumentedShape(t *testing.T) {
	a, b := NewID(), NewID()
	if a == b {
		t.Fatal("two NewID calls returned the same id")
	}
	if !validID(a) || len(a) != 13 || a[0] != '@' {
		t.Errorf("NewID() = %q, want '@' plus 12 hex digits", a)
	}
}

func TestSameNameAllowedInDifferentFoldersOnly(t *testing.T) {
	a, b := valid(), valid()
	a.Name, b.Name = "db", "db"
	a.Folder, b.Folder = "prod", "stage"
	s := Store{Version: Version, Profiles: []Profile{a, b}}
	if err := s.Validate(); err != nil {
		t.Fatalf("same name in different folders: %v", err)
	}
	b.Folder = "prod"
	s = Store{Version: Version, Profiles: []Profile{a, b}}
	if err := s.Validate(); err == nil {
		t.Fatal("same name in the same folder must be rejected")
	}
}

func TestValidateRejectsBadAndDuplicateIDs(t *testing.T) {
	a, b := valid(), valid()
	a.Name, b.Name = "a", "b"
	a.ID, b.ID = "@aaaaaaaaaaaa", "@aaaaaaaaaaaa"
	if err := (Store{Version: Version, Profiles: []Profile{a, b}}).Validate(); err == nil {
		t.Error("duplicate ids accepted")
	}
	b.ID = "nope"
	if err := (Store{Version: Version, Profiles: []Profile{a, b}}).Validate(); err == nil {
		t.Error("malformed id accepted")
	}
}

func TestParseGivesLegacyProfilesStableIDsAndConvertsJumpNames(t *testing.T) {
	src := `version = 1
[[profile]]
name = "bastion"
host = "b"
user = "u"
port = 22
auth = "agent"
folder = "work"
[[profile]]
name = "app"
host = "a"
user = "u"
port = 22
auth = "agent"
jump = "bastion"
`
	s1, err := Parse([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	s2, _ := Parse([]byte(src))
	app1, _ := s1.Find("app")
	bast1, _ := s1.Find("bastion")
	app2, _ := s2.Find("app")
	if !validID(app1.ID) || app1.ID != app2.ID {
		t.Errorf("legacy ids not stable: %q vs %q", app1.ID, app2.ID)
	}
	if app1.Jump != bast1.ID {
		t.Errorf("jump = %q, want the bastion id %q", app1.Jump, bast1.ID)
	}
}

func TestSaveAssignsRandomIDsAndRoundTripsThem(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.toml")
	a := valid()
	a.Name = "a"
	if err := Save(path, Store{Version: Version, Profiles: []Profile{a}}); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !validID(got.Profiles[0].ID) {
		t.Fatalf("saved profile has no id: %+v", got.Profiles[0])
	}
	again, _ := Load(path)
	if again.Profiles[0].ID != got.Profiles[0].ID {
		t.Error("id changed between loads")
	}
}

func TestResolveByIDAddressAndUniqueName(t *testing.T) {
	a, b, c := valid(), valid(), valid()
	a.Name, b.Name, c.Name = "db", "db", "solo"
	a.Folder, b.Folder = "prod", "stage"
	s := Store{Version: Version, Profiles: []Profile{a, b, c}}
	s.Normalize(false)
	if p, err := s.Resolve("prod/db"); err != nil || p.Folder != "prod" {
		t.Errorf("Resolve(prod/db) = %+v, %v", p, err)
	}
	if p, err := s.Resolve(s.Profiles[1].ID); err != nil || p.Folder != "stage" {
		t.Errorf("Resolve(id) = %+v, %v", p, err)
	}
	if p, err := s.Resolve("solo"); err != nil || p.Name != "solo" {
		t.Errorf("Resolve(solo) = %+v, %v", p, err)
	}
	_, err := s.Resolve("db")
	if err == nil || !strings.Contains(err.Error(), "prod/db") || !strings.Contains(err.Error(), "stage/db") {
		t.Errorf("Resolve(db) error = %v, want both candidates listed", err)
	}
	if _, err := s.Resolve("ghost"); err == nil {
		t.Error("Resolve(ghost) succeeded")
	}
	if _, ok := s.Find("db"); ok {
		t.Error("Find(db) must not guess between two profiles")
	}
}

func TestJumpCyclesAreFoundByID(t *testing.T) {
	a, b := valid(), valid()
	a.Name, b.Name = "x", "x"
	a.Folder, b.Folder = "one", "two"
	a.ID, b.ID = "@aaaaaaaaaaaa", "@bbbbbbbbbbbb"
	a.Jump, b.Jump = b.ID, a.ID
	if err := (Store{Version: Version, Profiles: []Profile{a, b}}).Validate(); err == nil {
		t.Fatal("cycle between same-named profiles not detected")
	}
}
