package main

import (
	"sync"

	"github.com/wailsapp/wails/v3/pkg/application"
)

var (
	appVaultMu sync.RWMutex
	appVault   *Vault
)

func getVault() *Vault {
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
		appVault = NewVault(p)
	}
	return appVault
}

func setVaultForTest(v *Vault) func() {
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

// VaultService exposes secret vault operations to the frontend. No method returns a secret,
// and none stores one: passwords go in through ProfilesService.SaveWithPassword, which binds
// them to the connection being saved.
type VaultService struct{}

func (VaultService) Status() VaultStatus {
	return getVault().Status()
}

func (VaultService) Create(master string) error {
	return getVault().Create(master)
}

func (VaultService) Unlock(master string) error {
	return getVault().Unlock(master)
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
