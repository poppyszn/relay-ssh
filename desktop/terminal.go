package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/coder/websocket"
	"github.com/poppyszn/super-ssh-v2/internal/config"
	"github.com/poppyszn/super-ssh-v2/internal/core"
	"github.com/poppyszn/super-ssh-v2/internal/paths"
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
	argv, err := argvFor(q.Get("target"))
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
	s, err := startSession(argv, uint16(max(cols, 20)), uint16(max(rows, 5)))
	if err != nil {
		_ = c.Close(websocket.StatusInternalError, err.Error())
		return
	}
	s.run(r.Context(), c)
}

// argvFor maps a target to the process to run: "local" is the user's shell,
// anything else is a profile addressed by its Host alias in the generated config.
func argvFor(target string) ([]string, error) {
	if target == "" || target == "local" {
		sh := os.Getenv("SHELL")
		if sh == "" {
			sh = "/bin/sh"
		}
		return []string{sh}, nil
	}
	p, err := paths.Default()
	if err != nil {
		return nil, err
	}
	s, err := config.Load(p.StoreFile)
	if err != nil {
		return nil, err
	}
	prof, ok := core.ResolveProfile(s, target)
	if !ok {
		return nil, fmt.Errorf("profile %q not found", target)
	}
	if err := core.Regenerate(p, s); err != nil {
		return nil, err
	}
	return []string{"ssh", "-F", p.GenConfig, strings.TrimSpace(prof.Name)}, nil
}
