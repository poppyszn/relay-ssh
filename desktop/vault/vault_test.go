package vault

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

type testFakeTimer struct {
	deadline time.Time
	fn       func()
	clock    *testFakeClock
	stopped  bool
}

func (t *testFakeTimer) Stop() bool {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	if t.stopped {
		return false
	}
	t.stopped = true
	return true
}

func (t *testFakeTimer) Reset(d time.Duration) bool {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	wasActive := !t.stopped
	t.stopped = false
	t.deadline = t.clock.now.Add(d)
	return wasActive
}

type testFakeClock struct {
	mu     sync.Mutex
	now    time.Time
	timers []*testFakeTimer
}

func newTestFakeClock() *testFakeClock {
	return &testFakeClock{now: time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)}
}

func (c *testFakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *testFakeClock) AfterFunc(d time.Duration, f func()) Timer {
	c.mu.Lock()
	defer c.mu.Unlock()
	tmr := &testFakeTimer{
		deadline: c.now.Add(d),
		fn:       f,
		clock:    c,
	}
	c.timers = append(c.timers, tmr)
	return tmr
}

func (c *testFakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	toRun := []func(){}
	for _, t := range c.timers {
		if !t.stopped && !t.deadline.After(c.now) {
			t.stopped = true
			toRun = append(toRun, t.fn)
		}
	}
	c.mu.Unlock()
	for _, fn := range toRun {
		fn()
	}
}

func TestVaultCreateAndStatus(t *testing.T) {
	vpath := filepath.Join(t.TempDir(), "vault.age")
	v := New(vpath)

	st := v.Status()
	if st.Exists || st.Unlocked {
		t.Fatalf("expected non-existent and locked, got %+v", st)
	}

	// Reject too short master password (< 12 characters)
	if err := v.Create("short-pass"); !errors.Is(err, ErrMasterTooShort) {
		t.Fatalf("expected ErrMasterTooShort, got %v", err)
	}

	// Create with valid master
	master := "correct-horse-battery-staple"
	if err := v.Create(master); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	st = v.Status()
	if !st.Exists || !st.Unlocked {
		t.Fatalf("expected exists and unlocked, got %+v", st)
	}

	// Ensure file permissions are 0600
	info, err := os.Stat(vpath)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("expected permissions 0600, got %o", perm)
	}

	// Refuse if file already exists
	if err := v.Create(master); !errors.Is(err, ErrVaultExists) {
		t.Fatalf("expected ErrVaultExists, got %v", err)
	}
}

func TestVaultUnlockWrongPasswordAndCorruptFile(t *testing.T) {
	vpath := filepath.Join(t.TempDir(), "vault.age")
	v := New(vpath)

	// Unlock missing file
	if err := v.Unlock("any-password-that-is-long-enough"); !errors.Is(err, ErrVaultNotFound) {
		t.Fatalf("expected ErrVaultNotFound, got %v", err)
	}

	master := "correct-horse-battery-staple"
	if err := v.Create(master); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	v.Lock()

	// Unlock with wrong password
	if err := v.Unlock("wrong-password-for-unlock-test"); !errors.Is(err, ErrIncorrectPassword) {
		t.Fatalf("expected ErrIncorrectPassword, got %v", err)
	}

	// Unlock with right password
	if err := v.Unlock(master); err != nil {
		t.Fatalf("Unlock failed: %v", err)
	}
	v.Lock()

	// Corrupt the file
	if err := os.WriteFile(vpath, []byte("this is not an age file at all!"), 0o600); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Unlock with corrupt file must return ErrVaultCorrupt, distinct from ErrIncorrectPassword
	if err := v.Unlock(master); !errors.Is(err, ErrVaultCorrupt) {
		t.Fatalf("expected ErrVaultCorrupt, got %v", err)
	}
}

func TestVaultZeroingOnLock(t *testing.T) {
	vpath := filepath.Join(t.TempDir(), "vault.age")
	v := New(vpath)
	master := "correct-horse-battery-staple"
	if err := v.Create(master); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if err := v.Set("srv1", "secret-password-123", ""); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Capture underlying byte slice references before locking
	masterBytes := v.masterBytesForTest()
	secretBytes := v.secretBytesForTest("srv1")

	if len(masterBytes) == 0 || len(secretBytes) == 0 {
		t.Fatalf("expected non-empty byte slices before lock")
	}

	v.Lock()

	// Verify slices were zeroed
	for i, b := range masterBytes {
		if b != 0 {
			t.Fatalf("master byte at index %d was not zeroed: %v", i, b)
		}
	}
	for i, b := range secretBytes {
		if b != 0 {
			t.Fatalf("secret byte at index %d was not zeroed: %v", i, b)
		}
	}

	st := v.Status()
	if !st.Exists || st.Unlocked {
		t.Fatalf("expected exists and locked, got %+v", st)
	}

	// Get after lock must return ErrVaultLocked
	if _, _, err := v.Get("srv1", ""); !errors.Is(err, ErrVaultLocked) {
		t.Fatalf("expected ErrVaultLocked, got %v", err)
	}
}

func TestVaultGetSetForgetRenameHas(t *testing.T) {
	vpath := filepath.Join(t.TempDir(), "vault.age")
	v := New(vpath)
	master := "correct-horse-battery-staple"
	if err := v.Create(master); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Has / Get when empty
	if v.Has("prod") {
		t.Fatalf("expected Has(prod) to be false")
	}
	sec, ok, err := v.Get("prod", "")
	if err != nil || ok || sec != "" {
		t.Fatalf("expected empty Get, got %q, %v, %v", sec, ok, err)
	}

	// Set
	if err := v.Set("prod", "alpha-secret-99", ""); err != nil {
		t.Fatalf("Set failed: %v", err)
	}
	if !v.Has("prod") {
		t.Fatalf("expected Has(prod) to be true")
	}
	sec, ok, err = v.Get("prod", "")
	if err != nil || !ok || sec != "alpha-secret-99" {
		t.Fatalf("expected alpha-secret-99, got %q, %v, %v", sec, ok, err)
	}

	// Rename
	if err := v.Rename("prod", "production-main", "", ""); err != nil {
		t.Fatalf("Rename failed: %v", err)
	}
	if v.Has("prod") {
		t.Fatalf("expected old name prod to be gone")
	}
	if !v.Has("production-main") {
		t.Fatalf("expected new name production-main to exist")
	}
	sec, ok, err = v.Get("production-main", "")
	if err != nil || !ok || sec != "alpha-secret-99" {
		t.Fatalf("expected secret preserved after rename, got %q, %v, %v", sec, ok, err)
	}

	// Persists across Lock and Unlock
	v.Lock()
	v2 := New(vpath)
	if err := v2.Unlock(master); err != nil {
		t.Fatalf("Unlock v2 failed: %v", err)
	}
	sec, ok, err = v2.Get("production-main", "")
	if err != nil || !ok || sec != "alpha-secret-99" {
		t.Fatalf("expected secret after re-unlock, got %q, %v, %v", sec, ok, err)
	}

	// Forget
	if err := v2.Forget("production-main"); err != nil {
		t.Fatalf("Forget failed: %v", err)
	}
	if v2.Has("production-main") {
		t.Fatalf("expected forgotten secret to not exist")
	}
}

func TestVaultReset(t *testing.T) {
	vpath := filepath.Join(t.TempDir(), "vault.age")
	v := New(vpath)
	master := "correct-horse-battery-staple"
	if err := v.Create(master); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Reset without confirmation fails
	if err := v.Reset("no"); err == nil {
		t.Fatalf("expected error without confirmation")
	}

	if err := v.Reset("RESET"); err != nil {
		t.Fatalf("Reset failed: %v", err)
	}

	st := v.Status()
	if st.Exists || st.Unlocked {
		t.Fatalf("expected reset vault to not exist, got %+v", st)
	}
	if _, err := os.Stat(vpath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected file to be removed, got err: %v", err)
	}
}

func TestVaultChangeMaster(t *testing.T) {
	vpath := filepath.Join(t.TempDir(), "vault.age")
	v := New(vpath)
	master1 := "first-master-password-1234"
	master2 := "second-master-password-5678"

	if err := v.Create(master1); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := v.Set("bastion", "bastion-secret-pass", ""); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Wrong old master
	if err := v.ChangeMaster("wrong-old-master-pass", master2); !errors.Is(err, ErrIncorrectPassword) {
		t.Fatalf("expected ErrIncorrectPassword, got %v", err)
	}

	// Too short new master
	if err := v.ChangeMaster(master1, "too-short"); !errors.Is(err, ErrMasterTooShort) {
		t.Fatalf("expected ErrMasterTooShort, got %v", err)
	}

	// Successful change
	if err := v.ChangeMaster(master1, master2); err != nil {
		t.Fatalf("ChangeMaster failed: %v", err)
	}

	// Re-open with master2
	v.Lock()
	v2 := New(vpath)
	if err := v2.Unlock(master1); !errors.Is(err, ErrIncorrectPassword) {
		t.Fatalf("old master must no longer work, got %v", err)
	}
	if err := v2.Unlock(master2); err != nil {
		t.Fatalf("new master failed to unlock: %v", err)
	}
	sec, ok, err := v2.Get("bastion", "")
	if err != nil || !ok || sec != "bastion-secret-pass" {
		t.Fatalf("expected secret intact after master change, got %q, %v, %v", sec, ok, err)
	}
}

func TestVaultIdleAutoLock(t *testing.T) {
	vpath := filepath.Join(t.TempDir(), "vault.age")
	clock := newTestFakeClock()
	v := New(vpath, WithClock(clock), WithIdleTimeout(15*time.Minute))
	master := "correct-horse-battery-staple"

	if err := v.Create(master); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if err := v.Set("dev", "dev-secret-42", ""); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Advance 10 minutes: still unlocked
	clock.Advance(10 * time.Minute)
	if !v.Status().Unlocked {
		t.Fatalf("expected unlocked after 10m")
	}

	// Operation resets the timer
	if _, _, err := v.Get("dev", ""); err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Advance 10 minutes (20m total from start, but 10m from last op): still unlocked
	clock.Advance(10 * time.Minute)
	if !v.Status().Unlocked {
		t.Fatalf("expected unlocked because timer was reset by Get")
	}

	// Advance another 6 minutes (16m from last op): should auto-lock!
	clock.Advance(6 * time.Minute)
	if v.Status().Unlocked {
		t.Fatalf("expected vault to auto-lock after 15m idle")
	}

	if _, _, err := v.Get("dev", ""); !errors.Is(err, ErrVaultLocked) {
		t.Fatalf("expected ErrVaultLocked, got %v", err)
	}
}

func TestVaultAtomicWriteSafety(t *testing.T) {
	vpath := filepath.Join(t.TempDir(), "vault.age")
	v := New(vpath)
	master := "correct-horse-battery-staple"
	if err := v.Create(master); err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	_ = v.Set("node1", "pass1", "")

	// Verify original file exists and is readable
	origBytes, err := os.ReadFile(vpath)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	// Ensure no partial or tmp file remains in the directory
	entries, err := os.ReadDir(filepath.Dir(vpath))
	if err != nil {
		t.Fatalf("ReadDir failed: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "vault.age" {
		t.Fatalf("expected only vault.age in dir, found %v", entries)
	}

	// Check content non-empty
	if len(origBytes) == 0 || bytes.Contains(origBytes, []byte("pass1")) {
		t.Fatalf("plaintext secret leaked in vault.age file!")
	}
}
