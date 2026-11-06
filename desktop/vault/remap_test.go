package vault

import (
	"path/filepath"
	"testing"
)

func TestRemapRekeysEntriesAndSurvivesRelock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "vault.age")
	v := New(path)
	if err := v.Create("master-pass-1"); err != nil {
		t.Fatal(err)
	}
	if err := v.Set("web", "s3cret", "old-binding"); err != nil {
		t.Fatal(err)
	}
	if err := v.Set("keep", "other", "b"); err != nil {
		t.Fatal(err)
	}
	err := v.Remap(func(k, b string) (string, string, bool) {
		if k == "web" {
			return "@aaaaaaaaaaaa", "new-binding", true
		}
		return "", "", false
	})
	if err != nil {
		t.Fatal(err)
	}
	v.Lock()
	if err := New(path).Unlock("master-pass-1"); err != nil {
		t.Fatal(err)
	}
	if err := v.Unlock("master-pass-1"); err != nil {
		t.Fatal(err)
	}
	if v.Has("web") {
		t.Error("old key still present")
	}
	got, ok, err := v.Get("@aaaaaaaaaaaa", "new-binding")
	if err != nil || !ok || got != "s3cret" {
		t.Errorf("Get(new key) = %q, %v, %v", got, ok, err)
	}
	if !v.Has("keep") {
		t.Error("untouched entry was lost")
	}
}
