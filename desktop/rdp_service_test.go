package main

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poppyszn/relay-ssh/desktop/vault"
	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/paths"
)

func setupRdpTestEnv(t *testing.T) (*RdpService, *RdpGateway, *vault.Vault) {
	t.Helper()
	p, v := serviceTestHome(t)

	certPath := filepath.Join(filepath.Dir(p.StoreFile), "rdp_hosts.toml")
	cs, err := LoadRdpCertStore(certPath)
	if err != nil {
		t.Fatalf("LoadRdpCertStore failed: %v", err)
	}

	gw, err := NewRdpGateway(cs, nil)
	if err != nil {
		t.Fatalf("NewRdpGateway failed: %v", err)
	}
	t.Cleanup(func() {
		_ = gw.Close()
	})

	svc := NewRdpService(gw)
	t.Cleanup(func() {
		svc.CloseAll()
	})

	return svc, gw, v
}

func TestRdpServiceOpen_Direct(t *testing.T) {
	svc, _, _ := setupRdpTestEnv(t)
	ps := ProfilesService{}

	prof := Profile{
		Protocol: config.ProtocolRDP,
		Name:     "win-dev",
		Host:     "192.168.1.100",
		User:     "Administrator",
		Domain:   "CORP",
		Port:     3389,
	}
	if err := ps.Save("", prof); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	info, err := svc.Open("win-dev")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if info.SessionID == "" {
		t.Errorf("expected non-empty SessionID")
	}
	if !strings.HasPrefix(info.WSURL, "ws://127.0.0.1:") {
		t.Errorf("expected ws://127.0.0.1: URL, got %q", info.WSURL)
	}
	if info.Token == "" {
		t.Errorf("expected non-empty Token")
	}
	if info.Host != "192.168.1.100" || info.User != "Administrator" || info.Domain != "CORP" || info.Port != 3389 {
		t.Errorf("unexpected connection info: %+v", info)
	}
	if info.HasSavedPassword {
		t.Errorf("expected HasSavedPassword=false")
	}
}

func TestRdpServiceOpen_NotFoundOrNotRDP(t *testing.T) {
	svc, _, _ := setupRdpTestEnv(t)
	ps := ProfilesService{}

	if _, err := svc.Open("non-existent"); err == nil {
		t.Errorf("expected error for non-existent profile")
	}

	sshProf := Profile{
		Name: "ssh-box",
		Host: "10.0.0.1",
		User: "ubuntu",
		Port: 22,
		Auth: "agent",
	}
	if err := ps.Save("", sshProf); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if _, err := svc.Open("ssh-box"); err == nil || !strings.Contains(err.Error(), "not an RDP connection") {
		t.Errorf("expected 'not an RDP connection' error, got %v", err)
	}
}

func TestRdpServiceCredentials_OneShot(t *testing.T) {
	svc, _, _ := setupRdpTestEnv(t)
	ps := ProfilesService{}

	prof := Profile{
		Protocol: config.ProtocolRDP,
		Name:     "win-srv",
		Host:     "192.168.1.200",
		User:     "admin",
		Port:     3389,
	}
	secret := "SecretP@ssword123!"
	if err := ps.SaveWithPassword("", prof, secret); err != nil {
		t.Fatalf("SaveWithPassword failed: %v", err)
	}

	info, err := svc.Open("win-srv")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if !info.HasSavedPassword {
		t.Fatalf("expected HasSavedPassword=true")
	}

	// First call succeeds
	cred, err := svc.Credentials(info.SessionID)
	if err != nil {
		t.Fatalf("Credentials first call failed: %v", err)
	}
	if cred != secret {
		t.Fatalf("expected secret %q, got %q", secret, cred)
	}

	// Second call fails (one-shot)
	_, err = svc.Credentials(info.SessionID)
	if err == nil || !strings.Contains(err.Error(), "already retrieved") {
		t.Fatalf("expected 'already retrieved' error on second call, got %v", err)
	}
}

func TestRdpServiceCredentials_VaultLocked(t *testing.T) {
	svc, _, v := setupRdpTestEnv(t)
	ps := ProfilesService{}

	prof := Profile{
		Protocol: config.ProtocolRDP,
		Name:     "win-locked",
		Host:     "192.168.1.201",
		User:     "admin",
		Port:     3389,
	}
	if err := ps.SaveWithPassword("", prof, "Pass123"); err != nil {
		t.Fatalf("SaveWithPassword failed: %v", err)
	}

	info, err := svc.Open("win-locked")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	v.Lock()

	_, err = svc.Credentials(info.SessionID)
	if !errors.Is(err, vault.ErrVaultLocked) {
		t.Fatalf("expected ErrVaultLocked, got %v", err)
	}
}

func TestRdpServiceCredentials_BindingMismatch(t *testing.T) {
	svc, _, _ := setupRdpTestEnv(t)
	ps := ProfilesService{}

	prof := Profile{
		Protocol: config.ProtocolRDP,
		Name:     "win-tampered",
		Host:     "192.168.1.202",
		User:     "admin",
		Port:     3389,
	}
	if err := ps.SaveWithPassword("", prof, "Pass123"); err != nil {
		t.Fatalf("SaveWithPassword failed: %v", err)
	}

	// Edit connection outside app
	p, err := paths.Default()
	if err != nil {
		t.Fatal(err)
	}
	store, _ := config.Load(p.StoreFile)
	for i := range store.Profiles {
		if store.Profiles[i].Name == "win-tampered" {
			store.Profiles[i].Host = "evil.example.com"
		}
	}
	if err := config.Save(p.StoreFile, store); err != nil {
		t.Fatal(err)
	}

	info, err := svc.Open("win-tampered")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	_, err = svc.Credentials(info.SessionID)
	if !errors.Is(err, vault.ErrBindingChanged) {
		t.Fatalf("expected ErrBindingChanged, got %v", err)
	}
}

func TestRdpServiceCredentials_SSHBindingDoesNotFillRDP(t *testing.T) {
	svc, _, v := setupRdpTestEnv(t)
	ps := ProfilesService{}

	// Save an SSH profile with password
	sshProf := Profile{
		Name: "shared-host",
		Host: "10.0.0.50",
		User: "admin",
		Port: 22,
		Auth: "password",
	}
	if err := ps.SaveWithPassword("", sshProf, "SSHSecret"); err != nil {
		t.Fatalf("SaveWithPassword failed: %v", err)
	}

	// Create an RDP profile with same host, user, port 22
	rdpProf := Profile{
		Protocol: config.ProtocolRDP,
		Name:     "rdp-host",
		Host:     "10.0.0.50",
		User:     "admin",
		Port:     22,
	}
	if err := ps.Save("", rdpProf); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Try to get credentials with RDP binding for "shared-host"
	rdpBinding := fmt.Sprintf("rdp|10.0.0.50|admin|22||")
	if _, _, err := v.Get("shared-host", rdpBinding); !errors.Is(err, vault.ErrBindingChanged) {
		t.Fatalf("expected ErrBindingChanged when looking up SSH password with RDP binding, got %v", err)
	}

	// RdpService Open & Credentials on rdp-host has no password saved
	info, err := svc.Open("rdp-host")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if info.HasSavedPassword {
		t.Fatalf("expected HasSavedPassword=false")
	}
	_, err = svc.Credentials(info.SessionID)
	if err == nil {
		t.Fatalf("expected error retrieving credentials for connection with no password")
	}
}

func TestRdpServiceClose_IdempotentAndCleansUp(t *testing.T) {
	svc, _, _ := setupRdpTestEnv(t)
	ps := ProfilesService{}

	prof := Profile{
		Protocol: config.ProtocolRDP,
		Name:     "win-close-test",
		Host:     "192.168.1.205",
		User:     "admin",
		Port:     3389,
	}
	if err := ps.Save("", prof); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	info, err := svc.Open("win-close-test")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if err := svc.Close(info.SessionID); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
	// Idempotent second close
	if err := svc.Close(info.SessionID); err != nil {
		t.Fatalf("Close second call failed: %v", err)
	}

	// Credentials fails on closed session
	if _, err := svc.Credentials(info.SessionID); err == nil || !strings.Contains(err.Error(), "session not found") {
		t.Fatalf("expected 'session not found' error, got %v", err)
	}
}

func TestRdpServiceOpen_WithJumpTunnel(t *testing.T) {
	svc, _, _ := setupRdpTestEnv(t)
	ps := ProfilesService{}

	// Setup fake ssh
	tmpDir := t.TempDir()
	fakeSSH := filepath.Join(tmpDir, "ssh")
	script := `#!/bin/sh
port=""
for arg in "$@"; do
    case "$arg" in
        127.0.0.1:*:*)
            port=$(echo "$arg" | cut -d: -f2)
            ;;
    esac
done
if [ -n "$port" ]; then
    exec python3 -c '
import socket, sys, time
s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
s.bind(("127.0.0.1", int(sys.argv[1])))
s.listen(1)
while True:
    time.sleep(1)
' "$port"
fi
exec sleep 10
`
	if err := os.WriteFile(fakeSSH, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmpDir+":"+os.Getenv("PATH"))

	// Save jump host (SSH)
	jumpProf := Profile{
		Name: "jump-srv",
		Host: "jump.example.com",
		User: "bastion",
		Port: 22,
		Auth: "agent",
	}
	if err := ps.Save("", jumpProf); err != nil {
		t.Fatalf("Save jump host failed: %v", err)
	}

	// Save RDP profile with Jump
	rdpProf := Profile{
		Protocol: config.ProtocolRDP,
		Name:     "win-jump",
		Host:     "10.10.10.10",
		User:     "admin",
		Port:     3389,
		Jump:     "jump-srv",
	}
	if err := ps.Save("", rdpProf); err != nil {
		t.Fatalf("Save rdp host failed: %v", err)
	}

	info, err := svc.Open("win-jump")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	if info.SessionID == "" {
		t.Fatalf("expected non-empty SessionID")
	}

	// Close session cleans up tunnel
	if err := svc.Close(info.SessionID); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

// The tab reports where it is in a connection (client loaded, connecting, how it ended) so the log can show where
// time goes. It carries stage names and error text only, is bounded, and an unknown session is ignored.
func TestRdpServiceNoteLogsStagesWithoutTrustingTheText(t *testing.T) {
	svc, _, _ := setupRdpTestEnv(t)
	var logged bytes.Buffer
	log.SetOutput(&logged)
	defer log.SetOutput(os.Stderr)

	svc.mu.Lock()
	svc.sessions["abcdef0123456789"] = &rdpActiveSession{id: "abcdef0123456789", connection: "win-dev", createdAt: time.Now().Add(-3 * time.Second)}
	svc.mu.Unlock()

	svc.Note("abcdef0123456789", "client loaded")
	svc.Note("abcdef0123456789", strings.Repeat("x", 5000)+"\nforged log line")
	svc.Note("no-such-session", "ignored")

	out := logged.String()
	if !strings.Contains(out, "rdp session abcdef01 (win-dev, +3s): client loaded") {
		t.Errorf("stage not logged with its elapsed time:\n%s", out)
	}
	if strings.Contains(out, "forged log line") || strings.Contains(out, "\nforged") {
		t.Errorf("an over-long or multi-line note was not cut:\n%s", out)
	}
	if strings.Contains(out, "no-such-session") || strings.Contains(out, "ignored") {
		t.Errorf("a note for an unknown session was logged:\n%s", out)
	}
}

func TestRdpTokenOutlivesASlowStart(t *testing.T) {
	if rdpTokenLifetime < 3*time.Minute {
		t.Fatalf("a gateway token must survive a slow first start (client load, certificate and password prompts); lifetime is %s", rdpTokenLifetime)
	}
}
