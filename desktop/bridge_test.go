package main

import (
	"reflect"
	"sort"
	"strings"
	"testing"
)

// Everything the app exposes to the page. The page (a web view) can call every one of these, so a script that
// ever got into it could too. This is the reviewed inventory: adding a bound method fails this test until
// someone puts it here with a category, which is the moment to ask "what could a hostile page do with this?".
//
//	read        reads state the user can already see
//	config      changes saved connections, folders, snippets (the source of truth the user can edit anyway)
//	local-file  touches a local file; refuses any path the user did not choose (see pathgrants.go)
//	picker      opens a native dialog; the only way a path becomes allowed
//	remote      acts on a remote host over ssh as the user (file browser, port forwards)
//	vault       manages the encrypted vault; secrets need the master password
//	secret      returns a secret: RevealSecret needs the master password, Credentials is one-shot for a session
//	session     opens or closes a terminal or remote desktop session
//	dev         benchmark and smoke harness only
var bridge = map[string]string{
	"FilesService.Download":              "local-file",
	"FilesService.Upload":                "local-file",
	"FilesService.List":                  "remote",
	"FilesService.Mkdir":                 "remote",
	"FilesService.Remove":                "remote",
	"FilesService.Rename":                "remote",
	"FilesService.PickDownloadPath":      "picker",
	"FilesService.PickUploadPath":        "picker",
	"ForwardsService.Add":                "remote",
	"ForwardsService.Cancel":             "remote",
	"ForwardsService.Status":             "remote",
	"ProfilesService.CreateFolder":       "config",
	"ProfilesService.Delete":             "config",
	"ProfilesService.DeleteFolder":       "config",
	"ProfilesService.Folders":            "read",
	"ProfilesService.List":               "read",
	"ProfilesService.MoveProfile":        "config",
	"ProfilesService.PickKeyFile":        "picker",
	"ProfilesService.Preview":            "read",
	"ProfilesService.RenameFolder":       "config",
	"ProfilesService.Save":               "config",
	"ProfilesService.SaveWithPassword":   "vault",
	"RdpService.Close":                   "session",
	"RdpService.CloseAll":                "session",
	"RdpService.Credentials":             "secret",
	"RdpService.Note":                    "session",
	"RdpService.Open":                    "session",
	"RdpService.TrustCert":               "session",
	"SnippetsService.Delete":             "config",
	"SnippetsService.List":               "read",
	"SnippetsService.Save":               "config",
	"TerminalService.BenchMode":          "read",
	"TerminalService.CustomChrome":       "read",
	"TerminalService.Endpoint":           "session",
	"TerminalService.IntegrationSnippet": "read",
	"TerminalService.ReportBench":        "dev",
	"ToolsService.AdoptKey":              "local-file",
	"ToolsService.Backup":                "local-file",
	"ToolsService.DefaultBackupName":     "read",
	"ToolsService.FixHealth":             "config",
	"ToolsService.Health":                "read",
	"ToolsService.ImportSSHConfig":       "config",
	"ToolsService.KeyNeedsPassphrase":    "local-file",
	"ToolsService.Keys":                  "read",
	"ToolsService.NeedsPassphrase":       "read",
	"ToolsService.PickBackupPath":        "picker",
	"ToolsService.PickKeyToAdopt":        "picker",
	"ToolsService.PickRestorePath":       "picker",
	"ToolsService.RememberPassphrase":    "vault",
	"ToolsService.Restore":               "local-file",
	"ToolsService.Version":               "read",
	"VaultService.ChangeMaster":          "vault",
	"VaultService.Create":                "vault",
	"VaultService.ForgetSecret":          "vault",
	"VaultService.HasSecret":             "read",
	"VaultService.Lock":                  "vault",
	"VaultService.Reset":                 "vault",
	"VaultService.RevealSecret":          "secret",
	"VaultService.SetIdleTimeout":        "vault",
	"VaultService.Status":                "read",
	"VaultService.Unlock":                "vault",
}

func boundMethods() []string {
	var got []string
	for _, svc := range []any{&TerminalService{}, &ProfilesService{}, &ToolsService{}, &FilesService{}, &ForwardsService{}, &SnippetsService{}, &VaultService{}, &RdpService{}} {
		ty := reflect.TypeOf(svc)
		for i := 0; i < ty.NumMethod(); i++ {
			got = append(got, ty.Elem().Name()+"."+ty.Method(i).Name)
		}
	}
	sort.Strings(got)
	return got
}

func TestEveryBoundMethodIsReviewed(t *testing.T) {
	seen := map[string]bool{}
	for _, m := range boundMethods() {
		seen[m] = true
		if _, ok := bridge[m]; !ok {
			t.Errorf("%s is exposed to the page but not in the bridge inventory (bridge_test.go): decide what a hostile page could do with it, then add it with a category", m)
		}
	}
	for m := range bridge {
		if !seen[m] {
			t.Errorf("%s is in the bridge inventory but no longer exists: remove it", m)
		}
	}
}

// Only a few methods may return a secret or touch local files; the categories must say so.
func TestBridgeCategoriesAreKnownAndTight(t *testing.T) {
	known := map[string]bool{"read": true, "config": true, "local-file": true, "picker": true, "remote": true, "vault": true, "secret": true, "session": true, "dev": true}
	var secrets, localFiles []string
	for m, c := range bridge {
		if !known[c] {
			t.Errorf("%s has unknown category %q", m, c)
		}
		switch c {
		case "secret":
			secrets = append(secrets, m)
		case "local-file":
			localFiles = append(localFiles, m)
		}
	}
	sort.Strings(secrets)
	sort.Strings(localFiles)
	if got := strings.Join(secrets, ","); got != "RdpService.Credentials,VaultService.RevealSecret" {
		t.Errorf("secret-returning methods changed: %s (update the secrets test and SECURITY.md too)", got)
	}
	if got := strings.Join(localFiles, ","); got != "FilesService.Download,FilesService.Upload,ToolsService.AdoptKey,ToolsService.Backup,ToolsService.KeyNeedsPassphrase,ToolsService.Restore" {
		t.Errorf("methods that touch local files changed: %s (each must call grants.require; see pathgrants.go)", got)
	}
}
