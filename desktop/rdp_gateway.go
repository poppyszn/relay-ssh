package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

// RdpCertEvent is emitted when an unknown or changed TLS certificate is encountered.
type RdpCertEvent struct {
	SessionID         string `json:"sessionId"`
	Endpoint          string `json:"endpoint"`
	Fingerprint       string `json:"fingerprint"`
	StoredFingerprint string `json:"storedFingerprint"`
	IsMismatch        bool   `json:"isMismatch"`
}

type rdpSessionRegistration struct {
	SessionID   string
	Destination string
	TunnelPort  int
	Token       string
	ExpiresAt   time.Time
	Used        bool

	mu      sync.Mutex
	certCh  chan bool
	pending bool
}

// RdpGateway provides the loopback gateway bridging the WebAssembly IronRDP webview client
// to target RDP hosts over TCP+TLS, speaking the RDCleanPath handshake protocol.
type RdpGateway struct {
	listener  net.Listener
	server    *http.Server
	certStore *RdpCertStore
	onCert    func(RdpCertEvent)

	mu       sync.Mutex
	sessions map[string]*rdpSessionRegistration
	tokens   map[string]*rdpSessionRegistration
	closed   bool
}

// NewRdpGateway creates and starts a loopback gateway on 127.0.0.1:0.
func NewRdpGateway(certStore *RdpCertStore, onCert func(RdpCertEvent)) (*RdpGateway, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("starting rdp gateway listener: %w", err)
	}

	gw := &RdpGateway{
		listener:  ln,
		certStore: certStore,
		onCert:    onCert,
		sessions:  make(map[string]*rdpSessionRegistration),
		tokens:    make(map[string]*rdpSessionRegistration),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/rdp", gw.handle)

	gw.server = &http.Server{
		Handler: mux,
	}

	go func() {
		_ = gw.server.Serve(ln)
	}()

	return gw, nil
}

// rdpTokenLifetime is how long a session's single-use token stays valid. It has to outlast a slow first start
// (loading the client, the certificate prompt, typing a password); the token still works only once, from the
// app's own origin, on loopback, so a longer life adds little. It was 60s, and first connections ran out of it.
var rdpTokenLifetime = 5 * time.Minute

// RegisterSession registers an allowed RDP destination and generates a single-use token (see rdpTokenLifetime).
func (g *RdpGateway) RegisterSession(sessionID, dest string, tunnelPort int) (string, string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	tokBytes := make([]byte, 24)
	if _, err := rand.Read(tokBytes); err != nil {
		return "", "", fmt.Errorf("generating token: %w", err)
	}
	token := hex.EncodeToString(tokBytes)

	reg := &rdpSessionRegistration{
		SessionID:   sessionID,
		Destination: dest,
		TunnelPort:  tunnelPort,
		Token:       token,
		ExpiresAt:   time.Now().Add(rdpTokenLifetime),
		certCh:      make(chan bool, 1),
	}

	g.sessions[sessionID] = reg
	g.tokens[token] = reg

	wsURL := "ws://" + g.listener.Addr().String() + "/rdp"
	return token, wsURL, nil
}

// TrustCert delivers the user's accept/reject decision for a pending certificate prompt.
func (g *RdpGateway) TrustCert(sessionID string, accept bool) bool {
	g.mu.Lock()
	reg, exists := g.sessions[sessionID]
	g.mu.Unlock()

	if !exists {
		return false
	}

	reg.mu.Lock()
	defer reg.mu.Unlock()

	if !reg.pending {
		return false
	}

	reg.pending = false
	select {
	case reg.certCh <- accept:
		return true
	default:
		return false
	}
}

// UnregisterSession removes a session and its token from the gateway.
func (g *RdpGateway) UnregisterSession(sessionID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	if reg, exists := g.sessions[sessionID]; exists {
		delete(g.sessions, sessionID)
		delete(g.tokens, reg.Token)
	}
}

// Close shuts down the gateway server and listener.
func (g *RdpGateway) Close() error {
	g.mu.Lock()
	g.closed = true
	g.mu.Unlock()

	return g.server.Close()
}

// rdpHandshakeTimeout bounds each step before the byte relay starts (the server's negotiation reply, the TLS
// handshake). A server that accepts the connection and then goes quiet fails fast instead of hanging the tab.
var rdpHandshakeTimeout = 20 * time.Second

// stepError names the handshake step that failed; a timeout reads as "timed out" rather than "i/o timeout".
func stepError(step string, err error) error {
	var ne net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &ne) && ne.Timeout()) {
		return fmt.Errorf("%s: timed out after %s", step, rdpHandshakeTimeout)
	}
	return fmt.Errorf("%s: %w", step, err)
}

// rdpClientTLSConfig is how the gateway speaks TLS to an RDP host. Capped at TLS 1.2: some Windows servers drop the
// session ("internal error") when the NLA login starts over TLS 1.3, and RDP clients are conservative about it.
// The chain is not verified: the host is trusted by its certificate's fingerprint (see RdpCertStore).
func rdpClientTLSConfig() *tls.Config {
	return &tls.Config{
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS12,
		InsecureSkipVerify: true,
	}
}

// selectedProtocol describes the security the server chose in its X.224 Connection Confirm, for the log.
func selectedProtocol(x224 []byte) string {
	if len(x224) < 11 {
		return "unreadable"
	}
	neg := x224[11:]
	if len(neg) < 8 {
		return "standard RDP security (no TLS)"
	}
	val := uint32(neg[4]) | uint32(neg[5])<<8 | uint32(neg[6])<<16 | uint32(neg[7])<<24
	if neg[0] == 0x03 {
		return fmt.Sprintf("negotiation failure, code %d", val)
	}
	switch val {
	case 0:
		return "standard RDP security (no TLS)"
	case 1:
		return "TLS"
	case 2:
		return "NLA (CredSSP)"
	case 4:
		return "RDSTLS"
	case 8:
		return "NLA with early authorization (CredSSP)"
	}
	return fmt.Sprintf("unknown protocol %d", val)
}

// refuse answers 403 and logs why. The client only ever sees "WebSocket is Closed", so this is where a
// refused connection can be told apart. A token is never logged.
func refuse(w http.ResponseWriter, why string) {
	log.Printf("rdp gateway: refused a connection: %s", why)
	http.Error(w, why, http.StatusForbidden)
}

func (g *RdpGateway) handle(w http.ResponseWriter, r *http.Request) {
	// Origin allow-list check
	origin := r.Header.Get("Origin")
	if !allowedOrigin(origin) {
		refuse(w, fmt.Sprintf("origin %q is not allowed", origin))
		return
	}

	// Single-use token check
	token := r.URL.Query().Get("token")
	if token == "" {
		refuse(w, "no token")
		return
	}

	g.mu.Lock()
	reg, exists := g.tokens[token]
	switch {
	case !exists:
		g.mu.Unlock()
		refuse(w, "token not recognised (already used, or its session was closed)")
		return
	case reg.Used || time.Now().After(reg.ExpiresAt):
		delete(g.tokens, token)
		g.mu.Unlock()
		refuse(w, "token already used or expired")
		return
	}
	reg.Used = true
	delete(g.tokens, token)
	g.mu.Unlock()

	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	defer c.Close(websocket.StatusInternalError, "gateway error")
	// The library default (32 KiB) would end a session the first time the client sent a bigger RDP message; a
	// TPKT is at most 64 KiB. Still bounded, so a misbehaving page cannot make the gateway buffer without limit.
	c.SetReadLimit(128 << 10)

	started := time.Now()
	if err := g.handshakeAndRelay(r.Context(), c, reg); err != nil {
		log.Printf("rdp gateway: %s: handshake failed after %s: %v", reg.Destination, time.Since(started).Round(time.Millisecond), err)
	}
}

func (g *RdpGateway) handshakeAndRelay(ctx context.Context, c *websocket.Conn, reg *rdpSessionRegistration) error {
	msgType, reqBytes, err := c.Read(ctx)
	if err != nil {
		return err
	}
	if msgType != websocket.MessageBinary {
		return fmt.Errorf("expected binary message")
	}

	pdu, err := DecodeRDCleanPath(reqBytes)
	if err != nil {
		return err
	}

	// SSRF check: destination must match registration
	if pdu.Destination != "" && pdu.Destination != reg.Destination {
		errPdu := NewRDCleanPathErrorPdu(TargetUnreachableErrorCode)
		b, _ := EncodeRDCleanPath(errPdu)
		_ = c.Write(ctx, websocket.MessageBinary, b)
		return fmt.Errorf("ssrf destination mismatch: %q != %q", pdu.Destination, reg.Destination)
	}

	target := reg.Destination
	if reg.TunnelPort > 0 {
		target = fmt.Sprintf("127.0.0.1:%d", reg.TunnelPort)
	}

	// failWith tells the client why (an RDCleanPath error) before the connection closes, so it fails at once
	// with a reason instead of a bare "closed" or "not enough bytes".
	failWith := func(code int, err error) error {
		b, _ := EncodeRDCleanPath(NewRDCleanPathErrorPdu(code))
		_ = c.Write(ctx, websocket.MessageBinary, b)
		return err
	}

	tcpConn, err := net.DialTimeout("tcp", target, 10*time.Second)
	if err != nil {
		return failWith(TargetUnreachableErrorCode, stepError("could not connect to "+reg.Destination, err))
	}
	defer tcpConn.Close()

	_ = tcpConn.SetDeadline(time.Now().Add(rdpHandshakeTimeout))
	if _, err := tcpConn.Write(pdu.X224ConnectionPdu); err != nil {
		return failWith(GeneralErrorCode, stepError("sending the negotiation request", err))
	}

	hdr := make([]byte, 4)
	if _, err := io.ReadFull(tcpConn, hdr); err != nil {
		return failWith(GeneralErrorCode, stepError("waiting for the server's negotiation reply", err))
	}
	totalLen := int(hdr[2])<<8 | int(hdr[3])
	if totalLen < 4 || totalLen > 65535 {
		return failWith(GeneralErrorCode, fmt.Errorf("invalid x224 tpkt length: %d", totalLen))
	}

	serverX224 := make([]byte, totalLen)
	copy(serverX224[:4], hdr)
	if _, err := io.ReadFull(tcpConn, serverX224[4:]); err != nil {
		return failWith(GeneralErrorCode, stepError("reading the server's negotiation reply", err))
	}

	log.Printf("rdp gateway: %s: server selected %s", reg.Destination, selectedProtocol(serverX224))

	tlsCtx, cancelTLS := context.WithTimeout(ctx, rdpHandshakeTimeout)
	defer cancelTLS()
	tlsConn := tls.Client(tcpConn, rdpClientTLSConfig())
	if err := tlsConn.HandshakeContext(tlsCtx); err != nil {
		return failWith(TLSHandshakeFailedErrorCode, stepError("TLS handshake", err))
	}
	_ = tcpConn.SetDeadline(time.Time{}) // the relay below has no deadline
	defer tlsConn.Close()

	state := tlsConn.ConnectionState()
	log.Printf("rdp gateway: %s: TLS established (%s, %s)", reg.Destination, tls.VersionName(state.Version), tls.CipherSuiteName(state.CipherSuite))
	if len(state.PeerCertificates) == 0 {
		return fmt.Errorf("no peer certificates returned by server")
	}
	leaf := state.PeerCertificates[0]
	fp := FingerprintSHA256(leaf.Raw)

	dec, storedFp := g.certStore.Check(reg.Destination, fp)
	if dec != CertTrust {
		reg.mu.Lock()
		reg.pending = true
		reg.mu.Unlock()

		if g.onCert != nil {
			g.onCert(RdpCertEvent{
				SessionID:         reg.SessionID,
				Endpoint:          reg.Destination,
				Fingerprint:       fp,
				StoredFingerprint: storedFp,
				IsMismatch:        dec == CertMismatch,
			})
		}

		log.Printf("rdp gateway: %s: waiting for a decision on the certificate (up to 60s)", reg.Destination)
		select {
		case accept := <-reg.certCh:
			if !accept {
				errPdu := NewRDCleanPathErrorPdu(GeneralErrorCode)
				b, _ := EncodeRDCleanPath(errPdu)
				_ = c.Write(ctx, websocket.MessageBinary, b)
				return fmt.Errorf("certificate rejected by user")
			}
			_ = g.certStore.Trust(reg.Destination, fp)
		case <-time.After(60 * time.Second):
			errPdu := NewRDCleanPathErrorPdu(GeneralErrorCode)
			b, _ := EncodeRDCleanPath(errPdu)
			_ = c.Write(ctx, websocket.MessageBinary, b)
			return fmt.Errorf("certificate prompt timed out")
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	var certChain [][]byte
	for _, crt := range state.PeerCertificates {
		certChain = append(certChain, crt.Raw)
	}

	respPdu := NewRDCleanPathResponsePdu(reg.Destination, serverX224, certChain)
	respBytes, err := EncodeRDCleanPath(respPdu)
	if err != nil {
		return err
	}
	if err := c.Write(ctx, websocket.MessageBinary, respBytes); err != nil {
		return err
	}

	// Bidirectional byte relay with backpressure. Only sizes and who ended it are logged, never content.
	type ended struct {
		who string
		err error
	}
	var toServer, toClient atomic.Int64
	started := time.Now()
	errCh := make(chan ended, 2)
	go func() {
		for {
			mt, data, rerr := c.Read(ctx)
			if rerr != nil {
				errCh <- ended{"client", rerr}
				return
			}
			if mt == websocket.MessageBinary && len(data) > 0 {
				if _, werr := tlsConn.Write(data); werr != nil {
					errCh <- ended{"server", werr}
					return
				}
				toServer.Add(int64(len(data)))
			}
		}
	}()

	go func() {
		buf := make([]byte, 32*1024)
		for {
			n, rerr := tlsConn.Read(buf)
			if n > 0 {
				if werr := c.Write(ctx, websocket.MessageBinary, buf[:n]); werr != nil {
					errCh <- ended{"client", werr}
					return
				}
				toClient.Add(int64(n))
			}
			if rerr != nil {
				errCh <- ended{"server", rerr}
				return
			}
		}
	}()

	first := <-errCh
	tlsConn.Close()
	c.Close(websocket.StatusNormalClosure, "session closed")
	<-errCh
	log.Printf("rdp gateway: %s: session ended after %s, %s closed it (%v); %d bytes to the server, %d bytes to the client",
		reg.Destination, time.Since(started).Round(time.Millisecond), first.who, first.err, toServer.Load(), toClient.Load())
	return nil
}
