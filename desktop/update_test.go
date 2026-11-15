package main

import (
	"context"
	"errors"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
)

type fakeEngine struct {
	rel        *updater.Release
	checkErr   error
	installErr error
	installed  int
	restarted  int
}

func (f *fakeEngine) Check(context.Context) (*updater.Release, error) { return f.rel, f.checkErr }
func (f *fakeEngine) DownloadAndInstall(context.Context) error        { f.installed++; return f.installErr }
func (f *fakeEngine) Restart(context.Context) error                   { f.restarted++; return nil }

func TestUpdatesAreOffForDevBuilds(t *testing.T) {
	for _, v := range []string{"dev", "", "v0.1.0-3-gabc123-dirty", "abc1234", "unknown"} {
		s := newUpdateService(&fakeEngine{}, v)
		if s.Info().Enabled {
			t.Fatalf("version %q must not check for updates", v)
		}
		if _, err := s.Check(); err == nil {
			t.Fatalf("version %q: Check must refuse", v)
		}
	}
}

func TestReleaseVersionsAreEnabled(t *testing.T) {
	for _, v := range []string{"1.2.3", "0.1.0", "2.0.0-rc.1"} {
		if !newUpdateService(&fakeEngine{}, v).Info().Enabled {
			t.Fatalf("version %q should check for updates", v)
		}
	}
}

func TestCheckReportsANewerRelease(t *testing.T) {
	eng := &fakeEngine{rel: &updater.Release{Version: "1.3.0", Notes: "Faster."}}
	s := newUpdateService(eng, "1.2.0")
	got, err := s.Check()
	if err != nil || !got.Available || got.Latest != "1.3.0" || got.Notes != "Faster." || got.Current != "1.2.0" {
		t.Fatalf("Check = %+v, %v", got, err)
	}
}

func TestCheckWhenUpToDate(t *testing.T) {
	s := newUpdateService(&fakeEngine{}, "1.2.0")
	got, err := s.Check()
	if err != nil || got.Available {
		t.Fatalf("Check = %+v, %v", got, err)
	}
}

func TestCheckFailureIsReturned(t *testing.T) {
	s := newUpdateService(&fakeEngine{checkErr: errors.New("offline")}, "1.2.0")
	if _, err := s.Check(); err == nil {
		t.Fatal("a failed check must be an error")
	}
}

func TestInstallNeedsACheckThatFoundAnUpdate(t *testing.T) {
	eng := &fakeEngine{}
	s := newUpdateService(eng, "1.2.0")
	if err := s.Install(); err == nil || eng.installed != 0 {
		t.Fatal("Install before a successful check must not download anything")
	}
	_, _ = s.Check() // up to date
	if err := s.Install(); err == nil || eng.installed != 0 {
		t.Fatal("Install with nothing newer must not download anything")
	}
}

func TestInstallThenRestart(t *testing.T) {
	eng := &fakeEngine{rel: &updater.Release{Version: "1.3.0"}}
	s := newUpdateService(eng, "1.2.0")
	if _, err := s.Check(); err != nil {
		t.Fatal(err)
	}
	if err := s.Restart(); err == nil || eng.restarted != 0 {
		t.Fatal("Restart before Install must not restart")
	}
	if err := s.Install(); err != nil || eng.installed != 1 {
		t.Fatalf("Install: %v (installed %d)", err, eng.installed)
	}
	if err := s.Restart(); err != nil || eng.restarted != 1 {
		t.Fatalf("Restart: %v (restarted %d)", err, eng.restarted)
	}
}

func TestInstallFailureLeavesRestartUnavailable(t *testing.T) {
	eng := &fakeEngine{rel: &updater.Release{Version: "1.3.0"}, installErr: errors.New("signature did not verify")}
	s := newUpdateService(eng, "1.2.0")
	_, _ = s.Check()
	if err := s.Install(); err == nil {
		t.Fatal("a failed install must be an error")
	}
	if err := s.Restart(); err == nil || eng.restarted != 0 {
		t.Fatal("Restart after a failed install must not restart")
	}
}

func TestPinnedKeyIsAValidEd25519Key(t *testing.T) {
	if _, err := updatePublicKey(); err != nil {
		t.Fatal(err)
	}
}

func TestACheckWithoutAnEngineIsAnErrorNotAPanic(t *testing.T) {
	s := newUpdateService(nil, "1.2.0")
	if _, err := s.Check(); err == nil {
		t.Fatal("expected an error")
	}
}
