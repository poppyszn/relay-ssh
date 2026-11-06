package main

import (
	"github.com/poppyszn/relay-ssh/desktop/vault"
	"github.com/poppyszn/relay-ssh/internal/paths"
	"path/filepath"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

var (
	appVaultMu sync.RWMutex
	appVault   *vault.Vault
)

func getVault() *vault.Vault {
	appVaultMu.RLock()
	v := appVault
	appVaultMu.RUnlock()
	if v != nil {
		return v
	}

	appVaultMu.Lock()
	defer appVaultMu.Unlock()
	if appVault == nil {
		p, err := defaultVaultPath()
		if err != nil {
			p = "vault.age"
		}
		appVault = vault.New(p)
	}
	return appVault
}

func setVaultForTest(v *vault.Vault) func() {
	appVaultMu.Lock()
	old := appVault
	appVault = v
	appVaultMu.Unlock()
	return func() {
		appVaultMu.Lock()
		appVault = old
		appVaultMu.Unlock()
	}
}

// VaultStatus describes the presence and unlock state of the vault.
type VaultStatus struct {
	Exists   bool `json:"exists"`
	Unlocked bool `json:"unlocked"`
}

func defaultVaultPath() (string, error) {
	p, err := paths.Default()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(p.StoreFile), "vault.age"), nil
}

// VaultService exposes secret vault operations to the frontend. Passwords go in only through
// ProfilesService.SaveWithPassword, which binds them to the connection being saved, and come back
// out only through RevealSecret, which needs the master password every time. Nothing else here
// returns a secret.
type VaultService struct{}

func (VaultService) Status() VaultStatus {
	st := getVault().Status()
	return VaultStatus{Exists: st.Exists, Unlocked: st.Unlocked}
}

func (VaultService) Create(master string) error {
	return getVault().Create(master)
}

func (VaultService) Unlock(master string) error {
	if err := getVault().Unlock(master); err != nil {
		return err
	}
	migrateVaultKeys()
	return nil
}

func (VaultService) Lock() {
	getVault().Lock()
}

func (VaultService) ChangeMaster(old, new string) error {
	return getVault().ChangeMaster(old, new)
}

func (VaultService) Reset(confirm string) error {
	return getVault().Reset(confirm)
}

func (VaultService) SetIdleTimeout(minutes int) {
	getVault().SetIdleTimeout(minutes)
}

func (VaultService) HasSecret(connection string) bool {
	return getVault().Has(connection)
}

func (VaultService) ForgetSecret(connection string) error {
	return getVault().Forget(connection)
}

// announceVaultLocked tells the window the vault locked itself (idle timeout or quit),
// so the status bar does not keep showing it as unlocked.
func announceVaultLocked() {
	if a := application.Get(); a != nil {
		a.Event.Emit("vault", "locked")
	}
}

// RevealSecret returns the password saved for a connection so the person can read it. The master
// password must be supplied for every call, whether or not the vault is unlocked.
func (VaultService) RevealSecret(connection, master string) (string, error) {
	return getVault().Reveal(connection, master)
}
