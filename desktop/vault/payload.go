package vault

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"filippo.io/age"
)

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

func entriesFromPayload(p *vaultPayload) map[string]*secretEntry {
	m := make(map[string]*secretEntry, len(p.Secrets))
	for k, e := range p.Secrets {
		m[k] = &secretEntry{password: []byte(e.Password), binding: e.Binding}
	}
	return m
}

func payloadOf(secrets map[string]*secretEntry) vaultPayload {
	out := make(map[string]payloadEntry, len(secrets))
	for k, e := range secrets {
		out[k] = payloadEntry{Password: string(e.password), Binding: e.binding}
	}
	return vaultPayload{Version: 2, Secrets: out}
}

func (v *Vault) encryptPayload(master string, payload vaultPayload) ([]byte, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return v.encrypt(master, data)
}

func (v *Vault) encrypt(master string, data []byte) ([]byte, error) {
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
	defer Wipe(decrypted)

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
