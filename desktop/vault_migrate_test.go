package main

import (
	"testing"

	"github.com/poppyszn/relay-ssh/internal/config"
)

// A password saved before connections had ids is keyed by name; unlocking moves it to the id,
// binding included (a jump host used to be recorded by name), so it still fills.
func TestUnlockMovesNameKeyedPasswordsToIds(t *testing.T) {
	p, v := serviceTestHome(t)
	ps := ProfilesService{}
	bastion := Profile{Name: "bastion", Host: "b.example.com", User: "ops", Port: 22, Auth: "agent"}
	if err := ps.Save("", bastion); err != nil {
		t.Fatal(err)
	}
	web := Profile{Name: "web", Host: "10.0.0.5", User: "alice", Port: 22, Auth: "password", Jump: idOf(t, "bastion")}
	if err := ps.Save("", web); err != nil {
		t.Fatal(err)
	}
	store, _ := config.Load(p.StoreFile)
	saved, _ := store.Find("web")
	legacy := saved
	legacy.Jump = "bastion" // how the old scheme recorded it
	if err := v.Set("web", "old-pass", bindingOf(legacy)); err != nil {
		t.Fatal(err)
	}
	v.Lock()
	if err := (VaultService{}).Unlock("correct-horse-battery-staple"); err != nil {
		t.Fatal(err)
	}
	if v.Has("web") {
		t.Error("name key was not moved")
	}
	got, ok, err := v.Get(saved.ID, bindingOf(saved))
	if err != nil || !ok || got != "old-pass" {
		t.Errorf("after migration Get = %q, %v, %v", got, ok, err)
	}
}

func TestTwoConnectionsMayShareANameInDifferentFolders(t *testing.T) {
	serviceTestHome(t)
	ps := ProfilesService{}
	for _, folder := range []string{"prod", "stage"} {
		if err := ps.Save("", Profile{Name: "db", Folder: folder, Host: "h-" + folder, User: "u", Port: 22, Auth: "agent"}); err != nil {
			t.Fatalf("save db in %s: %v", folder, err)
		}
	}
	list, _ := ps.List()
	if len(list) != 2 || list[0].ID == "" || list[0].ID == list[1].ID {
		t.Fatalf("List = %+v", list)
	}
	if err := ps.Delete(list[0].ID); err != nil {
		t.Fatal(err)
	}
	if list, _ = ps.List(); len(list) != 1 {
		t.Fatalf("Delete by id removed %d, left %d", 1, len(list))
	}
}
