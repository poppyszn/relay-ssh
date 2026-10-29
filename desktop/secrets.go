package main

import (
	"errors"
	"path/filepath"

	"github.com/poppyszn/relay-ssh/internal/paths"
)

var (
	ErrVaultLocked       = errors.New("vault is locked")
	ErrVaultExists       = errors.New("vault already exists")
	ErrVaultNotFound     = errors.New("vault does not exist")
	ErrIncorrectPassword = errors.New("incorrect master password")
	ErrVaultCorrupt      = errors.New("vault file is corrupt")
	ErrMasterTooShort    = errors.New("master password must be at least 12 characters")
)

// VaultStatus describes the presence and unlock state of the vault.
type VaultStatus struct {
	Exists   bool `json:"exists"`
	Unlocked bool `json:"unlocked"`
}

// SecretStore is the storage interface for remembering connection credentials.
// Get is strictly Go-internal and is never exposed over Wails bindings.
type SecretStore interface {
	Status() VaultStatus
	Get(connection string) (string, bool, error)
	Set(connection, secret string) error
	Forget(connection string) error
	Rename(oldName, newName string) error
	Has(connection string) bool
}

func defaultVaultPath() (string, error) {
	p, err := paths.Default()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(p.StoreFile), "vault.age"), nil
}
