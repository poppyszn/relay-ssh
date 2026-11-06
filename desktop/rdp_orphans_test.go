package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// If Relay is force-quit or crashes, the ssh process forwarding to an RDP host keeps running. Each tunnel is
// recorded, and the next start ends the survivors that are still exactly that tunnel.
func newLedger(t *testing.T, procs map[int]string) (*tunnelLedger, *[]int) {
	t.Helper()
	var killed []int
	l := newTunnelLedger(filepath.Join(t.TempDir(), "rdp-tunnels.json"))
	l.procCommand = func(pid int) (string, bool) { c, ok := procs[pid]; return c, ok }
	l.kill = func(pid int) error { killed = append(killed, pid); delete(procs, pid); return nil }
	return l, &killed
}

const tunnelSpec = "127.0.0.1:52345:172.31.35.231:8933"

func TestOrphanedTunnelsAreEndedOnTheNextStart(t *testing.T) {
	procs := map[int]string{
		101: "ssh -F /Users/u/.ssh/relay-ssh.d/config -o BatchMode=yes -N -L " + tunnelSpec + " bastion",
	}
	l, killed := newLedger(t, procs)
	l.add(101, tunnelSpec)

	// A new run of the app: a fresh ledger over the same file.
	next := newTunnelLedger(l.path)
	next.procCommand = l.procCommand
	next.kill = l.kill
	if n := next.reap(); n != 1 || len(*killed) != 1 || (*killed)[0] != 101 {
		t.Fatalf("the surviving tunnel was not ended: reaped %d, killed %v", n, *killed)
	}
	if data, _ := os.ReadFile(l.path); len(data) > 0 && string(data) != "[]" && string(data) != "[]\n" {
		t.Errorf("the ledger should be empty after a reap, got %s", data)
	}
}

func TestReapNeverKillsAnUnrelatedProcess(t *testing.T) {
	procs := map[int]string{
		201: "/usr/bin/python3 server.py",                              // the pid was reused by something else
		202: "ssh -N -L 127.0.0.1:9999:other.example.com:22 elsewhere", // an ssh, but not our tunnel
		203: "sshd: user@notty -L " + tunnelSpec,                       // mentions the forward but is not ssh
		204: "ssh -N -L " + tunnelSpec + " bastion",                    // the real one
	}
	l, killed := newLedger(t, procs)
	for pid := range procs {
		l.add(pid, tunnelSpec)
	}
	next := newTunnelLedger(l.path)
	next.procCommand, next.kill = l.procCommand, l.kill
	next.reap()
	sort.Ints(*killed)
	if len(*killed) != 1 || (*killed)[0] != 204 {
		t.Fatalf("only the exact tunnel may be killed, killed %v", *killed)
	}
}

func TestFinishedTunnelsLeaveNoRecord(t *testing.T) {
	procs := map[int]string{301: "ssh -N -L " + tunnelSpec + " bastion"}
	l, killed := newLedger(t, procs)
	l.add(301, tunnelSpec)
	l.remove(301) // the tunnel was closed normally
	next := newTunnelLedger(l.path)
	next.procCommand, next.kill = l.procCommand, l.kill
	if n := next.reap(); n != 0 || len(*killed) != 0 {
		t.Fatalf("a tunnel that was closed normally must not be touched: reaped %d, killed %v", n, *killed)
	}
}

func TestReapToleratesAMissingOrDamagedLedger(t *testing.T) {
	l := newTunnelLedger(filepath.Join(t.TempDir(), "none.json"))
	if n := l.reap(); n != 0 {
		t.Fatalf("no ledger: %d", n)
	}
	bad := filepath.Join(t.TempDir(), "rdp-tunnels.json")
	os.WriteFile(bad, []byte("{ not json"), 0o600)
	l = newTunnelLedger(bad)
	if n := l.reap(); n != 0 {
		t.Fatalf("damaged ledger: %d", n)
	}
}

// The real process lookup and kill, against a harmless child process.
func TestProcessCommandAndKillWorkOnARealProcess(t *testing.T) {
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Skip("no sleep command")
	}
	done := make(chan struct{})
	go func() { cmd.Wait(); close(done) }()
	defer cmd.Process.Kill()

	got, ok := processCommand(cmd.Process.Pid)
	if !ok || !strings.HasPrefix(got, "sleep") {
		t.Fatalf("processCommand = %q, %v", got, ok)
	}
	if isOurTunnel(got, tunnelSpec) {
		t.Fatal("sleep is not an ssh tunnel")
	}
	if err := killProcess(cmd.Process.Pid); err != nil {
		t.Fatal(err)
	}
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("the process was not ended")
	}
	if _, ok := processCommand(cmd.Process.Pid); ok {
		t.Error("an ended process must not be reported as running")
	}
}
