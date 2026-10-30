package main

import (
	"crypto/sha256"
	"fmt"
	"github.com/poppyszn/relay-ssh/desktop/vault"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/paths"
)

// makeFakeSSH creates an executable script that simulates an SSH server prompting for passwords.
// It echos a SHA-256 hash of the received input, NEVER the secret itself.
func makeFakeSSH(t *testing.T, scriptBody string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-ssh.sh")
	content := "#!/bin/sh\nset -e\n" + scriptBody
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	return path
}

func hashSecret(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", sum)
}

func setupTestStoreAndVault(t *testing.T) (paths.Paths, *vault.Vault) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("RELAY_SSH_HOME", home)

	p, err := paths.Default()
	if err != nil {
		t.Fatalf("paths.Default failed: %v", err)
	}
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatalf("EnsureDirs failed: %v", err)
	}

	vpath := filepath.Join(filepath.Dir(p.StoreFile), "vault.age")
	v := vault.New(vpath)
	cleanup := setVaultForTest(v)
	t.Cleanup(cleanup)

	if err := v.Create("correct-horse-battery-staple"); err != nil {
		t.Fatalf("Create vault failed: %v", err)
	}

	return p, v
}

// setBound saves a password the way the app does: bound to the profile as it is in the store.
func setBound(t *testing.T, v *vault.Vault, store config.Store, name, secret string) {
	t.Helper()
	for _, p := range store.Profiles {
		if p.Name == name {
			if err := v.Set(name, secret, bindingOf(p)); err != nil {
				t.Fatalf("Set(%s): %v", name, err)
			}
			return
		}
	}
	t.Fatalf("no profile %q in the test store", name)
}

type ptyReader struct {
	mu  sync.Mutex
	out strings.Builder
}

func newPtyReader(s *session) *ptyReader {
	pr := &ptyReader{}
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := s.pty.Read(buf)
			if n > 0 {
				pr.mu.Lock()
				pr.out.Write(buf[:n])
				pr.mu.Unlock()
			}
			if err != nil {
				return
			}
		}
	}()
	return pr
}

func (pr *ptyReader) readUntil(t *testing.T, timeout time.Duration, want ...string) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		pr.mu.Lock()
		got := pr.out.String()
		pr.mu.Unlock()
		ok := true
		for _, w := range want {
			if !strings.Contains(got, w) {
				ok = false
				break
			}
		}
		if ok {
			return got
		}
		time.Sleep(20 * time.Millisecond)
	}
	pr.mu.Lock()
	defer pr.mu.Unlock()
	return pr.out.String()
}

// Rule 8: The session was started by Relay for a saved connection (not a local shell).
func TestFillRule8LocalShellRefused(t *testing.T) {
	_, v := setupTestStoreAndVault(t)
	_ = v.Set("srv", "secret123", "")

	script := makeFakeSSH(t, `
printf "alice@10.0.0.5's password: "
read -r line
printf "GOT:%s\n" "$(printf "%s" "$line" | shasum -a 256 | awk '{print $1}')"
`)
	s, err := startSession([]string{script}, nil, 80, 24, "local")
	if err != nil {
		t.Fatalf("startSession failed: %v", err)
	}
	defer s.close()
	pr := newPtyReader(s)

	_ = pr.readUntil(t, 2*time.Second, "password: ")

	// Attempt fill on local shell
	s.handleFill(nil, nil, control{
		T:    "fill",
		Kind: "account",
		User: "alice",
		Host: "10.0.0.5",
	})

	time.Sleep(200 * time.Millisecond)
	s.mu.Lock()
	fills := s.totalFills
	s.mu.Unlock()
	if fills != 0 {
		t.Fatalf("expected 0 fills on local shell, got %d", fills)
	}
}

// Rule 7: Never fill after authentication (authed frame or authed state).
func TestFillRule7NeverFillAfterAuthed(t *testing.T) {
	p, v := setupTestStoreAndVault(t)

	store := config.Store{
		Version: config.Version,
		Profiles: []config.Profile{
			{Name: "web", Host: "10.0.0.5", User: "alice", Port: 22, Auth: config.AuthPassword},
		},
	}
	if err := config.Save(p.StoreFile, store); err != nil {
		t.Fatalf("Save store failed: %v", err)
	}
	setBound(t, v, store, "web", "secret-web-99")

	script := makeFakeSSH(t, `
printf "alice@10.0.0.5's password: "
read -r line
printf "GOT:%s\n" "$(printf "%s" "$line" | shasum -a 256 | awk '{print $1}')"
`)
	s, err := startSession([]string{script}, nil, 80, 24, "web")
	if err != nil {
		t.Fatalf("startSession failed: %v", err)
	}
	defer s.close()
	pr := newPtyReader(s)

	_ = pr.readUntil(t, 2*time.Second, "password: ")

	// Mark authed as true (e.g. received authed frame)
	s.mu.Lock()
	s.authed = true
	s.mu.Unlock()

	s.handleFill(nil, nil, control{
		T:    "fill",
		Kind: "account",
		User: "alice",
		Host: "10.0.0.5",
	})

	time.Sleep(200 * time.Millisecond)
	s.mu.Lock()
	fills := s.totalFills
	s.mu.Unlock()
	if fills != 0 {
		t.Fatalf("expected 0 fills after authed, got %d", fills)
	}
}

// Rule 1: Vault must be unlocked and have a secret for the resolved connection.
func TestFillRule1VaultLockedOrMissingSecret(t *testing.T) {
	p, v := setupTestStoreAndVault(t)
	store := config.Store{
		Version: config.Version,
		Profiles: []config.Profile{
			{Name: "web", Host: "10.0.0.5", User: "alice", Port: 22, Auth: config.AuthPassword},
		},
	}
	_ = config.Save(p.StoreFile, store)
	setBound(t, v, store, "web", "secret-web-99")

	script := makeFakeSSH(t, `
printf "alice@10.0.0.5's password: "
read -r line
printf "GOT:%s\n" "$(printf "%s" "$line" | shasum -a 256 | awk '{print $1}')"
`)
	s, err := startSession([]string{script}, nil, 80, 24, "web")
	if err != nil {
		t.Fatalf("startSession failed: %v", err)
	}
	defer s.close()
	pr := newPtyReader(s)

	_ = pr.readUntil(t, 2*time.Second, "password: ")

	// Lock the vault
	v.Lock()

	s.handleFill(nil, nil, control{
		T:    "fill",
		Kind: "account",
		User: "alice",
		Host: "10.0.0.5",
	})

	time.Sleep(200 * time.Millisecond)
	s.mu.Lock()
	fills := s.totalFills
	s.mu.Unlock()
	if fills != 0 {
		t.Fatalf("expected 0 fills when vault locked, got %d", fills)
	}
}

// Rule 2: Target profile's auth method must be password.
func TestFillRule2TargetAuthNotPassword(t *testing.T) {
	p, v := setupTestStoreAndVault(t)
	store := config.Store{
		Version: config.Version,
		Profiles: []config.Profile{
			{Name: "key-host", Host: "10.0.0.5", User: "alice", Port: 22, Auth: config.AuthKey, Key: "id_ed25519"},
		},
	}
	_ = config.Save(p.StoreFile, store)
	setBound(t, v, store, "key-host", "secret-pass")

	script := makeFakeSSH(t, `
printf "alice@10.0.0.5's password: "
read -r line
printf "GOT:%s\n" "$(printf "%s" "$line" | shasum -a 256 | awk '{print $1}')"
`)
	s, err := startSession([]string{script}, nil, 80, 24, "key-host")
	if err != nil {
		t.Fatalf("startSession failed: %v", err)
	}
	defer s.close()
	pr := newPtyReader(s)

	_ = pr.readUntil(t, 2*time.Second, "password: ")

	s.handleFill(nil, nil, control{
		T:    "fill",
		Kind: "account",
		User: "alice",
		Host: "10.0.0.5",
	})

	time.Sleep(200 * time.Millisecond)
	s.mu.Lock()
	fills := s.totalFills
	s.mu.Unlock()
	if fills != 0 {
		t.Fatalf("expected 0 fills when target auth is key, got %d", fills)
	}
}

// Rule 3: Account prompt matches user@host against saved profiles.
// Resolves jump host prompt to jump host profile; refuses ambiguous matches.
func TestFillRule3AccountPromptResolution(t *testing.T) {
	p, v := setupTestStoreAndVault(t)
	targetSecret := "target-secret-password-xyz"
	jumpSecret := "jump-secret-password-abc"

	store := config.Store{
		Version: config.Version,
		Profiles: []config.Profile{
			{Name: "bastion", Host: "bastion.corp.net", User: "op", Port: 22, Auth: config.AuthPassword},
			{Name: "target", Host: "10.0.0.10", User: "ubuntu", Port: 22, Auth: config.AuthPassword, Jump: "bastion"},
		},
	}
	_ = config.Save(p.StoreFile, store)
	setBound(t, v, store, "bastion", jumpSecret)
	setBound(t, v, store, "target", targetSecret)

	// Simulate 2-hop prompt: first jump host, then target
	script := makeFakeSSH(t, `
printf "op@bastion.corp.net's password: "
read -r line1
printf "JUMP_GOT:%s\n" "$(printf "%s" "$line1" | shasum -a 256 | awk '{print $1}')"

printf "ubuntu@10.0.0.10's password: "
read -r line2
printf "TARGET_GOT:%s\n" "$(printf "%s" "$line2" | shasum -a 256 | awk '{print $1}')"
`)
	s, err := startSession([]string{script}, nil, 80, 24, "target")
	if err != nil {
		t.Fatalf("startSession failed: %v", err)
	}
	defer s.close()
	pr := newPtyReader(s)

	_ = pr.readUntil(t, 2*time.Second, "op@bastion.corp.net's password: ")

	// Fill jump host
	s.handleFill(nil, nil, control{
		T:    "fill",
		Kind: "account",
		User: "op",
		Host: "bastion.corp.net",
	})

	got := pr.readUntil(t, 3*time.Second, "ubuntu@10.0.0.10's password: ")
	expectedJumpHash := hashSecret(jumpSecret)
	if !strings.Contains(got, "JUMP_GOT:"+expectedJumpHash) {
		t.Fatalf("jump host did not receive correct jump password hash, output:\n%s", got)
	}

	// Fill target host
	s.handleFill(nil, nil, control{
		T:    "fill",
		Kind: "account",
		User: "ubuntu",
		Host: "10.0.0.10",
	})

	expectedTargetHash := hashSecret(targetSecret)
	got2 := pr.readUntil(t, 3*time.Second, "TARGET_GOT:"+expectedTargetHash)
	if !strings.Contains(got2, "TARGET_GOT:"+expectedTargetHash) {
		t.Fatalf("target host did not receive correct target password hash, output:\n%s", got2)
	}
}

// Rule 3 Ambiguity: More than one profile matching user@host with a secret must refuse.
func TestFillRule3AmbiguousRefusal(t *testing.T) {
	p, v := setupTestStoreAndVault(t)
	store := config.Store{
		Version: config.Version,
		Profiles: []config.Profile{
			{Name: "srv-primary", Host: "10.0.0.5", User: "admin", Port: 22, Auth: config.AuthPassword},
			{Name: "srv-replica", Host: "10.0.0.5", User: "admin", Port: 22, Auth: config.AuthPassword, Jump: "srv-primary"},
		},
	}
	_ = config.Save(p.StoreFile, store)
	setBound(t, v, store, "srv-primary", "pass1")
	setBound(t, v, store, "srv-replica", "pass2")

	script := makeFakeSSH(t, `
printf "admin@10.0.0.5's password: "
read -r line
printf "GOT:%s\n" "$(printf "%s" "$line" | shasum -a 256 | awk '{print $1}')"
`)
	s, err := startSession([]string{script}, nil, 80, 24, "srv-replica")
	if err != nil {
		t.Fatalf("startSession failed: %v", err)
	}
	defer s.close()
	pr := newPtyReader(s)

	_ = pr.readUntil(t, 2*time.Second, "password: ")

	// Ambiguous account prompt
	s.handleFill(nil, nil, control{
		T:    "fill",
		Kind: "account",
		User: "admin",
		Host: "10.0.0.5",
	})

	time.Sleep(200 * time.Millisecond)
	s.mu.Lock()
	fills := s.totalFills
	s.mu.Unlock()
	if fills != 0 {
		t.Fatalf("expected 0 fills on ambiguous match, got %d", fills)
	}
}

// Rule 4: Generic prompts answered with session target secret, ONLY if no jump host.
func TestFillRule4GenericPrompt(t *testing.T) {
	p, v := setupTestStoreAndVault(t)
	targetSecret := "generic-test-secret-42"

	store := config.Store{
		Version: config.Version,
		Profiles: []config.Profile{
			{Name: "direct-host", Host: "direct.net", User: "user", Port: 22, Auth: config.AuthPassword},
			{Name: "jumped-host", Host: "jumped.net", User: "user", Port: 22, Auth: config.AuthPassword, Jump: "direct-host"},
		},
	}
	_ = config.Save(p.StoreFile, store)
	setBound(t, v, store, "direct-host", targetSecret)
	setBound(t, v, store, "jumped-host", "jumped-secret")

	// 1. Direct host: generic prompt works
	script1 := makeFakeSSH(t, `
printf "Password: "
read -r line
printf "GOT:%s\n" "$(printf "%s" "$line" | shasum -a 256 | awk '{print $1}')"
`)
	s1, err := startSession([]string{script1}, nil, 80, 24, "direct-host")
	if err != nil {
		t.Fatalf("startSession failed: %v", err)
	}
	defer s1.close()
	pr1 := newPtyReader(s1)

	_ = pr1.readUntil(t, 2*time.Second, "Password: ")
	s1.handleFill(nil, nil, control{T: "fill", Kind: "generic"})

	expectedHash := hashSecret(targetSecret)
	got1 := pr1.readUntil(t, 3*time.Second, "GOT:"+expectedHash)
	if !strings.Contains(got1, "GOT:"+expectedHash) {
		t.Fatalf("generic prompt failed for direct host:\n%s", got1)
	}

	// 2. Jumped host: generic prompt must be refused
	script2 := makeFakeSSH(t, `
printf "Password: "
read -r line
printf "GOT:%s\n" "$(printf "%s" "$line" | shasum -a 256 | awk '{print $1}')"
`)
	s2, err := startSession([]string{script2}, nil, 80, 24, "jumped-host")
	if err != nil {
		t.Fatalf("startSession failed: %v", err)
	}
	defer s2.close()
	pr2 := newPtyReader(s2)

	_ = pr2.readUntil(t, 2*time.Second, "Password: ")
	s2.handleFill(nil, nil, control{T: "fill", Kind: "generic"})

	time.Sleep(200 * time.Millisecond)
	s2.mu.Lock()
	fills2 := s2.totalFills
	s2.mu.Unlock()
	if fills2 != 0 {
		t.Fatalf("expected 0 fills on generic prompt when jump host present, got %d", fills2)
	}
}

// Rule 5: At most one fill per session per connection (prevents retry lockout).
func TestFillRule5AtMostOneFillPerConnection(t *testing.T) {
	p, v := setupTestStoreAndVault(t)
	secret := "secret-to-try-once"

	store := config.Store{
		Version: config.Version,
		Profiles: []config.Profile{
			{Name: "web", Host: "10.0.0.5", User: "alice", Port: 22, Auth: config.AuthPassword},
		},
	}
	_ = config.Save(p.StoreFile, store)
	setBound(t, v, store, "web", secret)

	script := makeFakeSSH(t, `
printf "alice@10.0.0.5's password: "
read -r line1
printf "FIRST_TRY:%s\n" "$(printf "%s" "$line1" | shasum -a 256 | awk '{print $1}')"

printf "alice@10.0.0.5's password: "
read -r line2
printf "SECOND_TRY:%s\n" "$(printf "%s" "$line2" | shasum -a 256 | awk '{print $1}')"
`)
	s, err := startSession([]string{script}, nil, 80, 24, "web")
	if err != nil {
		t.Fatalf("startSession failed: %v", err)
	}
	defer s.close()
	pr := newPtyReader(s)

	_ = pr.readUntil(t, 2*time.Second, "password: ")

	// 1st fill
	s.handleFill(nil, nil, control{T: "fill", Kind: "account", User: "alice", Host: "10.0.0.5"})

	got := pr.readUntil(t, 3*time.Second, "FIRST_TRY:"+hashSecret(secret), "password: ")
	if !strings.Contains(got, "FIRST_TRY:"+hashSecret(secret)) {
		t.Fatalf("first try not received:\n%s", got)
	}

	// 2nd fill for same connection must be refused
	s.handleFill(nil, nil, control{T: "fill", Kind: "account", User: "alice", Host: "10.0.0.5"})

	time.Sleep(200 * time.Millisecond)
	s.mu.Lock()
	fills := s.totalFills
	s.mu.Unlock()
	if fills != 1 {
		t.Fatalf("expected totalFills to remain 1 after repeat prompt, got %d", fills)
	}
}

// Rule 6: hops are filled in the order ssh authenticates them, each once.
func TestFillFollowsTheHopOrder(t *testing.T) {
	p, v := setupTestStoreAndVault(t)
	store := config.Store{
		Version: config.Version,
		Profiles: []config.Profile{
			{Name: "h1", Host: "10.0.0.1", User: "u1", Port: 22, Auth: config.AuthPassword},
			{Name: "h2", Host: "10.0.0.2", User: "u2", Port: 22, Auth: config.AuthPassword, Jump: "h1"},
			{Name: "h3", Host: "10.0.0.3", User: "u3", Port: 22, Auth: config.AuthPassword, Jump: "h2"},
		},
	}
	_ = config.Save(p.StoreFile, store)
	for _, n := range []string{"h1", "h2", "h3"} {
		setBound(t, v, store, n, "pass-for-"+n)
	}
	s, err := startSession([]string{makeFakeSSH(t, "sleep 10\n")}, nil, 80, 24, "h3")
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()

	fill := func(u, h string) { s.handleFill(nil, nil, control{T: "fill", Kind: "account", User: u, Host: h}) }
	count := func() int { s.mu.Lock(); defer s.mu.Unlock(); return s.totalFills }

	fill("u2", "10.0.0.2") // the middle hop first: allowed (h1 may use a key and never ask)
	fill("u1", "10.0.0.1") // but the outer hop after it is out of order
	if count() != 1 {
		t.Fatalf("an earlier hop was filled after a later one: fills=%d", count())
	}
	fill("u2", "10.0.0.2") // the same hop twice
	if count() != 1 {
		t.Fatalf("a hop was filled twice: fills=%d", count())
	}
	fill("u3", "10.0.0.3")
	if count() != 2 {
		t.Fatalf("expected the target to follow, fills=%d", count())
	}
}

// The prompt text is whatever the remote side prints. A server that prints another saved
// host's prompt must not get that host's password.
func TestFillRefusesAPromptForAHostOutsideTheChain(t *testing.T) {
	p, v := setupTestStoreAndVault(t)
	store := config.Store{
		Version: config.Version,
		Profiles: []config.Profile{
			{Name: "sketchy", Host: "evil.example", User: "bob", Port: 22, Auth: config.AuthPassword},
			{Name: "bank", Host: "bank.internal", User: "bob", Port: 22, Auth: config.AuthPassword},
		},
	}
	_ = config.Save(p.StoreFile, store)
	setBound(t, v, store, "sketchy", "sketchy-password")
	setBound(t, v, store, "bank", "the-bank-password")

	script := makeFakeSSH(t, `
printf "bob@bank.internal's password: "
read -r line
printf "GOT:%s\n" "$(printf "%s" "$line" | shasum -a 256 | awk '{print $1}')"
`)
	s, err := startSession([]string{script}, nil, 80, 24, "sketchy")
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	pr := newPtyReader(s)
	_ = pr.readUntil(t, 2*time.Second, "password: ")

	s.handleFill(nil, nil, control{T: "fill", Kind: "account", User: "bob", Host: "bank.internal"})

	got := pr.readUntil(t, 700*time.Millisecond, "GOT:")
	if strings.Contains(got, "GOT:") {
		t.Fatalf("another connection's password was written to the session: %s", got)
	}
}

// A connection edited outside the app (or restored from a backup) with a new host must not
// receive the password saved for the old one.
func TestFillRefusesWhenTheConnectionChangedSinceTheSave(t *testing.T) {
	p, v := setupTestStoreAndVault(t)
	store := config.Store{
		Version:  config.Version,
		Profiles: []config.Profile{{Name: "web", Host: "10.0.0.5", User: "alice", Port: 22, Auth: config.AuthPassword}},
	}
	_ = config.Save(p.StoreFile, store)
	setBound(t, v, store, "web", "secret-web-99")

	store.Profiles[0].Host = "attacker.example"
	_ = config.Save(p.StoreFile, store)

	script := makeFakeSSH(t, `
printf "alice@attacker.example's password: "
read -r line
printf "GOT:%s\n" "$(printf "%s" "$line" | shasum -a 256 | awk '{print $1}')"
`)
	s, err := startSession([]string{script}, nil, 80, 24, "web")
	if err != nil {
		t.Fatal(err)
	}
	defer s.close()
	pr := newPtyReader(s)
	_ = pr.readUntil(t, 2*time.Second, "password: ")

	s.handleFill(nil, nil, control{T: "fill", Kind: "account", User: "alice", Host: "attacker.example"})

	if got := pr.readUntil(t, 700*time.Millisecond, "GOT:"); strings.Contains(got, "GOT:") {
		t.Fatalf("the saved password was sent to a changed host: %s", got)
	}
	if s.totalFills != 0 {
		t.Fatalf("fills=%d", s.totalFills)
	}
}
