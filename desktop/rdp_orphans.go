package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/poppyszn/relay-ssh/internal/fsutil"
	"github.com/poppyszn/relay-ssh/internal/paths"
)

// tunnelLedger records the ssh processes Relay started to tunnel to RDP hosts. If Relay is force-quit or crashes
// they keep running, so the next start ends the survivors that are still exactly that tunnel.
type tunnelLedger struct {
	path string

	mu      sync.Mutex
	entries map[int]string // pid -> the -L forward spec

	// procCommand reports the command line of a running process; kill ends one. Both are replaced in tests.
	procCommand func(pid int) (string, bool)
	kill        func(pid int) error
}

func newTunnelLedger(path string) *tunnelLedger {
	l := &tunnelLedger{path: path, entries: map[int]string{}, procCommand: processCommand, kill: killProcess}
	for _, e := range l.read() {
		l.entries[e.PID] = e.Spec
	}
	return l
}

type tunnelRecord struct {
	PID  int    `json:"pid"`
	Spec string `json:"spec"`
}

// read returns what is on disk; a missing or damaged file is simply empty.
func (l *tunnelLedger) read() []tunnelRecord {
	data, err := os.ReadFile(l.path)
	if err != nil {
		return nil
	}
	var recs []tunnelRecord
	if json.Unmarshal(data, &recs) != nil {
		return nil
	}
	return recs
}

func (l *tunnelLedger) saveLocked() {
	recs := make([]tunnelRecord, 0, len(l.entries))
	for pid, spec := range l.entries {
		recs = append(recs, tunnelRecord{PID: pid, Spec: spec})
	}
	data, _ := json.Marshal(recs)
	_ = os.MkdirAll(filepath.Dir(l.path), 0o700)
	if err := fsutil.WriteFile(l.path, data, 0o600); err != nil {
		log.Printf("rdp tunnels: could not record %s: %v", filepath.Base(l.path), err)
	}
}

func (l *tunnelLedger) add(pid int, spec string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries[pid] = spec
	l.saveLocked()
}

func (l *tunnelLedger) remove(pid int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, ok := l.entries[pid]; !ok {
		return
	}
	delete(l.entries, pid)
	l.saveLocked()
}

// reap ends every recorded tunnel that is still running as that exact ssh forward, then clears the ledger. A pid
// that now belongs to something else (it was reused) is left alone. It returns how many it ended.
func (l *tunnelLedger) reap() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	ended := 0
	for pid, spec := range l.entries {
		cmd, running := l.procCommand(pid)
		if running && isOurTunnel(cmd, spec) {
			if err := l.kill(pid); err == nil {
				ended++
				log.Printf("rdp tunnels: ended a tunnel left running by an earlier session (pid %d)", pid)
			}
		}
		delete(l.entries, pid)
	}
	l.saveLocked()
	return ended
}

// isOurTunnel is true only for an ssh command line that carries this exact forward.
func isOurTunnel(command, spec string) bool {
	fields := strings.Fields(command)
	if len(fields) == 0 || filepath.Base(fields[0]) != "ssh" {
		return false
	}
	for i := 0; i+1 < len(fields); i++ {
		if fields[i] == "-L" && fields[i+1] == spec {
			return true
		}
	}
	return false
}

func processCommand(pid int) (string, bool) {
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "command=").Output()
	if err != nil || len(strings.TrimSpace(string(out))) == 0 {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}

func killProcess(pid int) error {
	if err := syscall.Kill(pid, syscall.SIGTERM); err != nil && !errors.Is(err, syscall.ESRCH) {
		return err
	}
	return nil
}

// ledgerPath is where the tunnels are recorded: inside Relay's own config folder.
func ledgerPath(p paths.Paths) string { return filepath.Join(p.ConfigDir, "rdp-tunnels.json") }

// ledgers keeps one ledger per file for the life of the process.
var (
	ledgersMu sync.Mutex
	ledgers   = map[string]*tunnelLedger{}
)

func ledgerFor(p paths.Paths) *tunnelLedger {
	ledgersMu.Lock()
	defer ledgersMu.Unlock()
	path := ledgerPath(p)
	if l, ok := ledgers[path]; ok {
		return l
	}
	l := newTunnelLedger(path)
	ledgers[path] = l
	return l
}

// ReapOrphanTunnels ends ssh tunnels an earlier run of the app left behind. Call once at startup.
func ReapOrphanTunnels() {
	p, err := paths.Default()
	if err != nil {
		return
	}
	if _, err := os.Stat(ledgerPath(p)); errors.Is(err, fs.ErrNotExist) {
		return
	}
	ledgerFor(p).reap()
}
