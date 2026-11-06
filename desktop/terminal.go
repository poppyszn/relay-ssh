package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/core"
	"github.com/poppyszn/relay-ssh/internal/paths"
	"github.com/poppyszn/relay-ssh/internal/sshconfig"
)

// allowedOrigins are the only pages permitted to open a terminal socket: the
// packaged app on macOS/Linux, on Windows, and the Vite dev server. Wails dev
// reports its page as "wails://localhost:<port>" (and "http://localhost:<port>"
// or 127.0.0.1 for a plain browser) where the port is WAILS_VITE_PORT.
func allowedOrigin(origin string) bool {
	switch origin {
	case "wails://localhost", "http://wails.localhost":
		return true
	}
	port := os.Getenv("WAILS_VITE_PORT")
	if port == "" {
		port = "9245"
	}
	for _, o := range []string{"wails://localhost:", "http://localhost:", "http://127.0.0.1:"} {
		if origin == o+port {
			return true
		}
	}
	return false
}

// Endpoint is what the frontend needs to open a terminal socket.
type Endpoint struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

// TerminalService serves terminal bytes over a loopback WebSocket, guarded by
// a per-launch random token and an Origin allow-list.
type TerminalService struct {
	endpoint Endpoint
	onResult func(string)
}

func newTerminalService(onResult func(string)) (*TerminalService, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	tok := make([]byte, 24)
	if _, err := rand.Read(tok); err != nil {
		return nil, err
	}
	t := &TerminalService{
		endpoint: Endpoint{URL: "ws://" + ln.Addr().String() + "/pty", Token: hex.EncodeToString(tok)},
		onResult: onResult,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/pty", t.handle)
	go http.Serve(ln, mux)
	return t, nil
}

// Endpoint returns the socket URL and token. Bound to the frontend.
func (t *TerminalService) Endpoint() Endpoint { return t.endpoint }

// CustomChrome reports whether the frontend should draw the window buttons
// itself (the window is frameless). Bound to the frontend.
func (t *TerminalService) CustomChrome() bool { return customChrome() }

// BenchMode returns the RELAY_BENCH value ("" when not benchmarking). Bound to the frontend.
func (t *TerminalService) BenchMode() string { return os.Getenv("RELAY_BENCH") }

// ReportBench receives the frontend's benchmark JSON. Bound to the frontend.
func (t *TerminalService) ReportBench(result string) { t.onResult(result) }

func (t *TerminalService) handle(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	if q.Get("token") != t.endpoint.Token {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	argv, env, err := argvFor(q.Get("target"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cols, _ := strconv.Atoi(q.Get("cols"))
	rows, _ := strconv.Atoi(q.Get("rows"))
	if origin := r.Header.Get("Origin"); !allowedOrigin(origin) {
		log.Printf("terminal socket: rejected origin %q", origin)
		http.Error(w, "bad origin", http.StatusForbidden)
		return
	}
	// The library's own Origin check compares against Host, which never matches
	// a custom-scheme webview, so it is skipped in favour of the exact list above.
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	// A session that attaches to a login that is already open never authenticates, so
	// ssh prints no marker for it: say so, so the page knows no prompt can follow.
	if t := q.Get("target"); t != "" && t != "local" && masterAlive(argv) {
		_ = c.Write(r.Context(), websocket.MessageText, []byte(`{"t":"mux"}`))
	}
	s, err := startSession(argv, env, uint16(max(cols, 20)), uint16(max(rows, 5)), q.Get("target"))
	if err != nil {
		_ = c.Close(websocket.StatusInternalError, err.Error())
		return
	}
	s.run(r.Context(), c)
}

// argvFor maps a target to the process to run: "local" is the user's shell,
// anything else is a profile addressed by its Host alias in the generated config.
func argvFor(target string) (argv, env []string, err error) {
	if target == "" || target == "local" {
		sh := os.Getenv("SHELL")
		if sh == "" {
			sh = "/bin/sh"
		}
		argv, env = shellCommand(sh, integrationDir())
		return argv, env, nil
	}
	p, err := paths.Default()
	if err != nil {
		return nil, nil, err
	}
	store, err := config.Load(p.StoreFile)
	if err != nil {
		return nil, nil, err
	}
	prof, ok := core.ResolveProfile(store, target)
	if !ok {
		return nil, nil, fmt.Errorf("profile %q not found", target)
	}
	if err := core.Regenerate(p, store); err != nil {
		return nil, nil, err
	}
	// The terminal's connection is the multiplexing master, so file transfers
	// (files.go) reuse its authentication instead of asking again.
	argv = []string{"ssh", "-F", p.GenConfig}
	argv = append(argv, muxOptions(true)...)
	// Once ssh has authenticated it runs this on this machine, before the remote shell
	// starts. The page uses the marker to know the login is done (see loginMarker).
	argv = append(argv, "-o", "PermitLocalCommand=yes", "-o", "LocalCommand="+loginMarker)
	if keychainEnabled && prof.Auth == config.AuthKey && keyIsEncrypted(p, prof.Key) {
		// macOS: ask the Keychain for the key's passphrase and keep the unlocked key in the agent. The
		// generated config is read alone (-F), so the user's own ~/.ssh/config settings do not apply here.
		argv = append(argv, "-o", "IgnoreUnknown=UseKeychain", "-o", "UseKeychain=yes", "-o", "AddKeysToAgent=yes")
	}
	return append(argv, strings.TrimSpace(sshconfig.Alias(store, prof))), nil, nil
}

// loginMarker is the LocalCommand that prints a private escape sequence (OSC 7771) into
// the terminal. It only reaches the page, whose parser swallows it, and it is passed to
// this ssh alone, never written to the generated config, so a jump host's child ssh
// (whose stdout carries the tunnelled stream) cannot run it.
const loginMarker = `printf '\033]7771;auth\007'`

// masterAlive reports whether the connection in argv (an argv from argvFor) already has a
// live shared login to attach to.
func masterAlive(argv []string) bool {
	if len(argv) < 2 {
		return false
	}
	check := append(append([]string{}, argv[:len(argv)-1]...), "-O", "check", argv[len(argv)-1])
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return exec.CommandContext(ctx, check[0], check[1:]...).Run() == nil
}
