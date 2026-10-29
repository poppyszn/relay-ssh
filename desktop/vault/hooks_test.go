package vault

// Test-only views of the unexported state.

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
