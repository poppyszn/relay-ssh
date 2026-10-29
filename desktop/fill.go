package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"strings"

	"github.com/coder/websocket"
	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/core"
	"github.com/poppyszn/relay-ssh/internal/paths"
)

// bindingOf names what a saved password is for: the host it is sent to, the user, the port
// and the jump host in front. A password is only released while these are unchanged, so a
// connection that was edited outside the app (or restored from a backup) with a new host
// cannot make Relay hand the old password to that host.
func bindingOf(p config.Profile) string {
	port := p.Port
	if port == 0 {
		port = 22
	}
	return fmt.Sprintf("%s|%s|%d|%s", strings.ToLower(p.Host), p.User, port, p.Jump)
}

// jumpChain returns the connections ssh authenticates against, outermost jump host first and
// the target last. A loop or a missing jump profile ends the chain.
func jumpChain(store config.Store, target config.Profile) []config.Profile {
	chain := []config.Profile{target}
	seen := map[string]bool{target.Name: true}
	for cur := target; cur.Jump != "" && len(chain) < 8; {
		next, ok := core.ResolveProfile(store, cur.Jump)
		if !ok || seen[next.Name] {
			break
		}
		seen[next.Name] = true
		chain = append([]config.Profile{next}, chain...)
		cur = next
	}
	return chain
}

func currentUser() string {
	u, err := user.Current()
	if err != nil {
		return os.Getenv("USER")
	}
	return u.Username
}

func (s *session) refuseFill(ctx context.Context, c *websocket.Conn, why string) {
	if c != nil && ctx != nil {
		msg := `{"t":"fillrefused"}`
		if why != "" {
			msg = `{"t":"fillrefused","why":"` + why + `"}`
		}
		_ = c.Write(ctx, websocket.MessageText, []byte(msg))
	}
}

// handleFill answers a password prompt from the vault when every one of these holds. The
// prompt text comes from the remote side and can be forged, so the checks never trust it
// beyond picking which of the session's own connections the prompt is for.
//
//  1. The session was opened for a saved connection, not a local shell.
//  2. The login has not finished (a password asked for later is not ssh's).
//  3. The target connection uses password auth and the vault is unlocked.
//  4. Only connections in the target's own jump chain can be filled: an account prompt
//     ("user@host's password:") must match exactly one of them by user and host. A prompt
//     naming any other saved host is refused, so a server cannot ask for another server's password.
//  5. A bare "Password:" is answered only for a target without a jump host.
//  6. Hops are filled in the order ssh authenticates them, each at most once, and no
//     more times in total than there are hops.
//  7. The saved password is bound to the connection's current host, user, port and jump host.
func (s *session) handleFill(ctx context.Context, c *websocket.Conn, m control) {
	s.mu.Lock()
	if s.target == "" || s.target == "local" || s.authed {
		s.mu.Unlock()
		s.refuseFill(ctx, c, "")
		return
	}
	s.mu.Unlock()

	p, err := paths.Default()
	if err != nil {
		s.refuseFill(ctx, c, "")
		return
	}
	store, err := config.Load(p.StoreFile)
	if err != nil {
		s.refuseFill(ctx, c, "")
		return
	}
	target, ok := core.ResolveProfile(store, s.target)
	v := getVault()
	if !ok || target.Auth != config.AuthPassword || !v.Status().Unlocked {
		s.refuseFill(ctx, c, "")
		return
	}
	chain := jumpChain(store, target)

	hop := -1
	switch m.Kind {
	case "generic":
		if len(chain) == 1 {
			hop = 0
		}
	case "account":
		hop = matchHop(chain, m, v)
	}
	if hop < 0 {
		s.refuseFill(ctx, c, "")
		return
	}
	prof := chain[hop]

	if !s.mayFill(prof.Name, hop, len(chain)) {
		s.refuseFill(ctx, c, "")
		return
	}
	secret, has, err := v.Get(prof.Name, bindingOf(prof))
	if err != nil || !has {
		why := ""
		if errors.Is(err, ErrBindingChanged) {
			why = "changed"
		}
		s.refuseFill(ctx, c, why)
		return
	}

	// Re-check and claim the slot together: another prompt may have been answered meanwhile.
	if !s.claimFill(prof.Name, hop, len(chain)) {
		s.refuseFill(ctx, c, "")
		return
	}
	secretBytes := []byte(secret + "\n")
	defer zeroBytes(secretBytes)
	if _, err := s.pty.Write(secretBytes); err != nil {
		return
	}
	if c != nil && ctx != nil {
		_ = c.Write(ctx, websocket.MessageText, []byte(`{"t":"filled"}`))
	}
}

// matchHop finds the one password hop in the chain an account prompt is for. ssh prints the
// real host name, not the profile name, and the local user when the profile names none.
func matchHop(chain []config.Profile, m control, v *Vault) int {
	if m.Host == "" {
		return -1
	}
	found := -1
	for i, prof := range chain {
		u := prof.User
		if u == "" {
			u = currentUser()
		}
		if prof.Auth != config.AuthPassword || !strings.EqualFold(prof.Host, m.Host) || u != m.User || !v.Has(prof.Name) {
			continue
		}
		if found >= 0 {
			return -1 // two hops look the same: cannot tell which is asking
		}
		found = i
	}
	return found
}

func (s *session) mayFill(conn string, hop, hops int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.authed && hop >= s.nextHop && !s.filledConns[conn] && s.totalFills < hops
}

func (s *session) claimFill(conn string, hop, hops int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.authed || hop < s.nextHop || s.filledConns[conn] || s.totalFills >= hops {
		return false
	}
	s.filledConns[conn] = true
	s.totalFills++
	s.nextHop = hop + 1
	return true
}
