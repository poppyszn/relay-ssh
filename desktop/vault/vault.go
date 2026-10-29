package vault

import (
	"crypto/subtle"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/poppyszn/relay-ssh/internal/fsutil"
)

type secretEntry struct {
	password []byte
	binding  string
}

func (e *secretEntry) wipe() { Wipe(e.password) }

// Wipe zeroes b in place.
func Wipe(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

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

// New creates a new Vault pointing to path. An empty path yields a vault that refuses
// to store anything (it never falls back to the working directory).
func New(path string, opts ...Option) *Vault {
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

func (v *Vault) Status() Status {
	v.mu.Lock()
	defer v.mu.Unlock()

	_, err := os.Stat(v.path)
	return Status{Exists: v.path != "" && err == nil, Unlocked: v.master != nil}
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
		Wipe(v.master)
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

// wipeLocked zeroes and drops the master password and every secret.
func (v *Vault) wipeLocked() {
	Wipe(v.master)
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

	Wipe(v.master)
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
