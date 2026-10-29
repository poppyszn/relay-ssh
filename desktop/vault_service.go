package main

import (
	"sync"

	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/paths"
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

// VaultService exposes secret vault operations to the frontend.
// No method on this service ever returns a secret.
type VaultService struct{}

func (VaultService) Status() VaultStatus {
	return getVault().Status()
}

func (VaultService) Create(master string) error {
	return getVault().Create(master)
}

func (VaultService) Unlock(master string) error {
	v := getVault()
	if err := v.Unlock(master); err != nil {
		return err
	}
	// Reconcile stale secrets against existing password-auth profiles
	if p, err := paths.Default(); err == nil {
		if s, err := config.Load(p.StoreFile); err == nil {
			var valid []string
			for _, prof := range s.Profiles {
				if prof.Auth == config.AuthPassword {
					valid = append(valid, prof.Name)
				}
			}
			_ = v.Reconcile(valid)
		}
	}
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

func (VaultService) SetSecret(connection, password string) error {
	return getVault().Set(connection, password)
}

func (VaultService) ForgetSecret(connection string) error {
	return getVault().Forget(connection)
}
