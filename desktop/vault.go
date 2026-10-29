package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"filippo.io/age"
	"github.com/poppyszn/relay-ssh/internal/fsutil"
)

type Timer interface {
	Stop() bool
	Reset(d time.Duration) bool
}

type Clock interface {
	Now() time.Time
	AfterFunc(d time.Duration, f func()) Timer
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }
func (realClock) AfterFunc(d time.Duration, f func()) Timer {
	return time.AfterFunc(d, f)
}

type VaultOption func(*Vault)

func WithVaultClock(c Clock) VaultOption {
	return func(v *Vault) { v.clock = c }
}

func WithIdleTimeout(d time.Duration) VaultOption {
	return func(v *Vault) { v.idleTimeout = d }
}

func WithOnLock(fn func()) VaultOption {
	return func(v *Vault) { v.onLock = fn }
}

type vaultPayload struct {
	Version int               `json:"version"`
	Secrets map[string]string `json:"secrets"`
}

// Vault manages an encrypted store of SSH connection passwords in vault.age.
// All secrets and the master password are held as []byte and zeroed on lock.
type Vault struct {
	mu          sync.Mutex
	path        string
	master      []byte
	secrets     map[string][]byte
	idleTimeout time.Duration
	timer       Timer
	clock       Clock
	onLock      func()
}

// NewVault creates a new Vault pointing to path.
func NewVault(path string, opts ...VaultOption) *Vault {
	v := &Vault{
		path:        path,
		idleTimeout: 15 * time.Minute,
		clock:       realClock{},
	}
	for _, opt := range opts {
		opt(v)
	}
	return v
}

func zeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func (v *Vault) masterBytesForTest() []byte {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.master
}

func (v *Vault) secretBytesForTest(conn string) []byte {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.secrets[conn]
}

func (v *Vault) Status() VaultStatus {
	v.mu.Lock()
	defer v.mu.Unlock()

	_, err := os.Stat(v.path)
	exists := err == nil
	unlocked := v.master != nil
	return VaultStatus{
		Exists:   exists,
		Unlocked: unlocked,
	}
}

func (v *Vault) Create(master string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if len(master) < 12 {
		return ErrMasterTooShort
	}
	if _, err := os.Stat(v.path); err == nil {
		return ErrVaultExists
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	v.master = []byte(master)
	v.secrets = make(map[string][]byte)
	if err := v.saveLocked(); err != nil {
		zeroBytes(v.master)
		v.master = nil
		v.secrets = nil
		return err
	}
	v.resetIdleTimerLocked()
	return nil
}

func (v *Vault) Unlock(master string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	data, err := os.ReadFile(v.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrVaultNotFound
		}
		return err
	}

	payload, err := v.decryptPayload(master, data)
	if err != nil {
		return err
	}

	if v.master != nil {
		zeroBytes(v.master)
	}
	for _, s := range v.secrets {
		zeroBytes(s)
	}

	v.master = []byte(master)
	v.secrets = make(map[string][]byte, len(payload.Secrets))
	for k, val := range payload.Secrets {
		v.secrets[k] = []byte(val)
	}
	v.resetIdleTimerLocked()
	return nil
}

func (v *Vault) Lock() {
	v.mu.Lock()
	if v.master == nil && v.secrets == nil {
		v.mu.Unlock()
		return
	}

	if v.master != nil {
		zeroBytes(v.master)
		v.master = nil
	}
	for _, s := range v.secrets {
		zeroBytes(s)
	}
	v.secrets = nil

	if v.timer != nil {
		v.timer.Stop()
		v.timer = nil
	}
	onLock := v.onLock
	v.mu.Unlock()

	if onLock != nil {
		onLock()
	}
}

func (v *Vault) ChangeMaster(oldMaster, newMaster string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if len(newMaster) < 12 {
		return ErrMasterTooShort
	}

	if v.master != nil {
		if string(v.master) != oldMaster {
			return ErrIncorrectPassword
		}
	} else {
		data, err := os.ReadFile(v.path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return ErrVaultNotFound
			}
			return err
		}
		payload, err := v.decryptPayload(oldMaster, data)
		if err != nil {
			return err
		}
		v.secrets = make(map[string][]byte, len(payload.Secrets))
		for k, val := range payload.Secrets {
			v.secrets[k] = []byte(val)
		}
	}

	secretsStr := make(map[string]string, len(v.secrets))
	for k, vBytes := range v.secrets {
		secretsStr[k] = string(vBytes)
	}
	payload := vaultPayload{
		Version: 1,
		Secrets: secretsStr,
	}
	enc, err := v.encryptPayload(newMaster, payload)
	if err != nil {
		return err
	}

	if err := fsutil.WriteFile(v.path, enc, 0o600); err != nil {
		return err
	}

	if v.master != nil {
		zeroBytes(v.master)
	}
	v.master = []byte(newMaster)
	v.resetIdleTimerLocked()
	return nil
}

func (v *Vault) Reset(confirm string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if confirm != "RESET" {
		return errors.New("reset confirmation required (confirm must be 'RESET')")
	}

	if v.master != nil {
		zeroBytes(v.master)
		v.master = nil
	}
	for _, s := range v.secrets {
		zeroBytes(s)
	}
	v.secrets = nil

	if v.timer != nil {
		v.timer.Stop()
		v.timer = nil
	}

	if err := os.Remove(v.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (v *Vault) SetIdleTimeout(minutes int) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if minutes <= 0 {
		v.idleTimeout = 0
		if v.timer != nil {
			v.timer.Stop()
			v.timer = nil
		}
		return
	}
	v.idleTimeout = time.Duration(minutes) * time.Minute
	v.resetIdleTimerLocked()
}

func (v *Vault) Get(connection string) (string, bool, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.master == nil {
		return "", false, ErrVaultLocked
	}
	v.resetIdleTimerLocked()

	sec, ok := v.secrets[connection]
	if !ok {
		return "", false, nil
	}
	return string(sec), true, nil
}

func (v *Vault) Set(connection, secret string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.master == nil {
		return ErrVaultLocked
	}
	conn := strings.TrimSpace(connection)
	if conn == "" {
		return errors.New("connection name cannot be empty")
	}

	if old, ok := v.secrets[conn]; ok {
		zeroBytes(old)
	}
	v.secrets[conn] = []byte(secret)
	if err := v.saveLocked(); err != nil {
		return err
	}
	v.resetIdleTimerLocked()
	return nil
}

func (v *Vault) Forget(connection string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.master == nil {
		return ErrVaultLocked
	}
	conn := strings.TrimSpace(connection)
	if old, ok := v.secrets[conn]; ok {
		zeroBytes(old)
		delete(v.secrets, conn)
		if err := v.saveLocked(); err != nil {
			return err
		}
	}
	v.resetIdleTimerLocked()
	return nil
}

func (v *Vault) Rename(oldName, newName string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.master == nil {
		return ErrVaultLocked
	}
	oldN := strings.TrimSpace(oldName)
	newN := strings.TrimSpace(newName)
	if newN == "" {
		return errors.New("new connection name cannot be empty")
	}

	if sec, ok := v.secrets[oldN]; ok {
		v.secrets[newN] = sec
		delete(v.secrets, oldN)
		if err := v.saveLocked(); err != nil {
			return err
		}
	}
	v.resetIdleTimerLocked()
	return nil
}

func (v *Vault) Has(connection string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.master == nil {
		return false
	}
	v.resetIdleTimerLocked()
	_, ok := v.secrets[strings.TrimSpace(connection)]
	return ok
}

func (v *Vault) Reconcile(validProfiles []string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.master == nil {
		return ErrVaultLocked
	}
	valid := make(map[string]bool, len(validProfiles))
	for _, name := range validProfiles {
		valid[name] = true
	}

	changed := false
	for name, sec := range v.secrets {
		if !valid[name] {
			zeroBytes(sec)
			delete(v.secrets, name)
			changed = true
		}
	}
	if changed {
		return v.saveLocked()
	}
	return nil
}

func (v *Vault) saveLocked() error {
	if v.master == nil {
		return ErrVaultLocked
	}
	secretsStr := make(map[string]string, len(v.secrets))
	for k, vBytes := range v.secrets {
		secretsStr[k] = string(vBytes)
	}
	payload := vaultPayload{
		Version: 1,
		Secrets: secretsStr,
	}
	enc, err := v.encryptPayload(string(v.master), payload)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(v.path), 0o700); err != nil {
		return err
	}
	return fsutil.WriteFile(v.path, enc, 0o600)
}

func (v *Vault) encryptPayload(master string, payload vaultPayload) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	recipient, err := age.NewScryptRecipient(master)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, recipient)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write(data); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (v *Vault) decryptPayload(master string, encData []byte) (*vaultPayload, error) {
	identity, err := age.NewScryptIdentity(master)
	if err != nil {
		return nil, ErrIncorrectPassword
	}
	r, err := age.Decrypt(bytes.NewReader(encData), identity)
	if err != nil {
		var noIdent *age.NoIdentityMatchError
		if errors.As(err, &noIdent) {
			return nil, ErrIncorrectPassword
		}
		return nil, ErrVaultCorrupt
	}
	decrypted, err := io.ReadAll(r)
	if err != nil {
		var noIdent *age.NoIdentityMatchError
		if errors.As(err, &noIdent) {
			return nil, ErrIncorrectPassword
		}
		return nil, ErrVaultCorrupt
	}
	var payload vaultPayload
	if err := json.Unmarshal(decrypted, &payload); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrVaultCorrupt, err)
	}
	if payload.Version != 1 || payload.Secrets == nil {
		return nil, fmt.Errorf("%w: invalid version or payload", ErrVaultCorrupt)
	}
	return &payload, nil
}

func (v *Vault) resetIdleTimerLocked() {
	if v.idleTimeout <= 0 || v.master == nil {
		if v.timer != nil {
			v.timer.Stop()
			v.timer = nil
		}
		return
	}
	if v.timer != nil {
		v.timer.Reset(v.idleTimeout)
	} else {
		v.timer = v.clock.AfterFunc(v.idleTimeout, func() {
			v.Lock()
		})
	}
}
