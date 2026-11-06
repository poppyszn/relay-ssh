package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// grantLife is how long a chosen path stays usable. Long enough for a restore (choose, then type the passphrase).
const grantLife = 15 * time.Minute

// pathGrants records the local paths the page may hand to methods that read or write files. The page can call
// every bound method, so without this a compromised page could back up to, restore from, adopt or transfer any
// path it likes. A path is granted when the user chooses it in a native dialog, or when the app itself lists it
// for the user (the keys in ~/.ssh and the vault). One grant is for that exact file: not its folder, not files
// inside it.
type pathGrants struct {
	mu  sync.Mutex
	m   map[string]time.Time
	now func() time.Time
}

var grants = &pathGrants{m: map[string]time.Time{}, now: time.Now}

func normalize(path string) string {
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}

func (g *pathGrants) allow(path string) {
	if path == "" {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.m[normalize(path)] = g.now().Add(grantLife)
}

// require refuses a path nobody chose. The benchmark and smoke harness (RELAY_BENCH, set only by whoever launches
// the app) drives these methods with paths of its own and is exempt.
func (g *pathGrants) require(path string) error {
	if os.Getenv("RELAY_BENCH") != "" {
		return nil
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if path != "" {
		if until, ok := g.m[normalize(path)]; ok && g.now().Before(until) {
			return nil
		}
	}
	return fmt.Errorf("Relay only uses files you choose: pick %q in a file dialog first", path)
}

// grantPicked is what the native pickers return through: a path the user actually chose is granted; a cancelled
// or failed dialog grants nothing.
func grantPicked(path string, err error) (string, error) {
	if err != nil || path == "" {
		return path, err
	}
	grants.allow(path)
	return path, nil
}
