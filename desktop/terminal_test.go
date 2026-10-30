package main

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
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
	for _, origin := range []string{"wails://localhost", "http://wails.localhost", "wails://localhost:9245", "http://localhost:9245", "http://127.0.0.1:9245"} {
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

func TestDevOriginFollowsViteSource(t *testing.T) {
	t.Setenv("WAILS_VITE_PORT", "9345")
	if !allowedOrigin("wails://localhost:9345") || !allowedOrigin("http://localhost:9345") || allowedOrigin("http://localhost:9245") {
		t.Error("dev origin should follow WAILS_VITE_PORT")
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

// Profile sessions must start as the multiplexing master so the file browser can
// attach to their authenticated connection, and the profile name stays last.
func TestProfileSessionIsTheMultiplexMaster(t *testing.T) {
	t.Setenv("RELAY_SSH_HOME", t.TempDir())
	if err := (ProfilesService{}).Save("", Profile{Name: "web", Host: "h", User: "u", Port: 22, Auth: "agent"}); err != nil {
		t.Fatal(err)
	}
	argv, _, err := argvFor("web")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(argv, " ")
	for _, want := range []string{"ControlMaster=auto", "ControlPath=", "ControlPersist=60s"} {
		if runtime.GOOS != "windows" && !strings.Contains(joined, want) {
			t.Errorf("argv %q missing %q", joined, want)
		}
	}
	if argv[0] != "ssh" || argv[len(argv)-1] != "web" {
		t.Errorf("argv = %v, want ssh ... web", argv)
	}
}

// Where the Keychain is used (macOS), a connection whose key has a passphrase also asks it for the
// passphrase and keeps the unlocked key in the agent, so an adopted key is not asked for again after a
// restart. Connections that do not need it get no extra options (and no key is pushed into the agent).
func TestSessionsUseTheKeychainForEncryptedKeys(t *testing.T) {
	home := t.TempDir()
	t.Setenv("RELAY_SSH_HOME", home)
	dir := filepath.Join(home, "keys")
	_ = os.MkdirAll(dir, 0o700)
	locked := makeKey(t, dir, "locked", "hunter2")
	plain := makeKey(t, dir, "plain", "")
	svc := ProfilesService{}
	for name, p := range map[string]Profile{
		"locked": {Name: "locked", Host: "h", User: "u", Port: 22, Auth: "key", Key: locked},
		"plain":  {Name: "plain", Host: "h", User: "u", Port: 22, Auth: "key", Key: plain},
		"agent":  {Name: "agent", Host: "h", User: "u", Port: 22, Auth: "agent"},
	} {
		if err := svc.Save("", p); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	t.Cleanup(func(old bool) func() { return func() { keychainEnabled = old } }(keychainEnabled))
	for _, on := range []bool{true, false} {
		keychainEnabled = on
		for target, want := range map[string]bool{"locked": on, "plain": false, "agent": false} {
			argv, _, err := argvFor(target)
			if err != nil {
				t.Fatal(err)
			}
			joined := " " + strings.Join(argv, " ") + " "
			has := strings.Contains(joined, " UseKeychain=yes ") && strings.Contains(joined, " AddKeysToAgent=yes ") && strings.Contains(joined, " IgnoreUnknown=UseKeychain ")
			if has != want {
				t.Errorf("keychain %v, %s: options present = %v, want %v", on, target, has, want)
			}
		}
	}
}
