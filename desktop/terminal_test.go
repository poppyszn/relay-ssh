package main

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func dial(t *testing.T, svc *TerminalService, origin, token string) (*websocket.Conn, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, svc.endpoint.URL+"?target=local&cols=80&rows=24&token="+token, &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": {origin}},
	})
	return c, err
}

// Every origin the app can really load from must be able to open a shell:
// the packaged app, Windows, and the Wails dev server (which uses "localhost",
// not 127.0.0.1).
func TestLocalShellWorksFromEveryAppOrigin(t *testing.T) {
	svc, err := newTerminalService(func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("SHELL", "/bin/sh")
	for _, origin := range []string{"wails://localhost", "http://wails.localhost", "http://localhost:9245", "http://127.0.0.1:9245"} {
		t.Run(origin, func(t *testing.T) {
			c, err := dial(t, svc, origin, svc.endpoint.Token)
			if err != nil {
				t.Fatalf("dial: %v", err)
			}
			defer c.CloseNow()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := c.Write(ctx, websocket.MessageBinary, []byte("echo relay-$((6*7))\n")); err != nil {
				t.Fatal(err)
			}
			var got strings.Builder
			for !strings.Contains(got.String(), "relay-42\r\n") {
				_, data, err := c.Read(ctx)
				if err != nil {
					t.Fatalf("read: %v (got %q)", err, got.String())
				}
				got.Write(data)
			}
		})
	}
}

func TestTerminalSocketRejectsBadOriginAndToken(t *testing.T) {
	svc, err := newTerminalService(func(string) {})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dial(t, svc, "https://evil.example", svc.endpoint.Token); err == nil {
		t.Error("foreign origin must be rejected")
	}
	if _, err := dial(t, svc, "wails://localhost", "wrong"); err == nil {
		t.Error("wrong token must be rejected")
	}
}
