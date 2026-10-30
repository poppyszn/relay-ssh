package main

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/poppyszn/relay-ssh/internal/fsutil"
)

// CertDecision is the trust verdict for an RDP host's TLS certificate.
type CertDecision int

const (
	CertTrust CertDecision = iota
	CertUnknown
	CertMismatch
)

// FingerprintSHA256 formats a DER certificate's SHA-256 digest in standard colon-separated hex.
func FingerprintSHA256(der []byte) string {
	sum := sha256.Sum256(der)
	var parts []string
	for _, b := range sum {
		parts = append(parts, fmt.Sprintf("%02X", b))
	}
	return "SHA256:" + strings.Join(parts, ":")
}

type rdpHostsFile struct {
	Hosts map[string]string `toml:"hosts"`
}

// RdpCertStore manages TOFU certificate fingerprints stored in rdp_hosts.toml.
type RdpCertStore struct {
	path  string
	mu    sync.RWMutex
	hosts map[string]string
}

// LoadRdpCertStore reads rdp_hosts.toml, returning an empty store if the file does not exist.
func LoadRdpCertStore(path string) (*RdpCertStore, error) {
	s := &RdpCertStore{
		path:  path,
		hosts: make(map[string]string),
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	var f rdpHostsFile
	if _, err := toml.Decode(string(data), &f); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if f.Hosts != nil {
		s.hosts = f.Hosts
	}
	return s, nil
}

// Check evaluates the fingerprint against stored certificates for the endpoint (host:port).
func (s *RdpCertStore) Check(endpoint, fingerprint string) (CertDecision, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stored, exists := s.hosts[endpoint]
	if !exists {
		return CertUnknown, ""
	}
	if stored == fingerprint {
		return CertTrust, stored
	}
	return CertMismatch, stored
}

// Trust records or updates the trusted fingerprint for the endpoint and writes to disk atomically.
func (s *RdpCertStore) Trust(endpoint, fingerprint string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.hosts[endpoint] = fingerprint
	return s.saveLocked()
}

func (s *RdpCertStore) saveLocked() error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	keys := make([]string, 0, len(s.hosts))
	for k := range s.hosts {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b bytes.Buffer
	b.WriteString("# Managed by Relay. Trust-on-first-use certificate fingerprints for RDP hosts.\n[hosts]\n")
	for _, k := range keys {
		fmt.Fprintf(&b, "%q = %q\n", k, s.hosts[k])
	}

	return fsutil.WriteFile(s.path, b.Bytes(), 0o600)
}
