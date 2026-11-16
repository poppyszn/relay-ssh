// Package release builds the signed update manifest that installed copies of Relay check.
package release

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Artifact is one downloadable build: the platform and arch it is for, and the file.
type Artifact struct{ Platform, Arch, Path string }

type manifestArtifact struct {
	URL           string `json:"url"`
	Filename      string `json:"filename"`
	Size          int64  `json:"size"`
	Platform      string `json:"platform"`
	Arch          string `json:"arch"`
	DigestAlgo    string `json:"digestAlgo"`
	Digest        string `json:"digest"`
	SignatureAlgo string `json:"signatureAlgo"`
	Signature     string `json:"signature"`
}

type manifest struct {
	SchemaVersion int                `json:"schemaVersion"`
	Version       string             `json:"version"`
	Notes         string             `json:"notes"`
	PublishedAt   string             `json:"publishedAt"`
	Artifacts     []manifestArtifact `json:"artifacts"`
}

var semver = regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z][0-9A-Za-z.-]*)?$`)

// Build returns the manifest JSON. Each artifact is listed by its bare filename, which the updater
// resolves next to the manifest, and carries the sha256 of its bytes plus an ed25519 signature of that
// digest: the scheme the updater verifies against the public key pinned in the app.
func Build(version, notes string, published time.Time, arts []Artifact, key ed25519.PrivateKey) ([]byte, error) {
	if !semver.MatchString(version) {
		return nil, fmt.Errorf("version %q is not MAJOR.MINOR.PATCH (no leading v)", version)
	}
	if len(arts) == 0 {
		return nil, errors.New("no artifacts")
	}
	m := manifest{SchemaVersion: 1, Version: version, Notes: notes, PublishedAt: published.UTC().Format(time.RFC3339)}
	for _, a := range arts {
		digest, size, err := sha256File(a.Path)
		if err != nil {
			return nil, err
		}
		m.Artifacts = append(m.Artifacts, manifestArtifact{
			URL: filepath.Base(a.Path), Filename: filepath.Base(a.Path), Size: size,
			Platform: a.Platform, Arch: a.Arch,
			DigestAlgo: "sha256", Digest: base64.StdEncoding.EncodeToString(digest),
			SignatureAlgo: "ed25519", Signature: base64.StdEncoding.EncodeToString(ed25519.Sign(key, digest)),
		})
	}
	return json.MarshalIndent(m, "", "  ")
}

func sha256File(path string) ([]byte, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return nil, 0, err
	}
	return h.Sum(nil), n, nil
}

// ParseKey reads a base64 ed25519 private key: the 64-byte form or the 32-byte seed.
func ParseKey(s string) (ed25519.PrivateKey, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(s))
	if err != nil {
		return nil, errors.New("signing key is not valid base64")
	}
	switch len(raw) {
	case ed25519.PrivateKeySize:
		return ed25519.PrivateKey(raw), nil
	case ed25519.SeedSize:
		return ed25519.NewKeyFromSeed(raw), nil
	}
	return nil, fmt.Errorf("signing key is %d bytes, want %d or %d", len(raw), ed25519.SeedSize, ed25519.PrivateKeySize)
}
