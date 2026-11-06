package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/poppyszn/relay-ssh/internal/paths"
)

// makeFakeSSH creates an executable fake `ssh` script in a temp bin directory prepended to PATH.
func makeFakeSSHBidirectional(t *testing.T, scriptBody string) {
	t.Helper()
	binDir := t.TempDir()
	sshPath := filepath.Join(binDir, "ssh")
	content := "#!/bin/sh\n" + scriptBody
	if err := os.WriteFile(sshPath, []byte(content), 0o755); err != nil {
		t.Fatalf("creating fake ssh: %v", err)
	}
	t.Setenv("PATH", binDir+":"+os.Getenv("PATH"))
}

func TestStartRdpTunnelSuccess(t *testing.T) {
	// Fake ssh: parses the -L argument (127.0.0.1:<port>:<host>:<port>),
	// binds to that port, and waits.
	script := `
for arg in "$@"; do
    case "$arg" in
        127.0.0.1:*)
            port=$(echo "$arg" | cut -d: -f2)
            exec python3 -c "import socket; s = socket.socket(); s.bind(('127.0.0.1', $port)); s.listen(5); [s.accept() for _ in range(100)]"
            ;;
    esac
done
sleep 10
`
	makeFakeSSHBidirectional(t, script)

	p := paths.New(t.TempDir())
	tunnel, err := StartRdpTunnel(p, "bastion", "192.168.1.100", 3389)
	if err != nil {
		t.Fatalf("StartRdpTunnel: %v", err)
	}

	if tunnel.LocalPort <= 0 {
		t.Fatalf("invalid local port: %d", tunnel.LocalPort)
	}
	if tunnel.JumpAlias != "bastion" {
		t.Errorf("jump alias = %q, want bastion", tunnel.JumpAlias)
	}

	// Verify local port is listening
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", tunnel.LocalPort), 500*time.Millisecond)
	if err != nil {
		t.Fatalf("could not dial tunnel port: %v", err)
	}
	conn.Close()

	// Close tunnel and verify it stops
	if err := tunnel.Close(); err != nil {
		t.Fatalf("tunnel.Close: %v", err)
	}

	select {
	case <-tunnel.Wait():
		// Tunnel stopped cleanly
	case <-time.After(2 * time.Second):
		t.Fatal("tunnel did not stop within 2s after Close()")
	}
}

func TestStartRdpTunnelAuthFailure(t *testing.T) {
	// Fake ssh that simulates an authentication failure requiring a prompt
	script := `
echo "Permission denied (publickey,keyboard-interactive)." >&2
exit 255
`
	makeFakeSSHBidirectional(t, script)

	p := paths.New(t.TempDir())
	_, err := StartRdpTunnel(p, "bastion", "192.168.1.100", 3389)
	if err == nil {
		t.Fatal("expected StartRdpTunnel to fail when jump host needs login, got nil")
	}

	if !strings.Contains(err.Error(), "Open bastion first") {
		t.Errorf("expected error to contain 'Open bastion first', got: %v", err)
	}
}

func TestStartRdpTunnelDeadTunnelDetection(t *testing.T) {
	// Fake ssh binds port, then exits after 300ms
	script := `
for arg in "$@"; do
    case "$arg" in
        127.0.0.1:*)
            port=$(echo "$arg" | cut -d: -f2)
            nc -l 127.0.0.1 "$port" &
            nc_pid=$!
            sleep 0.3
            kill $nc_pid 2>/dev/null
            exit 1
            ;;
    esac
done
exit 1
`
	makeFakeSSHBidirectional(t, script)

	p := paths.New(t.TempDir())
	tunnel, err := StartRdpTunnel(p, "bastion", "192.168.1.100", 3389)
	if err != nil {
		t.Fatalf("StartRdpTunnel: %v", err)
	}

	select {
	case <-tunnel.Wait():
		// Detected dead tunnel!
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for dead tunnel detection")
	}

	_ = tunnel.Close()
}

func TestRdpTunnelArgumentSafety(t *testing.T) {
	p := paths.New(t.TempDir())
	cases := []struct {
		jump string
		host string
		port int
	}{
		{"-bad-jump", "host", 3389},
		{"bastion\n", "host", 3389},
		{"bastion", "-bad-host", 3389},
		{"bastion", "host\x00", 3389},
		{"bastion", "host", 0},
		{"bastion", "host", 70000},
	}
	for _, tc := range cases {
		_, err := StartRdpTunnel(p, tc.jump, tc.host, tc.port)
		if err == nil {
			t.Errorf("expected error for unsafe arguments (%q, %q, %d), got nil", tc.jump, tc.host, tc.port)
		}
	}
}

// Close returns only once the ssh process has ended and its record is gone, so nothing is left running or half
// written behind it (and a test's temp folder can be removed straight away).
func TestTunnelCloseWaitsForTheProcessToEnd(t *testing.T) {
	root := t.TempDir()
	t.Setenv("RELAY_SSH_HOME", root)
	fake := t.TempDir()
	script := "#!/bin/sh\nfor a in \"$@\"; do case \"$a\" in 127.0.0.1:*:*) port=$(echo \"$a\" | cut -d: -f2);; esac; done\nexec python3 -c 'import socket,sys,time\ns=socket.socket();s.setsockopt(socket.SOL_SOCKET,socket.SO_REUSEADDR,1);s.bind((\"127.0.0.1\",int(sys.argv[1])));s.listen(1)\nwhile True: time.sleep(1)' \"$port\"\n"
	if err := os.WriteFile(filepath.Join(fake, "ssh"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", fake+":"+os.Getenv("PATH"))

	p := paths.New(root)
	if err := paths.EnsureDirs(p); err != nil {
		t.Fatal(err)
	}
	tun, err := StartRdpTunnel(p, "jump", "10.0.0.5", 3389)
	if err != nil {
		t.Fatal(err)
	}
	if recs := ledgerFor(p).read(); len(recs) != 1 {
		t.Fatalf("a running tunnel must be recorded, ledger has %v", recs)
	}
	pid := tun.cmd.Process.Pid
	tun.Close()
	select {
	case <-tun.done:
	default:
		t.Fatal("Close returned while the ssh process was still running")
	}
	if _, running := processCommand(pid); running {
		t.Errorf("ssh process %d still running after Close", pid)
	}
	if recs := ledgerFor(p).read(); len(recs) != 0 {
		t.Errorf("a closed tunnel must leave no record, ledger has %v", recs)
	}
}
