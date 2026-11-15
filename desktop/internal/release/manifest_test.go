package release

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/endpoint"
)

func writeFile(t *testing.T, dir, name, body string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// The manifest this tool writes must be accepted by the updater's own endpoint provider,
// and the signature must verify against the public key the app pins.
func TestManifestRoundTripsThroughTheUpdaterProvider(t *testing.T) {
	dir := t.TempDir()
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	body := "relay macOS arm64 build"
	arts := []Artifact{
		{Platform: "darwin", Arch: "arm64", Path: writeFile(t, dir, "Relay-darwin-arm64.zip", body)},
		{Platform: "linux", Arch: "amd64", Path: writeFile(t, dir, "Relay-linux-amd64.tar.gz", "linux")},
	}
	manifest, err := Build("1.2.0", "Fixes things.", time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC), arts, priv)
	if err != nil {
		t.Fatal(err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(manifest)
	}))
	defer srv.Close()
	prov, err := endpoint.New(endpoint.Config{URL: srv.URL + "/update.json"})
	if err != nil {
		t.Fatal(err)
	}
	rel, err := prov.Check(t.Context(), updater.CheckRequest{CurrentVersion: "1.1.0", Platform: "darwin", Arch: "arm64"})
	if err != nil || rel == nil {
		t.Fatalf("Check = %v, %v", rel, err)
	}
	if rel.Version != "1.2.0" || rel.Artifact.Filename != "Relay-darwin-arm64.zip" || rel.Notes != "Fixes things." {
		t.Fatalf("unexpected release %+v", rel)
	}
	sum := sha256.Sum256([]byte(body))
	if string(rel.Verification.Digest) != string(sum[:]) {
		t.Fatal("digest does not match the artifact bytes")
	}
	if rel.Verification.SignatureAlgo != "ed25519" || !ed25519.Verify(pub, sum[:], rel.Verification.Signature) {
		t.Fatal("signature does not verify with the public key")
	}
}

func TestBuildRefusesAnEmptyOrMissingArtifact(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	if _, err := Build("1.0.0", "", time.Now(), nil, priv); err == nil {
		t.Fatal("no artifacts must be an error")
	}
	if _, err := Build("1.0.0", "", time.Now(), []Artifact{{Platform: "linux", Arch: "amd64", Path: "/no/such/file"}}, priv); err == nil {
		t.Fatal("a missing artifact file must be an error")
	}
}

func TestBuildRefusesAVersionThatIsNotSemver(t *testing.T) {
	dir := t.TempDir()
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	a := []Artifact{{Platform: "linux", Arch: "amd64", Path: writeFile(t, dir, "x.tar.gz", "x")}}
	for _, v := range []string{"", "dev", "1.2", "v1.2.3", "1.2.3-"} {
		if _, err := Build(v, "", time.Now(), a, priv); err == nil {
			t.Fatalf("version %q must be rejected", v)
		}
	}
}

func TestParseKeyAcceptsASeedOrAFullKey(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(rand.Reader)
	for name, raw := range map[string][]byte{"full": priv, "seed": priv.Seed()} {
		got, err := ParseKey(base64.StdEncoding.EncodeToString(raw) + "\n")
		if err != nil || !priv.Equal(got) {
			t.Fatalf("%s key: %v", name, err)
		}
	}
	if _, err := ParseKey("not base64 !!"); err == nil {
		t.Fatal("garbage must be rejected")
	}
	if _, err := ParseKey(base64.StdEncoding.EncodeToString([]byte("short"))); err == nil {
		t.Fatal("a short key must be rejected")
	}
}
