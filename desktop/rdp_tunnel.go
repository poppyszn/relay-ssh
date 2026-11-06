package main

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/poppyszn/relay-ssh/internal/paths"
)

// RdpTunnel manages an ssh child process creating a local port forward to an RDP host via a jump host.
type RdpTunnel struct {
	LocalPort int
	JumpAlias string
	Host      string
	Port      int

	cmd    *exec.Cmd
	cancel context.CancelFunc
	done   chan struct{}
	err    error
	mu     sync.Mutex
}

func checkTunnelArg(field, v string) error {
	if v == "" {
		return fmt.Errorf("%s is required", field)
	}
	if strings.HasPrefix(v, "-") {
		return fmt.Errorf("%s must not start with '-'", field)
	}
	for _, r := range v {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return fmt.Errorf("%s must not contain whitespace or control characters", field)
		}
	}
	return nil
}

func findFreePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("finding free port on 127.0.0.1: %w", err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

// StartRdpTunnel starts an ssh child process forwarding 127.0.0.1:<freePort> to host:port through jumpAlias.
func StartRdpTunnel(p paths.Paths, jumpAlias, host string, port int) (*RdpTunnel, error) {
	if err := checkTunnelArg("jump alias", jumpAlias); err != nil {
		return nil, err
	}
	if err := checkTunnelArg("host", host); err != nil {
		return nil, err
	}
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("port must be between 1 and 65535, got %d", port)
	}

	freePort, err := findFreePort()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	forwardSpec := fmt.Sprintf("127.0.0.1:%d:%s:%d", freePort, host, port)
	var args []string
	if _, err := os.Stat(p.GenConfig); err == nil {
		args = append(args, "-F", p.GenConfig)
	}
	args = append(args, "-o", "BatchMode=yes", "-N", "-L", forwardSpec, jumpAlias)

	cmd := exec.CommandContext(ctx, "ssh", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		cancel()
		return nil, fmt.Errorf("starting ssh tunnel: %w", err)
	}

	ledger := ledgerFor(p)
	ledger.add(cmd.Process.Pid, forwardSpec)

	tunnel := &RdpTunnel{
		LocalPort: freePort,
		JumpAlias: jumpAlias,
		Host:      host,
		Port:      port,
		cmd:       cmd,
		cancel:    cancel,
		done:      make(chan struct{}),
	}

	// Background monitor for child process exit
	go func() {
		werr := cmd.Wait()
		ledger.remove(cmd.Process.Pid) // it ended, so there is nothing left to clean up
		tunnel.mu.Lock()
		tunnel.err = werr
		tunnel.mu.Unlock()
		close(tunnel.done)
	}()

	// Readiness check: poll 127.0.0.1:freePort until open or process exits
	deadline := time.Now().Add(5 * time.Second)
	ready := false
	for time.Now().Before(deadline) {
		select {
		case <-tunnel.done:
			// Process exited early
			cancel()
			errText := strings.TrimSpace(stderr.String())
			if strings.Contains(errText, "Permission denied") || strings.Contains(errText, "Host key verification failed") || cmd.ProcessState != nil && !cmd.ProcessState.Success() {
				return nil, fmt.Errorf("Open %s first", jumpAlias)
			}
			return nil, fmt.Errorf("ssh tunnel exited prematurely: %s", errText)
		default:
		}

		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", freePort), 80*time.Millisecond)
		if err == nil {
			conn.Close()
			ready = true
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	if !ready {
		_ = tunnel.Close()
		return nil, fmt.Errorf("Open %s first (timed out establishing tunnel)", jumpAlias)
	}

	return tunnel, nil
}

// Close terminates the ssh tunnel process and waits (briefly) for it to be gone and its record cleared, so
// nothing is left running or half written behind it.
func (t *RdpTunnel) Close() error {
	t.mu.Lock()
	if t.cancel != nil {
		t.cancel()
	}
	if t.cmd != nil && t.cmd.Process != nil {
		_ = t.cmd.Process.Kill()
	}
	t.mu.Unlock() // not held while waiting: the goroutine that notices the exit takes it

	select {
	case <-t.done:
	case <-time.After(3 * time.Second):
	}
	return nil
}

// Wait returns a channel that is closed when the ssh tunnel process exits.
func (t *RdpTunnel) Wait() <-chan struct{} {
	return t.done
}

// Err returns the error that caused the tunnel to terminate, if any.
func (t *RdpTunnel) Err() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.err
}
