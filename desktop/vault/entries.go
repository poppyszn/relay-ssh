package vault

import (
	"crypto/subtle"
	"errors"
	"os"
	"strings"
)

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
	if !ValidSecret(secret) {
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

// Reveal returns a saved password for the person to read. The master password is checked against
// the file every time, even while the vault is unlocked: the check costs a full key derivation, so
// it cannot be guessed quickly, and an open vault is not enough to read a password back. The vault's
// lock state is left as it was, and the binding is not consulted (this is the owner looking at it).
func (v *Vault) Reveal(connection, master string) (string, error) {
	v.mu.Lock()
	path := v.path
	v.mu.Unlock()
	if path == "" {
		return "", ErrNoVaultPath
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", ErrVaultNotFound
		}
		return "", err
	}
	payload, err := v.decryptPayload(master, data)
	if err != nil {
		return "", err
	}
	e, ok := payload.Secrets[strings.TrimSpace(connection)]
	if !ok {
		return "", ErrNoSecret
	}
	return e.Password, nil
}
