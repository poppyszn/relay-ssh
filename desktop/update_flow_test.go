package main

import (
	"archive/zip"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/poppyszn/relay-ssh/desktop/internal/release"
	"github.com/wailsapp/wails/v3/pkg/updater"
)

type fakeHost struct{}

func (fakeHost) Emit(string, ...any) bool                              { return true }
func (fakeHost) OnEvent(string, func(any)) func()                      { return func() {} }
func (fakeHost) OpenWindow(updater.WindowOptions) updater.WindowHandle { return nil }
func (fakeHost) Quit()                                                 {}

// appZip is a stand-in release: a zip with one top-level entry, as the real macOS release has.
func appZip(t *testing.T, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("Relay.app/Contents/MacOS/relay")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = w.Write([]byte(content))
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// serveRelease publishes a manifest signed by signer for the artifact bytes, then serves artifactBytes
// (which a test can make differ from what was signed, as a tampered download would).
func serveRelease(t *testing.T, signer ed25519.PrivateKey, signed, artifactBytes []byte) string {
	t.Helper()
	dir := t.TempDir()
	platform, arch := "darwin", "arm64"
	name := "Relay-" + platform + "-" + arch + ".zip"
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, signed, 0o600); err != nil {
		t.Fatal(err)
	}
	manifest, err := release.Build("2.0.0", "New things.", time.Now(), []release.Artifact{{Platform: platform, Arch: arch, Path: path}}, signer)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/update.json", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(manifest) })
	mux.HandleFunc("/"+name, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(artifactBytes) })
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv.URL + "/update.json"
}

// stage runs the real updater (real provider, real verification) against a local release server.
func stage(t *testing.T, url string, pinned ed25519.PublicKey) error {
	t.Helper()
	cfg, err := updateConfig("1.0.0", url, pinned)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Platform, cfg.Arch = "darwin", "arm64"
	u := updater.New(fakeHost{})
	if err := u.Init(cfg); err != nil {
		t.Fatal(err)
	}
	rel, err := u.Check(t.Context())
	if err != nil || rel == nil {
		t.Fatalf("Check = %v, %v", rel, err)
	}
	return u.DownloadAndInstall(t.Context())
}

func TestAnUpdateSignedWithTheReleaseKeyIsAccepted(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	zipped := appZip(t, "new build")
	if err := stage(t, serveRelease(t, priv, zipped, zipped), pub); err != nil {
		t.Fatalf("a correctly signed update must install: %v", err)
	}
}

func TestAnUpdateSignedWithAnotherKeyIsRejected(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(rand.Reader)
	_, attacker, _ := ed25519.GenerateKey(rand.Reader)
	zipped := appZip(t, "evil build")
	if err := stage(t, serveRelease(t, attacker, zipped, zipped), pub); err == nil {
		t.Fatal("an update signed by a key the app does not pin must be rejected")
	}
}

func TestATamperedDownloadIsRejected(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	if err := stage(t, serveRelease(t, priv, appZip(t, "real build"), appZip(t, "swapped build")), pub); err == nil {
		t.Fatal("a download that differs from what was signed must be rejected")
	}
}
