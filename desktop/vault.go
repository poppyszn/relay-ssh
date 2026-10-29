package main

import (
	"bytes"
	"crypto/subtle"
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

// payloadEntry is one saved password and the connection details it was saved for.
type payloadEntry struct {
	Password string `json:"password"`
	Binding  string `json:"binding"`
}

// vaultPayload is the encrypted file's content. Version 1 stored a bare password per
// connection (Secrets as map[string]string); version 2 adds the binding.
type vaultPayload struct {
	Version int                     `json:"version"`
	Secrets map[string]payloadEntry `json:"secrets"`
}

type secretEntry struct {
	password []byte
	binding  string
}

func (e *secretEntry) wipe() { zeroBytes(e.password) }

// Vault manages an encrypted store of SSH connection passwords in vault.age.
// Every password is kept with a binding (see bindingOf): it is only released for the
// host, user and port it was saved for. Secrets and the master password are held as
// []byte and zeroed on lock; copies made for encryption and the final PTY write are
// Go strings that cannot be wiped, so this limits exposure rather than eliminating it.
type Vault struct {
	mu          sync.Mutex
	path        string
	master      []byte
	secrets     map[string]*secretEntry
	idleTimeout time.Duration
	timer       Timer
	gen         int // bumped on every restart of the idle countdown; a stale timer does nothing
	clock       Clock
	onLock      func()
}

// NewVault creates a new Vault pointing to path. An empty path yields a vault that refuses
// to store anything (it never falls back to the working directory).
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
	if e := v.secrets[conn]; e != nil {
		return e.password
	}
	return nil
}

func (v *Vault) secretCountForTest() int {
	v.mu.Lock()
	defer v.mu.Unlock()
	return len(v.secrets)
}

func (v *Vault) Status() VaultStatus {
	v.mu.Lock()
	defer v.mu.Unlock()

	_, err := os.Stat(v.path)
	return VaultStatus{Exists: v.path != "" && err == nil, Unlocked: v.master != nil}
}

func (v *Vault) Create(master string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.path == "" {
		return ErrNoVaultPath
	}
	if len(master) < 12 {
		return ErrMasterTooShort
	}
	if _, err := os.Stat(v.path); err == nil {
		return ErrVaultExists
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	v.master = []byte(master)
	v.secrets = make(map[string]*secretEntry)
	if err := v.saveLocked(); err != nil {
		zeroBytes(v.master)
		v.master = nil
		v.secrets = nil
		return err
	}
	v.restartIdleTimerLocked()
	return nil
}

func (v *Vault) Unlock(master string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.path == "" {
		return ErrNoVaultPath
	}
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

	v.wipeLocked()
	v.master = []byte(master)
	v.secrets = entriesFromPayload(payload)
	v.restartIdleTimerLocked()
	return nil
}

func entriesFromPayload(p *vaultPayload) map[string]*secretEntry {
	m := make(map[string]*secretEntry, len(p.Secrets))
	for k, e := range p.Secrets {
		m[k] = &secretEntry{password: []byte(e.Password), binding: e.Binding}
	}
	return m
}

// wipeLocked zeroes and drops the master password and every secret.
func (v *Vault) wipeLocked() {
	zeroBytes(v.master)
	v.master = nil
	for _, e := range v.secrets {
		e.wipe()
	}
	v.secrets = nil
	v.gen++
	if v.timer != nil {
		v.timer.Stop()
		v.timer = nil
	}
}

func (v *Vault) Lock() {
	v.mu.Lock()
	if v.master == nil && v.secrets == nil {
		v.mu.Unlock()
		return
	}
	v.wipeLocked()
	onLock := v.onLock
	v.mu.Unlock()

	if onLock != nil {
		onLock()
	}
}

// ChangeMaster re-encrypts the vault under a new master password. The old one must be right
// whether or not the vault is currently unlocked.
func (v *Vault) ChangeMaster(oldMaster, newMaster string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.path == "" {
		return ErrNoVaultPath
	}
	if len(newMaster) < 12 {
		return ErrMasterTooShort
	}

	secrets := v.secrets
	if v.master != nil {
		if subtle.ConstantTimeCompare(v.master, []byte(oldMaster)) != 1 {
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
		secrets = entriesFromPayload(payload)
		// Whatever happens next, the plaintext decrypted here is not kept unless we end up unlocked.
		defer func() {
			if v.master == nil {
				for _, e := range secrets {
					e.wipe()
				}
			}
		}()
	}

	enc, err := v.encryptPayload(newMaster, payloadOf(secrets))
	if err != nil {
		return err
	}
	if err := fsutil.WriteFile(v.path, enc, 0o600); err != nil {
		return err
	}

	zeroBytes(v.master)
	v.master = []byte(newMaster)
	v.secrets = secrets
	v.restartIdleTimerLocked()
	return nil
}

func (v *Vault) Reset(confirm string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if confirm != "RESET" {
		return errors.New("reset confirmation required (confirm must be 'RESET')")
	}
	v.wipeLocked()
	if v.path == "" {
		return nil
	}
	if err := os.Remove(v.path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func (v *Vault) SetIdleTimeout(minutes int) {
	v.mu.Lock()
	defer v.mu.Unlock()

	d := time.Duration(minutes) * time.Minute
	if minutes <= 0 {
		d = 0
	}
	if d == v.idleTimeout {
		return // repeating the current setting is not activity
	}
	v.idleTimeout = d
	v.restartIdleTimerLocked()
}

// Get releases the password saved for a connection, but only for the binding it was saved
// with: if the host, user, port or jump host changed since, it refuses with ErrBindingChanged.
// Using the vault restarts the idle countdown.
func (v *Vault) Get(connection, binding string) (string, bool, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.master == nil {
		return "", false, ErrVaultLocked
	}
	v.restartIdleTimerLocked()

	e, ok := v.secrets[connection]
	if !ok {
		return "", false, nil
	}
	if subtle.ConstantTimeCompare([]byte(e.binding), []byte(binding)) != 1 {
		return "", false, ErrBindingChanged
	}
	return string(e.password), true, nil
}

// validSecret rejects empty passwords and control characters: a stray ^C or ^D written to the
// terminal would interrupt ssh instead of answering it, and a line break would submit early.
func validSecret(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	return true
}

func (v *Vault) Set(connection, secret, binding string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.master == nil {
		return ErrVaultLocked
	}
	conn := strings.TrimSpace(connection)
	if conn == "" {
		return errors.New("connection name cannot be empty")
	}
	if !validSecret(secret) {
		return ErrBadSecret
	}

	old := v.secrets[conn]
	v.secrets[conn] = &secretEntry{password: []byte(secret), binding: binding}
	if err := v.saveLocked(); err != nil {
		v.restoreLocked(conn, old)
		return err
	}
	if old != nil {
		old.wipe()
	}
	v.restartIdleTimerLocked()
	return nil
}

// restoreLocked undoes an in-memory change whose save failed, dropping the attempted entry.
func (v *Vault) restoreLocked(conn string, old *secretEntry) {
	if cur := v.secrets[conn]; cur != nil && cur != old {
		cur.wipe()
	}
	if old == nil {
		delete(v.secrets, conn)
	} else {
		v.secrets[conn] = old
	}
}

func (v *Vault) Forget(connection string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.master == nil {
		return ErrVaultLocked
	}
	conn := strings.TrimSpace(connection)
	if old, ok := v.secrets[conn]; ok {
		delete(v.secrets, conn)
		if err := v.saveLocked(); err != nil {
			v.secrets[conn] = old
			return err
		}
		old.wipe()
	}
	v.restartIdleTimerLocked()
	return nil
}

// Rename moves a saved password to a new connection name. The binding follows the connection's
// new details only if the stored one still matched its old details (the connection was intact);
// otherwise the original binding is kept, so editing a tampered connection never re-approves it.
func (v *Vault) Rename(oldName, newName, oldBinding, newBinding string) error {
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

	e, ok := v.secrets[oldN]
	if !ok {
		v.restartIdleTimerLocked()
		return nil
	}
	moved := &secretEntry{password: e.password, binding: e.binding}
	if subtle.ConstantTimeCompare([]byte(e.binding), []byte(oldBinding)) == 1 {
		moved.binding = newBinding
	}
	displaced := v.secrets[newN]
	delete(v.secrets, oldN)
	v.secrets[newN] = moved
	if err := v.saveLocked(); err != nil {
		if oldN != newN {
			delete(v.secrets, newN)
			if displaced != nil {
				v.secrets[newN] = displaced
			}
		}
		v.secrets[oldN] = e
		return err
	}
	if displaced != nil && oldN != newN {
		displaced.wipe()
	}
	v.restartIdleTimerLocked()
	return nil
}

// Has reports whether a password is saved. It does not count as use, so asking cannot keep
// the vault unlocked.
func (v *Vault) Has(connection string) bool {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.master == nil {
		return false
	}
	_, ok := v.secrets[strings.TrimSpace(connection)]
	return ok
}

func payloadOf(secrets map[string]*secretEntry) vaultPayload {
	out := make(map[string]payloadEntry, len(secrets))
	for k, e := range secrets {
		out[k] = payloadEntry{Password: string(e.password), Binding: e.binding}
	}
	return vaultPayload{Version: 2, Secrets: out}
}

func (v *Vault) saveLocked() error {
	if v.master == nil {
		return ErrVaultLocked
	}
	if v.path == "" {
		return ErrNoVaultPath
	}
	enc, err := v.encryptPayload(string(v.master), payloadOf(v.secrets))
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
	return v.encryptRawForTest(master, data)
}

func (v *Vault) encryptRawForTest(master string, data []byte) ([]byte, error) {
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
	defer zeroBytes(decrypted)

	var head struct {
		Version int             `json:"version"`
		Secrets json.RawMessage `json:"secrets"`
	}
	if err := json.Unmarshal(decrypted, &head); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrVaultCorrupt, err)
	}
	payload := &vaultPayload{Version: 2, Secrets: map[string]payloadEntry{}}
	switch head.Version {
	case 1:
		var old map[string]string
		if err := json.Unmarshal(head.Secrets, &old); err != nil || old == nil {
			return nil, fmt.Errorf("%w: invalid payload", ErrVaultCorrupt)
		}
		for k, pw := range old {
			payload.Secrets[k] = payloadEntry{Password: pw}
		}
	case 2:
		if err := json.Unmarshal(head.Secrets, &payload.Secrets); err != nil || payload.Secrets == nil {
			return nil, fmt.Errorf("%w: invalid payload", ErrVaultCorrupt)
		}
	default:
		return nil, fmt.Errorf("%w: unknown version %d", ErrVaultCorrupt, head.Version)
	}
	return payload, nil
}

// restartIdleTimerLocked restarts the idle countdown (or stops it when there is none to run).
// Each restart bumps gen, so a timer that already fired but lost the race for the mutex
// finds a newer generation and leaves the vault alone.
func (v *Vault) restartIdleTimerLocked() {
	if v.timer != nil {
		v.timer.Stop()
		v.timer = nil
	}
	v.gen++
	if v.idleTimeout <= 0 || v.master == nil {
		return
	}
	gen := v.gen
	v.timer = v.clock.AfterFunc(v.idleTimeout, func() {
		v.mu.Lock()
		stale := v.gen != gen
		v.mu.Unlock()
		if !stale {
			v.Lock()
		}
	})
}
