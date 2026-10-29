package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"os/user"
	"strings"
	"sync"

	"github.com/coder/websocket"
	"github.com/creack/pty"
	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/core"
	"github.com/poppyszn/relay-ssh/internal/paths"
)

const (
	readChunk = 64 * 1024
	// highWater is how many bytes may be in flight to the webview before Go
	// stops reading the PTY. The frontend acks after term.write completes.
	highWater = 1024 * 1024
	// maxPending bounds how much the reader buffers while the sender is busy.
	maxPending = 256 * 1024
)

type control struct {
	T    string `json:"t"`
	N    int    `json:"n,omitempty"`
	C    uint16 `json:"c,omitempty"`
	R    uint16 `json:"r,omitempty"`
	Kind string `json:"kind,omitempty"`
	User string `json:"user,omitempty"`
	Host string `json:"host,omitempty"`
}

// session ties one PTY process to one WebSocket. Go never interprets the
// bytes: they go PTY -> socket and socket -> PTY unchanged.
type session struct {
	cmd *exec.Cmd
	pty *os.File

	mu       sync.Mutex
	cond     *sync.Cond
	inFlight int
	pending  []byte
	eof      bool
	closed   bool

	target      string
	authed      bool
	totalFills  int
	filledConns map[string]bool
}

func startSession(argv, env []string, cols, rows uint16, target ...string) (*session, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Env = append(append(os.Environ(), "TERM=xterm-256color", "COLORTERM=truecolor"), env...)
	f, err := pty.StartWithSize(cmd, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		return nil, err
	}
	s := &session{
		cmd:         cmd,
		pty:         f,
		filledConns: make(map[string]bool),
	}
	if len(target) > 0 {
		s.target = target[0]
	}
	s.cond = sync.NewCond(&s.mu)
	return s, nil
}

func (s *session) close() {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	s.cond.Broadcast()
	s.pty.Close()
	_ = s.cmd.Process.Kill()
	_ = s.cmd.Wait()
}

// run pumps both directions until either side ends.
func (s *session) run(ctx context.Context, c *websocket.Conn) {
	defer s.close()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() { <-ctx.Done(); s.cond.Broadcast() }()
	c.SetReadLimit(1 << 20)

	go func() { // socket -> PTY, plus acks and resizes
		defer cancel()
		for {
			typ, data, err := c.Read(ctx)
			if err != nil {
				return
			}
			if typ == websocket.MessageBinary {
				if _, err := s.pty.Write(data); err != nil {
					return
				}
				continue
			}
			var m control
			if json.Unmarshal(data, &m) != nil {
				continue
			}
			switch m.T {
			case "ack":
				s.mu.Lock()
				s.inFlight -= m.N
				s.mu.Unlock()
				s.cond.Broadcast()
			case "resize":
				_ = pty.Setsize(s.pty, &pty.Winsize{Cols: m.C, Rows: m.R})
			case "authed":
				s.mu.Lock()
				s.authed = true
				s.mu.Unlock()
			case "fill":
				go s.handleFill(ctx, c, m)
			}
		}
	}()

	// A reader goroutine drains the PTY into `pending`; the sender below ships
	// whatever has accumulated as one frame. Producers like `seq` write ~12
	// bytes at a time, so one frame per read would cost millions of frames.
	go s.fill(ctx, cancel)

	var out []byte
	for {
		s.mu.Lock()
		for (len(s.pending) == 0 || s.inFlight >= highWater) && !s.eof && ctx.Err() == nil {
			s.cond.Wait()
		}
		if ctx.Err() != nil || (len(s.pending) == 0 && s.eof) {
			s.mu.Unlock()
			_ = c.Close(websocket.StatusNormalClosure, "process exited")
			return
		}
		out, s.pending = s.pending, out[:0]
		s.inFlight += len(out)
		s.mu.Unlock()
		s.cond.Broadcast() // wake the reader if it was waiting for room

		if err := c.Write(ctx, websocket.MessageBinary, out); err != nil {
			return
		}
	}
}

// fill reads the PTY into s.pending, pausing while the buffer is full.
func (s *session) fill(ctx context.Context, cancel context.CancelFunc) {
	buf := make([]byte, readChunk)
	for ctx.Err() == nil {
		n, err := s.pty.Read(buf)
		s.mu.Lock()
		for len(s.pending) >= maxPending && ctx.Err() == nil {
			s.cond.Wait()
		}
		s.pending = append(s.pending, buf[:n]...)
		if err != nil {
			s.eof = true
		}
		s.mu.Unlock()
		s.cond.Broadcast()
		if err != nil {
			return
		}
	}
}

func currentUser() string {
	u, err := user.Current()
	if err != nil {
		return os.Getenv("USER")
	}
	return u.Username
}

func (s *session) refuseFill(ctx context.Context, c *websocket.Conn) {
	if c != nil && ctx != nil {
		_ = c.Write(ctx, websocket.MessageText, []byte(`{"t":"fillrefused"}`))
	}
}

func (s *session) handleFill(ctx context.Context, c *websocket.Conn, m control) {
	s.mu.Lock()
	// Rule 8: The session was started by Relay for a saved connection (not a local shell).
	if s.target == "" || s.target == "local" {
		s.mu.Unlock()
		s.refuseFill(ctx, c)
		return
	}

	// Rule 7: Never fill after authentication.
	if s.authed {
		s.mu.Unlock()
		s.refuseFill(ctx, c)
		return
	}

	// Rule 6: Per-session cap of 2 fills total across all connections.
	if s.totalFills >= 2 {
		s.mu.Unlock()
		s.refuseFill(ctx, c)
		return
	}
	s.mu.Unlock()

	p, err := paths.Default()
	if err != nil {
		s.refuseFill(ctx, c)
		return
	}
	store, err := config.Load(p.StoreFile)
	if err != nil {
		s.refuseFill(ctx, c)
		return
	}

	// Rule 2: The target profile's auth method is password.
	targetProf, ok := core.ResolveProfile(store, s.target)
	if !ok || targetProf.Auth != config.AuthPassword {
		s.refuseFill(ctx, c)
		return
	}

	// Rule 1: Vault must be unlocked.
	v := getVault()
	if !v.Status().Unlocked {
		s.refuseFill(ctx, c)
		return
	}

	var resolvedConn string

	switch m.Kind {
	case "generic":
		// Rule 4: Generic prompts ("Password:" with no host) are only ever answered
		// with the session's own target secret, and only if the session has no jump host.
		if targetProf.Jump != "" {
			s.refuseFill(ctx, c)
			return
		}
		resolvedConn = targetProf.Name

	case "account":
		// Rule 3: Account prompts: resolve which saved connection the prompt is for
		// by matching user and host against saved profiles (host compared to profile's
		// Host case-insensitively, and to its name). If match is ambiguous refuse.
		if m.Host == "" {
			s.refuseFill(ctx, c)
			return
		}
		curU := currentUser()
		var matchingProfiles []string
		for _, prof := range store.Profiles {
			hostMatch := strings.EqualFold(prof.Host, m.Host) || strings.EqualFold(prof.Name, m.Host)
			if !hostMatch {
				continue
			}
			userMatch := (prof.User == m.User) || (prof.User == "" && (m.User == curU || m.User == "root"))
			if !userMatch {
				continue
			}
			if v.Has(prof.Name) {
				matchingProfiles = append(matchingProfiles, prof.Name)
			}
		}

		if len(matchingProfiles) != 1 {
			s.refuseFill(ctx, c)
			return
		}
		resolvedConn = matchingProfiles[0]

	default:
		s.refuseFill(ctx, c)
		return
	}

	// Rule 5: At most one fill per session per connection, tracked in Go.
	s.mu.Lock()
	if s.filledConns[resolvedConn] {
		s.mu.Unlock()
		s.refuseFill(ctx, c)
		return
	}
	s.mu.Unlock()

	// Rule 1: Vault has a secret for the resolved connection.
	secret, has, err := v.Get(resolvedConn)
	if err != nil || !has || secret == "" {
		s.refuseFill(ctx, c)
		return
	}

	s.mu.Lock()
	if s.authed || s.totalFills >= 2 || s.filledConns[resolvedConn] {
		s.mu.Unlock()
		s.refuseFill(ctx, c)
		return
	}
	s.filledConns[resolvedConn] = true
	s.totalFills++
	s.mu.Unlock()

	// Write password + "\n" to PTY
	secretBytes := []byte(secret + "\n")
	defer zeroBytes(secretBytes)
	if _, err := s.pty.Write(secretBytes); err != nil {
		return
	}

	if c != nil && ctx != nil {
		_ = c.Write(ctx, websocket.MessageText, []byte(`{"t":"filled"}`))
	}
}
