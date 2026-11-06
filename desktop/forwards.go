package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/core"
	"github.com/poppyszn/relay-ssh/internal/paths"
	"github.com/poppyszn/relay-ssh/internal/sshconfig"
)

// ForwardsService starts and stops port forwards on a live connection using
// OpenSSH multiplexing (`ssh -O forward` / `-O cancel`) against the master that
// the connection's terminal tab started. No tunnelling code lives here.
type ForwardsService struct {
	extraOpts []string // test hook: skip host-key prompts against a throwaway server

	mu      sync.Mutex
	added   map[string][]ForwardInfo // runtime forwards by connection; ssh cannot list them
	stopped map[string]bool          // forwards cancelled here, which cannot be probed (remote ones)
}

// ForwardInfo is one forward as shown in the UI.
type ForwardInfo struct {
	Type      string `json:"type"` // local | remote | dynamic
	Listen    string `json:"listen"`
	Target    string `json:"target"`
	Source    string `json:"source"` // profile | runtime
	Listening bool   `json:"listening"`
}

// ForwardStatus is a connection's live state.
type ForwardStatus struct {
	Connected bool          `json:"connected"`
	Message   string        `json:"message"`
	Forwards  []ForwardInfo `json:"forwards"`
}

var (
	listenRe = regexp.MustCompile(`^(?:[A-Za-z0-9.\-]+:|\[[0-9a-fA-F:]+\]:)?\d{1,5}$`)
	targetRe = regexp.MustCompile(`^(?:[A-Za-z0-9.\-_]+|\[[0-9a-fA-F:]+\]):\d{1,5}$`)
)

func validPort(s string) bool {
	i := strings.LastIndex(s, ":")
	n, err := strconv.Atoi(s[i+1:])
	return err == nil && n >= 1 && n <= 65535
}

// spec builds the argument ssh expects after -L, -R or -D, rejecting anything
// that is not a plain port or host:port.
func spec(kind, listen, target string) (flag, arg string, err error) {
	if !listenRe.MatchString(listen) || !validPort(listen) {
		return "", "", fmt.Errorf("%q is not a valid port (use 8080 or 127.0.0.1:8080)", listen)
	}
	switch kind {
	case "dynamic":
		return "-D", listen, nil
	case "local", "remote":
		if !targetRe.MatchString(target) || !validPort(target) {
			return "", "", fmt.Errorf("%q is not a valid target (use host:port)", target)
		}
		f := "-L"
		if kind == "remote" {
			f = "-R"
		}
		return f, listen + ":" + target, nil
	}
	return "", "", fmt.Errorf("unknown forward type %q", kind)
}

// localPort is the port to probe for a local or SOCKS forward.
func localPort(listen string) string { return listen[strings.LastIndex(listen, ":")+1:] }

func (f *ForwardsService) ssh(target string, args ...string) (string, error) {
	if len(muxOptions(false)) == 0 {
		return "", errors.New("Changing forwards on a live connection needs OpenSSH connection sharing, which is not available on this system.")
	}
	p, err := paths.Default()
	if err != nil {
		return "", err
	}
	store, err := config.Load(p.StoreFile)
	if err != nil {
		return "", err
	}
	prof, ok := core.ResolveProfile(store, target)
	if !ok {
		return "", fmt.Errorf("connection %q not found", target)
	}
	if err := core.Regenerate(p, store); err != nil {
		return "", err
	}
	full := []string{"-F", p.GenConfig, "-o", "BatchMode=yes"}
	full = append(full, muxOptions(false)...)
	full = append(full, f.extraOpts...)
	full = append(full, args...)
	full = append(full, sshconfig.Alias(store, prof))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ssh", full...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Run(); err != nil {
		return out.String(), forwardErr(errb.String(), err)
	}
	return out.String() + errb.String(), nil
}

// forwardErr turns ssh's complaint into something actionable.
func forwardErr(stderr string, err error) error {
	s := strings.TrimSpace(stderr)
	switch {
	case strings.Contains(s, "No such file or directory"), strings.Contains(s, "Control socket connect"), strings.Contains(s, "Connection refused") && strings.Contains(s, "ontrol"):
		return errors.New("This connection is not open. Open it in a tab first, then manage its forwards here.")
	case strings.Contains(s, "Address already in use"), strings.Contains(s, "cannot listen"):
		return errors.New("That port is already in use. Pick a different one.")
	case strings.Contains(s, "Could not request"), strings.Contains(s, "forwarding request failed"):
		return errors.New("The server refused the forward. It may not allow port forwarding, or that remote port is taken.")
	case s != "":
		lines := strings.Split(s, "\n")
		return errors.New(strings.TrimSpace(lines[len(lines)-1]))
	}
	return err
}

func key(target, kind, listen, dest string) string {
	return target + "|" + kind + "|" + listen + "|" + dest
}

// listening probes local and SOCKS forwards. A remote forward is held by the
// server, so it counts as running unless it was stopped here.
func (f *ForwardsService) listening(target, kind, listen, dest string) bool {
	if kind == "remote" {
		f.mu.Lock()
		defer f.mu.Unlock()
		return !f.stopped[key(target, kind, listen, dest)]
	}
	c, err := net.DialTimeout("tcp", "127.0.0.1:"+localPort(listen), 400*time.Millisecond)
	if err != nil {
		return false
	}
	c.Close()
	return true
}

// Status reports whether the connection is open and which forwards it has:
// the saved connection's own tunnels plus any added here.
func (f *ForwardsService) Status(target string) (ForwardStatus, error) {
	out := ForwardStatus{Forwards: []ForwardInfo{}}
	if _, err := f.ssh(target, "-O", "check"); err != nil {
		out.Message = err.Error()
		return out, nil // not connected is a state, not a failure
	}
	out.Connected = true

	p, err := paths.Default()
	if err != nil {
		return out, err
	}
	store, err := config.Load(p.StoreFile)
	if err != nil {
		return out, err
	}
	prof, _ := core.ResolveProfile(store, target)
	seen := map[string]bool{}
	add := func(fi ForwardInfo) {
		key := fi.Type + "|" + fi.Listen + "|" + fi.Target
		if seen[key] {
			return
		}
		seen[key] = true
		fi.Listening = f.listening(target, fi.Type, fi.Listen, fi.Target)
		out.Forwards = append(out.Forwards, fi)
	}
	for _, t := range prof.Tunnels {
		add(ForwardInfo{Type: string(t.Type), Listen: t.Listen, Target: t.Target, Source: "profile"})
	}
	// Copy under the lock, probe outside it: listening() takes the lock itself.
	f.mu.Lock()
	runtime := append([]ForwardInfo(nil), f.added[target]...)
	f.mu.Unlock()
	for _, fi := range runtime {
		add(fi)
	}
	return out, nil
}

// Add opens a forward on the live connection.
func (f *ForwardsService) Add(target, kind, listen, dest string) error {
	flag, arg, err := spec(kind, listen, dest)
	if err != nil {
		return err
	}
	// ssh reports success if it can bind only one of ::1 and 127.0.0.1, which would
	// leave another program answering on the port we show as forwarded. Check first.
	if kind != "remote" {
		addr := listen
		if !strings.Contains(listen, ":") {
			addr = "127.0.0.1:" + listen
		}
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return errors.New("That port is already in use. Pick a different one.")
		}
		ln.Close()
	}
	if _, err := f.ssh(target, "-O", "forward", flag, arg); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.added == nil {
		f.added = map[string][]ForwardInfo{}
	}
	delete(f.stopped, key(target, kind, listen, dest))
	for _, fi := range f.added[target] {
		if fi.Type == kind && fi.Listen == listen && fi.Target == dest {
			return nil // restarting a runtime forward that is already tracked
		}
	}
	f.added[target] = append(f.added[target], ForwardInfo{Type: kind, Listen: listen, Target: dest, Source: "runtime"})
	return nil
}

// Cancel closes a forward on the live connection, whether it came from the
// saved connection or was added here.
func (f *ForwardsService) Cancel(target, kind, listen, dest string) error {
	flag, arg, err := spec(kind, listen, dest)
	if err != nil {
		return err
	}
	if _, err := f.ssh(target, "-O", "cancel", flag, arg); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.stopped == nil {
		f.stopped = map[string]bool{}
	}
	f.stopped[key(target, kind, listen, dest)] = true
	kept := f.added[target][:0]
	for _, fi := range f.added[target] {
		if !(fi.Type == kind && fi.Listen == listen && fi.Target == dest) {
			kept = append(kept, fi)
		}
	}
	f.added[target] = kept
	return nil
}
