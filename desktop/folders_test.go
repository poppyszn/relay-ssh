package main

import (
	"strings"
	"testing"
)

func saveAgent(t *testing.T, svc ProfilesService, name, folder string) {
	t.Helper()
	if err := svc.Save("", Profile{Name: name, Host: "h", User: "u", Port: 22, Auth: "agent", Folder: folder}); err != nil {
		t.Fatal(err)
	}
}

func folderList(t *testing.T, svc ProfilesService) string {
	t.Helper()
	f, err := svc.Folders()
	if err != nil {
		t.Fatal(err)
	}
	return strings.Join(f, ",")
}

func TestFolderLifecycle(t *testing.T) {
	t.Setenv("RELAY_SSH_HOME", t.TempDir())
	svc := ProfilesService{}
	saveAgent(t, svc, "web", "prod/eu")
	saveAgent(t, svc, "db", "")

	if got := folderList(t, svc); got != "prod,prod/eu" {
		t.Errorf("implied folders = %q, want prod,prod/eu (ancestors included)", got)
	}

	// an empty folder and subfolder can be created, and nothing may collide with them
	if err := svc.CreateFolder("staging"); err != nil {
		t.Fatal(err)
	}
	if err := svc.CreateFolder("staging/us"); err != nil {
		t.Fatal(err)
	}
	for _, dup := range []string{"staging", "prod", "prod/eu"} {
		if err := svc.CreateFolder(dup); err == nil {
			t.Errorf("creating existing folder %q should fail", dup)
		}
	}
	for _, bad := range []string{"", "/x", "x/", "a//b"} {
		if err := svc.CreateFolder(bad); err == nil {
			t.Errorf("CreateFolder(%q) should be rejected", bad)
		}
	}
	if got := folderList(t, svc); got != "prod,prod/eu,staging,staging/us" {
		t.Errorf("folders = %q", got)
	}

	// moving a connection out of a folder keeps that folder, even though it is now empty
	if err := svc.MoveProfile(idOf(t, "web"), "staging/us"); err != nil {
		t.Fatal(err)
	}
	if got := folderList(t, svc); !strings.Contains(got, "prod/eu") {
		t.Errorf("a folder emptied by a move must survive, folders = %q", got)
	}
	if err := svc.MoveProfile("ghost", "prod"); err == nil {
		t.Error("moving an unknown connection should fail")
	}

	// renaming or moving a folder rewrites everything beneath it
	if err := svc.RenameFolder("staging", "archive/old"); err != nil {
		t.Fatal(err)
	}
	ps, _ := svc.List()
	for _, p := range ps {
		if p.Name == "web" && p.Folder != "archive/old/us" {
			t.Errorf("web folder after rename = %q, want archive/old/us", p.Folder)
		}
	}
	if got := folderList(t, svc); !strings.Contains(got, "archive,archive/old,archive/old/us") || strings.Contains(got, "staging") {
		t.Errorf("folders after rename = %q", got)
	}
	if err := svc.RenameFolder("archive", "archive/old/archive"); err == nil {
		t.Error("moving a folder into its own subfolder must fail")
	}
	if err := svc.RenameFolder("archive", "prod"); err == nil {
		t.Error("renaming onto an existing folder must fail")
	}

	// deleting refuses a folder holding connections, and removes an empty one with its subfolders
	if err := svc.DeleteFolder("archive"); err == nil || !strings.Contains(err.Error(), "web") {
		t.Errorf("deleting a folder with a connection should fail naming it, got %v", err)
	}
	if err := svc.MoveProfile(idOf(t, "web"), ""); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteFolder("archive"); err != nil {
		t.Fatal(err)
	}
	if got := folderList(t, svc); strings.Contains(got, "archive") {
		t.Errorf("archive should be gone, folders = %q", got)
	}
	if err := svc.DeleteFolder("never-existed"); err == nil {
		t.Error("deleting an unknown folder should fail")
	}
}

func TestDeletingOrEditingTheLastConnectionKeepsItsFolder(t *testing.T) {
	t.Setenv("RELAY_SSH_HOME", t.TempDir())
	svc := ProfilesService{}
	saveAgent(t, svc, "a", "keep1")
	saveAgent(t, svc, "b", "keep2")
	if err := svc.Delete("a"); err != nil {
		t.Fatal(err)
	}
	p := Profile{Name: "b", Host: "h", User: "u", Port: 22, Auth: "agent", Folder: "elsewhere"}
	if err := svc.Save("b", p); err != nil {
		t.Fatal(err)
	}
	if got := folderList(t, svc); got != "elsewhere,keep1,keep2" {
		t.Errorf("folders = %q, want keep1 and keep2 to survive", got)
	}
}
