package vault

import "errors"

var (
	ErrVaultLocked       = errors.New("vault is locked")
	ErrVaultExists       = errors.New("vault already exists")
	ErrVaultNotFound     = errors.New("vault does not exist")
	ErrIncorrectPassword = errors.New("incorrect master password")
	ErrVaultCorrupt      = errors.New("vault file is corrupt")
	ErrMasterTooShort    = errors.New("master password must be at least 12 characters")
	ErrNoVaultPath       = errors.New("the vault has nowhere to live: Relay could not find your home folder")
	ErrBadSecret         = errors.New("a saved password cannot be empty or contain line breaks or control characters")
	ErrNoSecret          = errors.New("no password is saved for this connection")
	ErrBindingChanged    = errors.New("this connection's host, user, port or jump host changed since its password was saved")
)

// Status describes the presence and unlock state of the vault.
type Status struct {
	Exists   bool `json:"exists"`
	Unlocked bool `json:"unlocked"`
}

// ValidSecret reports whether s may be saved as a password: not empty, and free of line
// breaks and control characters (a stray ^C or ^D typed into the terminal would interrupt ssh
// instead of answering it, and a line break would submit early).
func ValidSecret(s string) bool {
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
