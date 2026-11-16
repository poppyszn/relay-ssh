package main

import (
	"context"
	"crypto/ed25519"
	_ "embed"
	"encoding/base64"
	"errors"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/endpoint"
)

// updateManifestURL is the signed manifest every release publishes next to its downloads.
// GitHub redirects "latest" to the newest non-draft, non-prerelease release.
const updateManifestURL = "https://github.com/poppyszn/relay-ssh/releases/latest/download/update.json"

// updatePubKey is the ed25519 public key release builds are verified against. Only the matching
// private key (a CI secret, never in this repository) can sign an update this app will install.
//
//go:embed update_pubkey.txt
var updatePubKey string

// updatePublicKey decodes the pinned key.
func updatePublicKey() (ed25519.PublicKey, error) {
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(updatePubKey))
	if err != nil || len(raw) != ed25519.PublicKeySize {
		return nil, errors.New("update public key is not a valid ed25519 key")
	}
	return ed25519.PublicKey(raw), nil
}

// releaseVersion matches the version a release build is stamped with. Anything else (dev, a commit
// hash, `git describe` output) is a local build, which never updates itself.
var releaseVersion = regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z][0-9A-Za-z.-]*)?$`)

// updateEngine is the part of the Wails updater this service uses.
type updateEngine interface {
	Check(ctx context.Context) (*updater.Release, error)
	DownloadAndInstall(ctx context.Context) error
	Restart(ctx context.Context) error
}

// UpdateInfo is what the UI shows about updates.
type UpdateInfo struct {
	Current   string `json:"current"`
	Latest    string `json:"latest"`
	Available bool   `json:"available"`
	Notes     string `json:"notes"`
	Enabled   bool   `json:"enabled"`
}

// UpdateService checks for, downloads and installs new releases. The download is verified against
// the pinned public key before anything on disk is replaced.
type UpdateService struct {
	engine  updateEngine
	version string

	mu        sync.Mutex
	found     *updater.Release
	installed bool
}

func newUpdateService(engine updateEngine, version string) *UpdateService {
	return &UpdateService{engine: engine, version: version}
}

// updateConfig is the updater configuration: the signed manifest at url, verified with key.
func updateConfig(version, url string, key ed25519.PublicKey) (updater.Config, error) {
	src, err := endpoint.New(endpoint.Config{URL: url})
	if err != nil {
		return updater.Config{}, err
	}
	return updater.Config{
		CurrentVersion: version,
		Providers:      []updater.Provider{src},
		PublicKey:      key,
		Window:         updater.WindowNone, // Relay shows its own prompt
	}, nil
}

// attach connects the service to the app's updater. Local builds are left unattached; they never check.
func (s *UpdateService) attach(u *updater.Updater) error {
	if !s.enabled() {
		return nil
	}
	key, err := updatePublicKey()
	if err != nil {
		return err
	}
	url := updateManifestURL
	if o := os.Getenv("RELAY_UPDATE_URL"); o != "" {
		url = o // for rehearsing a release against a local server; the signature is still checked
	}
	cfg, err := updateConfig(s.version, url, key)
	if err != nil {
		return err
	}
	s.engine = u
	return u.Init(cfg)
}

func (s *UpdateService) enabled() bool { return releaseVersion.MatchString(s.version) }

// Info reports the running version and whether this build can update itself.
func (s *UpdateService) Info() UpdateInfo {
	return UpdateInfo{Current: s.version, Enabled: s.enabled()}
}

// Check asks the release feed whether a newer version exists.
func (s *UpdateService) Check() (UpdateInfo, error) {
	info := s.Info()
	if !info.Enabled {
		return info, errors.New("this is a development build; it does not check for updates")
	}
	if s.engine == nil {
		return info, errors.New("updates are unavailable in this build")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	rel, err := s.engine.Check(ctx)
	if err != nil {
		return info, err
	}
	s.mu.Lock()
	s.found, s.installed = rel, false
	s.mu.Unlock()
	if rel != nil {
		info.Available, info.Latest, info.Notes = true, rel.Version, rel.Notes
	}
	return info, nil
}

// Install downloads the release the last Check found, verifies its signature and stages it.
// The running app is not touched until Restart.
func (s *UpdateService) Install() error {
	s.mu.Lock()
	found := s.found
	s.mu.Unlock()
	if found == nil {
		return errors.New("there is no update to install; check for updates first")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	if err := s.engine.DownloadAndInstall(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	s.installed = true
	s.mu.Unlock()
	return nil
}

// Restart relaunches Relay as the new version. It needs a successful Install first.
func (s *UpdateService) Restart() error {
	s.mu.Lock()
	ok := s.installed
	s.mu.Unlock()
	if !ok {
		return errors.New("no update is ready to install")
	}
	return s.engine.Restart(context.Background())
}
