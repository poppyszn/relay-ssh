package main

import (
	"strings"

	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/paths"
)

// migrateVaultKeys moves passwords saved before connections had ids (keyed by name) to the
// connection's id, so renaming or moving a connection never loses its password. The binding
// follows only if it still matches what the connection's details were under the old scheme
// (a jump host was recorded by name); otherwise it is kept and simply no longer matches.
func migrateVaultKeys() {
	p, err := paths.Default()
	if err != nil {
		return
	}
	store, err := config.Load(p.StoreFile)
	if err != nil {
		return
	}
	_ = getVault().Remap(func(key, binding string) (string, string, bool) {
		if strings.HasPrefix(key, "@") {
			return "", "", false
		}
		prof, err := store.Resolve(key)
		if err != nil {
			return "", "", false
		}
		legacy := prof
		if j, ok := store.Find(prof.Jump); ok {
			legacy.Jump = j.Name
		}
		if binding == bindingOf(legacy) {
			binding = bindingOf(prof)
		}
		return prof.ID, binding, true
	})
}
