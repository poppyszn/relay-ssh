package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// generateTestCertificate creates an in-memory self-signed certificate for fake RDP servers.
func generateTestCertificate(t *testing.T) tls.Certificate {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(123456),
		Subject: pkix.Name{
			CommonName: "fake-rdp-server",
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("CreateCertificate: %v", err)
	}
	return tls.Certificate{
		Certificate: [][]byte{der},
		PrivateKey:  priv,
	}
}

// startFakeRdpServer runs an in-process RDP server that speaks X.224 negotiation and upgrades to TLS.
func startFakeRdpServer(t *testing.T, cert tls.Certificate) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("fake server listen: %v", err)
	}

	stopCh := make(chan struct{})
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-stopCh:
					return
				default:
					return
				}
			}

			wg.Add(1)
			go func(c net.Conn) {
				defer wg.Done()
				defer c.Close()

				// 1. Read client X.224 CR (TPKT 4 bytes + X.224 CR)
				header := make([]byte, 4)
				if _, err := io.ReadFull(c, header); err != nil {
					return
				}
				pktLen := int(header[2])<<8 | int(header[3])
				if pktLen > 4 {
					payload := make([]byte, pktLen-4)
					if _, err := io.ReadFull(c, payload); err != nil {
						return
					}
				}

				// 2. Send server X.224 CC (Connection Confirm)
				x224CC := []byte{0x03, 0x00, 0x00, 0x0b, 0x06, 0xd0, 0x00, 0x00, 0x12, 0x34, 0x00}
				if _, err := c.Write(x224CC); err != nil {
					return
				}

				// 3. Upgrade to TLS
				tlsConn := tls.Server(c, &tls.Config{
					Certificates: []tls.Certificate{cert},
				})
				if err := tlsConn.Handshake(); err != nil {
					return
				}

				// 4. Echo loop: relay bytes back and forth until closed
				buf := make([]byte, 32*1024)
				for {
					n, err := tlsConn.Read(buf)
					if n > 0 {
						if _, werr := tlsConn.Write(buf[:n]); werr != nil {
							return
						}
					}
					if err != nil {
						return
					}
				}
			}(conn)
		}
	}()

	cleanup := func() {
		close(stopCh)
		ln.Close()
		wg.Wait()
	}

	return ln.Addr().String(), cleanup
}

func setupGateway(t *testing.T) (*RdpGateway, *RdpCertStore, func()) {
	t.Helper()
	storePath := filepath.Join(t.TempDir(), "rdp_hosts.toml")
	certStore, err := LoadRdpCertStore(storePath)
	if err != nil {
		t.Fatalf("LoadRdpCertStore: %v", err)
	}

	gw, err := NewRdpGateway(certStore, nil)
	if err != nil {
		t.Fatalf("NewRdpGateway: %v", err)
	}

	return gw, certStore, func() {
		gw.Close()
	}
}

func TestGatewayHandshakeAndByteRelay(t *testing.T) {
	cert := generateTestCertificate(t)
	serverAddr, stopServer := startFakeRdpServer(t, cert)
	defer stopServer()

	gw, certStore, stopGw := setupGateway(t)
	defer stopGw()

	// Pre-trust the server's certificate
	fp := FingerprintSHA256(cert.Certificate[0])
	if err := certStore.Trust(serverAddr, fp); err != nil {
		t.Fatalf("Trust: %v", err)
	}

	token, wsURL, err := gw.RegisterSession("sess-1", serverAddr, 0)
	if err != nil {
		t.Fatalf("RegisterSession: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Connect via WebSocket with allowed origin
	wsConn, _, err := websocket.Dial(ctx, wsURL+"?token="+token, &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": []string{"wails://localhost"}},
	})
	if err != nil {
		t.Fatalf("websocket.Dial: %v", err)
	}
	defer wsConn.Close(websocket.StatusNormalClosure, "done")

	// Send client RDCleanPath request PDU
	crPdu := RDCleanPathPdu{
		Version:           RDCleanPathVersion1,
		Destination:       serverAddr,
		ProxyAuth:         token,
		X224ConnectionPdu: []byte{0x03, 0x00, 0x00, 0x0b, 0x06, 0xe0, 0x00, 0x00, 0x00, 0x00, 0x00},
	}
	derReq, err := EncodeRDCleanPath(crPdu)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	if err := wsConn.Write(ctx, websocket.MessageBinary, derReq); err != nil {
		t.Fatalf("ws write: %v", err)
	}

	// Read gateway response PDU
	msgType, derResp, err := wsConn.Read(ctx)
	if err != nil {
		t.Fatalf("ws read: %v", err)
	}
	if msgType != websocket.MessageBinary {
		t.Fatalf("expected binary message, got %v", msgType)
	}

	resp, err := DecodeRDCleanPath(derResp)
	if err != nil {
		t.Fatalf("Decode response: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("gateway returned error code: %d", resp.Error.ErrorCode)
	}
	if len(resp.ServerCertChain) == 0 {
		t.Fatal("expected server cert chain in response")
	}
	if !bytes.Equal(resp.ServerCertChain[0], cert.Certificate[0]) {
		t.Fatal("server cert chain mismatch")
	}

	// Bidirectional byte relay test: send payload and receive echo
	testPayload := []byte("hello rdp over cleanpath gateway")
	if err := wsConn.Write(ctx, websocket.MessageBinary, testPayload); err != nil {
		t.Fatalf("write payload: %v", err)
	}

	_, echoResp, err := wsConn.Read(ctx)
	if err != nil {
		t.Fatalf("read echo: %v", err)
	}
	if !bytes.Equal(echoResp, testPayload) {
		t.Errorf("echo mismatch: got %q, want %q", echoResp, testPayload)
	}
}

func TestGatewayTokenSecurity(t *testing.T) {
	cert := generateTestCertificate(t)
	serverAddr, stopServer := startFakeRdpServer(t, cert)
	defer stopServer()

	gw, _, stopGw := setupGateway(t)
	defer stopGw()

	token, wsURL, err := gw.RegisterSession("sess-sec", serverAddr, 0)
	if err != nil {
		t.Fatalf("RegisterSession: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Wrong token rejected
	_, resp, err := websocket.Dial(ctx, wsURL+"?token=wrong-token", &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": []string{"wails://localhost"}},
	})
	if err == nil || (resp != nil && resp.StatusCode != http.StatusForbidden) {
		t.Errorf("expected 403 Forbidden for wrong token, got status %v, err %v", resp, err)
	}

	// 2. Wrong Origin rejected
	_, resp, err = websocket.Dial(ctx, wsURL+"?token="+token, &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": []string{"http://evil.com"}},
	})
	if err == nil || (resp != nil && resp.StatusCode != http.StatusForbidden) {
		t.Errorf("expected 403 Forbidden for bad origin, got status %v, err %v", resp, err)
	}

	// 3. Valid token used once succeeds
	c, _, err := websocket.Dial(ctx, wsURL+"?token="+token, &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": []string{"wails://localhost"}},
	})
	if err != nil {
		t.Fatalf("valid connection failed: %v", err)
	}
	c.Close(websocket.StatusNormalClosure, "done")

	// 4. Token cannot be reused (single-use)
	_, resp, err = websocket.Dial(ctx, wsURL+"?token="+token, &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": []string{"wails://localhost"}},
	})
	if err == nil || (resp != nil && resp.StatusCode != http.StatusForbidden) {
		t.Errorf("expected 403 Forbidden when reusing single-use token, got status %v, err: %v", resp, err)
	}
}

func TestGatewaySSRFProtection(t *testing.T) {
	cert := generateTestCertificate(t)
	serverAddr, stopServer := startFakeRdpServer(t, cert)
	defer stopServer()

	gw, _, stopGw := setupGateway(t)
	defer stopGw()

	token, wsURL, err := gw.RegisterSession("sess-ssrf", serverAddr, 0)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsConn, _, err := websocket.Dial(ctx, wsURL+"?token="+token, &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": []string{"wails://localhost"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer wsConn.Close(websocket.StatusNormalClosure, "done")

	// Client sends a request specifying a DIFFERENT destination than registered
	tamperedReq := RDCleanPathPdu{
		Version:           RDCleanPathVersion1,
		Destination:       "10.255.255.1:3389", // attacker trying SSRF
		ProxyAuth:         token,
		X224ConnectionPdu: []byte{0x03, 0x00, 0x00, 0x0b, 0x06, 0xe0, 0x00, 0x00, 0x00, 0x00, 0x00},
	}
	der, _ := EncodeRDCleanPath(tamperedReq)
	_ = wsConn.Write(ctx, websocket.MessageBinary, der)

	// Gateway must fail closed
	_, derResp, err := wsConn.Read(ctx)
	if err == nil {
		resp, _ := DecodeRDCleanPath(derResp)
		if resp.Error == nil {
			t.Fatal("expected gateway error response for SSRF destination mismatch, got success")
		}
	}
}

func TestGatewayCertEventAndTrust(t *testing.T) {
	cert := generateTestCertificate(t)
	serverAddr, stopServer := startFakeRdpServer(t, cert)
	defer stopServer()

	storePath := filepath.Join(t.TempDir(), "rdp_hosts.toml")
	certStore, _ := LoadRdpCertStore(storePath)

	certEventCh := make(chan RdpCertEvent, 1)
	gw, err := NewRdpGateway(certStore, func(ev RdpCertEvent) {
		certEventCh <- ev
	})
	if err != nil {
		t.Fatal(err)
	}
	defer gw.Close()

	token, wsURL, err := gw.RegisterSession("sess-cert", serverAddr, 0)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsConn, _, err := websocket.Dial(ctx, wsURL+"?token="+token, &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": []string{"wails://localhost"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer wsConn.Close(websocket.StatusNormalClosure, "done")

	crPdu := RDCleanPathPdu{
		Version:           RDCleanPathVersion1,
		Destination:       serverAddr,
		ProxyAuth:         token,
		X224ConnectionPdu: []byte{0x03, 0x00, 0x00, 0x0b, 0x06, 0xe0, 0x00, 0x00, 0x00, 0x00, 0x00},
	}
	derReq, _ := EncodeRDCleanPath(crPdu)
	_ = wsConn.Write(ctx, websocket.MessageBinary, derReq)

	// Verify cert event was fired
	select {
	case ev := <-certEventCh:
		if ev.SessionID != "sess-cert" || ev.Endpoint != serverAddr {
			t.Errorf("cert event mismatch: %+v", ev)
		}
		expectedFp := FingerprintSHA256(cert.Certificate[0])
		if ev.Fingerprint != expectedFp {
			t.Errorf("cert event fp = %q, want %q", ev.Fingerprint, expectedFp)
		}
		// Trust the cert
		if !gw.TrustCert("sess-cert", true) {
			t.Fatal("TrustCert returned false")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for cert event")
	}

	// Now read gateway response
	_, derResp, err := wsConn.Read(ctx)
	if err != nil {
		t.Fatalf("reading response after trust: %v", err)
	}
	resp, err := DecodeRDCleanPath(derResp)
	if err != nil || resp.Error != nil {
		t.Fatalf("unexpected error response after trust: %+v", resp)
	}

	// Verify fingerprint is now persisted in certStore
	dec, _ := certStore.Check(serverAddr, FingerprintSHA256(cert.Certificate[0]))
	if dec != CertTrust {
		t.Errorf("certStore after trust = %v, want CertTrust", dec)
	}
}

func TestGatewayLargeTransferBackpressure(t *testing.T) {
	cert := generateTestCertificate(t)
	serverAddr, stopServer := startFakeRdpServer(t, cert)
	defer stopServer()

	gw, certStore, stopGw := setupGateway(t)
	defer stopGw()

	fp := FingerprintSHA256(cert.Certificate[0])
	_ = certStore.Trust(serverAddr, fp)

	token, wsURL, _ := gw.RegisterSession("sess-large", serverAddr, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	wsConn, _, err := websocket.Dial(ctx, wsURL+"?token="+token, &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": []string{"wails://localhost"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer wsConn.Close(websocket.StatusNormalClosure, "done")

	crPdu := RDCleanPathPdu{
		Version:           RDCleanPathVersion1,
		Destination:       serverAddr,
		ProxyAuth:         token,
		X224ConnectionPdu: []byte{0x03, 0x00, 0x00, 0x0b, 0x06, 0xe0, 0x00, 0x00, 0x00, 0x00, 0x00},
	}
	derReq, _ := EncodeRDCleanPath(crPdu)
	_ = wsConn.Write(ctx, websocket.MessageBinary, derReq)

	_, _, err = wsConn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Send 1 MiB in 32 KiB chunks and verify each echo chunk
	chunk := make([]byte, 32*1024)
	for i := range chunk {
		chunk[i] = byte(i % 256)
	}

	for range 32 { // 32 * 32 KiB = 1 MiB
		if err := wsConn.Write(ctx, websocket.MessageBinary, chunk); err != nil {
			t.Fatalf("write chunk: %v", err)
		}
		var echo []byte
		for len(echo) < len(chunk) {
			_, msg, err := wsConn.Read(ctx)
			if err != nil {
				t.Fatalf("read echo chunk: %v", err)
			}
			echo = append(echo, msg...)
		}
		if !bytes.Equal(echo, chunk) {
			t.Fatal("large chunk mismatch")
		}
	}
}

func TestGatewayDisconnects(t *testing.T) {
	t.Run("client disconnect closes server connection", func(t *testing.T) {
		cert := generateTestCertificate(t)
		serverAddr, stopServer := startFakeRdpServer(t, cert)
		defer stopServer()

		gw, certStore, stopGw := setupGateway(t)
		defer stopGw()

		fp := FingerprintSHA256(cert.Certificate[0])
		_ = certStore.Trust(serverAddr, fp)

		token, wsURL, _ := gw.RegisterSession("sess-disc-client", serverAddr, 0)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		wsConn, _, err := websocket.Dial(ctx, wsURL+"?token="+token, &websocket.DialOptions{
			HTTPHeader: http.Header{"Origin": []string{"wails://localhost"}},
		})
		if err != nil {
			t.Fatal(err)
		}

		crPdu := RDCleanPathPdu{
			Version:           RDCleanPathVersion1,
			Destination:       serverAddr,
			ProxyAuth:         token,
			X224ConnectionPdu: []byte{0x03, 0x00, 0x00, 0x0b, 0x06, 0xe0, 0x00, 0x00, 0x00, 0x00, 0x00},
		}
		derReq, _ := EncodeRDCleanPath(crPdu)
		_ = wsConn.Write(ctx, websocket.MessageBinary, derReq)
		_, _, _ = wsConn.Read(ctx)

		// Close client
		wsConn.Close(websocket.StatusNormalClosure, "client closing")
		// Give gateway a brief moment to finish relay goroutines
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("server disconnect closes client connection", func(t *testing.T) {
		cert := generateTestCertificate(t)
		// Start a fake server that closes after handshake
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer ln.Close()

		go func() {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			defer c.Close()
			header := make([]byte, 4)
			_, _ = io.ReadFull(c, header)
			pktLen := int(header[2])<<8 | int(header[3])
			if pktLen > 4 {
				p := make([]byte, pktLen-4)
				_, _ = io.ReadFull(c, p)
			}
			x224CC := []byte{0x03, 0x00, 0x00, 0x0b, 0x06, 0xd0, 0x00, 0x00, 0x12, 0x34, 0x00}
			_, _ = c.Write(x224CC)
			tlsConn := tls.Server(c, &tls.Config{Certificates: []tls.Certificate{cert}})
			_ = tlsConn.Handshake()
			// Immediately close server side
			tlsConn.Close()
		}()

		gw, certStore, stopGw := setupGateway(t)
		defer stopGw()

		serverAddr := ln.Addr().String()
		fp := FingerprintSHA256(cert.Certificate[0])
		_ = certStore.Trust(serverAddr, fp)

		token, wsURL, _ := gw.RegisterSession("sess-disc-srv", serverAddr, 0)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		wsConn, _, err := websocket.Dial(ctx, wsURL+"?token="+token, &websocket.DialOptions{
			HTTPHeader: http.Header{"Origin": []string{"wails://localhost"}},
		})
		if err != nil {
			t.Fatal(err)
		}
		defer wsConn.Close(websocket.StatusNormalClosure, "done")

		crPdu := RDCleanPathPdu{
			Version:           RDCleanPathVersion1,
			Destination:       serverAddr,
			ProxyAuth:         token,
			X224ConnectionPdu: []byte{0x03, 0x00, 0x00, 0x0b, 0x06, 0xe0, 0x00, 0x00, 0x00, 0x00, 0x00},
		}
		derReq, _ := EncodeRDCleanPath(crPdu)
		_ = wsConn.Write(ctx, websocket.MessageBinary, derReq)
		_, _, _ = wsConn.Read(ctx)

		// Read should now fail because server disconnected
		_, _, err = wsConn.Read(ctx)
		if err == nil {
			t.Fatal("expected read to fail after server disconnect, got nil error")
		}
	})
}

// A refused socket used to be silent: the client only reports "WebSocket is Closed". The gateway now logs why,
// without ever writing a token to the log.
func TestGatewayLogsWhyItRefusedAConnection(t *testing.T) {
	var logged bytes.Buffer
	log.SetOutput(&logged)
	defer log.SetOutput(os.Stderr)

	gw, _, stopGw := setupGateway(t)
	defer stopGw()
	token, wsURL, err := gw.RegisterSession("sess-log", "127.0.0.1:1", 0)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dial := func(q, origin string) {
		websocket.Dial(ctx, wsURL+q, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": []string{origin}}})
	}

	dial("?token="+token, "null")
	dial("", "wails://localhost")
	dial("?token=not-a-token", "wails://localhost")
	gw.UnregisterSession("sess-log")
	dial("?token="+token, "wails://localhost")

	out := logged.String()
	for _, want := range []string{`origin "null" is not allowed`, "no token", "token not recognised", "token not recognised"} {
		if !strings.Contains(out, want) {
			t.Errorf("log missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, token) || strings.Contains(out, "not-a-token") {
		t.Errorf("a token was written to the log:\n%s", out)
	}
}

func TestSelectedProtocolDescribesWhatTheServerAgreedTo(t *testing.T) {
	cc := func(neg ...byte) []byte {
		base := []byte{0x03, 0x00, 0x00, byte(11 + len(neg)), byte(6 + len(neg)), 0xd0, 0, 0, 0x12, 0x34, 0}
		return append(base, neg...)
	}
	cases := []struct {
		name string
		x224 []byte
		want string
	}{
		{"no negotiation block means standard RDP security", cc(), "standard RDP security (no TLS)"},
		{"TLS", cc(0x02, 0, 8, 0, 1, 0, 0, 0), "TLS"},
		{"NLA", cc(0x02, 0, 8, 0, 2, 0, 0, 0), "NLA (CredSSP)"},
		{"NLA with early user authorization", cc(0x02, 0, 8, 0, 8, 0, 0, 0), "NLA with early authorization (CredSSP)"},
		{"the server refused the negotiation", cc(0x03, 0, 8, 0, 5, 0, 0, 0), "negotiation failure, code 5"},
		{"too short to read", []byte{3, 0, 0}, "unreadable"},
	}
	for _, c := range cases {
		if got := selectedProtocol(c.x224); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

// The gateway logs what the server agreed to and how a session ended (who closed it, and how many bytes went
// each way), never any content, so a connection that ends mid-handshake can be told from the log.
func TestGatewayLogsTheSessionLifecycle(t *testing.T) {
	var logged bytes.Buffer
	log.SetOutput(&logged)
	defer log.SetOutput(os.Stderr)

	cert := generateTestCertificate(t)
	serverAddr, stopServer := startFakeRdpServer(t, cert)
	defer stopServer()
	gw, store, stopGw := setupGateway(t)
	defer stopGw()
	if err := store.Trust(serverAddr, FingerprintSHA256(cert.Certificate[0])); err != nil { // known host: no prompt
		t.Fatal(err)
	}

	token, wsURL, err := gw.RegisterSession("sess-life", serverAddr, 0)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, wsURL+"?token="+token, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": []string{"wails://localhost"}}})
	if err != nil {
		t.Fatal(err)
	}
	req, _ := EncodeRDCleanPath(RDCleanPathPdu{Destination: serverAddr, ProxyAuth: token, X224ConnectionPdu: []byte{3, 0, 0, 11, 6, 0xe0, 0, 0, 0, 0, 0}})
	c.Write(ctx, websocket.MessageBinary, req)
	if _, _, err := c.Read(ctx); err != nil { // the RDCleanPath response
		t.Fatal(err)
	}
	c.Write(ctx, websocket.MessageBinary, []byte("hello"))
	c.Read(ctx) // the echo
	c.Close(websocket.StatusNormalClosure, "bye")
	time.Sleep(300 * time.Millisecond)

	out := logged.String()
	for _, want := range []string{"server selected standard RDP security", "TLS established", "session ended", "client closed", "5 bytes to the server"} {
		if !strings.Contains(out, want) {
			t.Errorf("log missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "hello") || strings.Contains(out, token) {
		t.Errorf("content or token leaked into the log:\n%s", out)
	}
}
