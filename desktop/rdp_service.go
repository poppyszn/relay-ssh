package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/poppyszn/relay-ssh/desktop/vault"
	"github.com/poppyszn/relay-ssh/internal/config"
	"github.com/poppyszn/relay-ssh/internal/paths"
)

// RdpSessionInfo provides the frontend with the loopback gateway connection details
// and remote target information needed to establish an IronRDP session.
// It deliberately contains NO secret.
type RdpSessionInfo struct {
	SessionID        string `json:"sessionId"`
	WSURL            string `json:"wsUrl"`
	Token            string `json:"token"`
	Host             string `json:"host"`
	User             string `json:"user"`
	Domain           string `json:"domain"`
	Port             int    `json:"port"`
	HasSavedPassword bool   `json:"hasSavedPassword"`
}

type rdpActiveSession struct {
	id          string
	connection  string
	profile     config.Profile
	tunnel      *RdpTunnel
	credClaimed bool
	createdAt   time.Time
}

// RdpService coordinates RDP sessions between the desktop UI, the loopback gateway,
// optional SSH jump tunnels, and the credentials vault.
type RdpService struct {
	mu       sync.Mutex
	gateway  *RdpGateway
	sessions map[string]*rdpActiveSession
}

// NewRdpService constructs an RdpService bound to the specified loopback gateway.
func NewRdpService(gateway *RdpGateway) *RdpService {
	return &RdpService{
		gateway:  gateway,
		sessions: make(map[string]*rdpActiveSession),
	}
}

// Open initiates an RDP session for connection. If Jump is set, an SSH tunnel is established.
// The session is registered with the gateway and returns connection metadata with NO secret.
func (s *RdpService) Open(connection string) (RdpSessionInfo, error) {
	p, err := paths.Default()
	if err != nil {
		return RdpSessionInfo{}, err
	}
	store, err := config.Load(p.StoreFile)
	if err != nil {
		return RdpSessionInfo{}, err
	}
	prof, ok := store.Find(connection)
	if !ok {
		return RdpSessionInfo{}, fmt.Errorf("connection %q not found", connection)
	}
	if !prof.IsRDP() {
		return RdpSessionInfo{}, fmt.Errorf("connection %q is not an RDP connection", connection)
	}

	port := prof.Port
	if port == 0 {
		port = 3389
	}

	var tunnel *RdpTunnel
	tunnelPort := 0
	if prof.Jump != "" {
		var err error
		tunnel, err = StartRdpTunnel(p, prof.Jump, prof.Host, port)
		if err != nil {
			return RdpSessionInfo{}, err
		}
		tunnelPort = tunnel.LocalPort
	}

	sessBytes := make([]byte, 16)
	if _, err := rand.Read(sessBytes); err != nil {
		if tunnel != nil {
			_ = tunnel.Close()
		}
		return RdpSessionInfo{}, fmt.Errorf("generating session ID: %w", err)
	}
	sessionID := hex.EncodeToString(sessBytes)

	dest := fmt.Sprintf("%s:%d", prof.Host, port)
	token, wsURL, err := s.gateway.RegisterSession(sessionID, dest, tunnelPort)
	if err != nil {
		if tunnel != nil {
			_ = tunnel.Close()
		}
		return RdpSessionInfo{}, fmt.Errorf("registering session with gateway: %w", err)
	}

	v := getVault()
	hasSavedPassword := v.Status().Unlocked && v.Has(prof.Name)

	s.mu.Lock()
	s.sessions[sessionID] = &rdpActiveSession{
		id:          sessionID,
		connection:  prof.Name,
		profile:     prof,
		tunnel:      tunnel,
		credClaimed: false,
		createdAt:   time.Now(),
	}
	s.mu.Unlock()

	if tunnel != nil {
		go func() {
			<-tunnel.Wait()
			_ = s.Close(sessionID)
		}()
	}

	return RdpSessionInfo{
		SessionID:        sessionID,
		WSURL:            wsURL,
		Token:            token,
		Host:             prof.Host,
		User:             prof.User,
		Domain:           prof.Domain,
		Port:             port,
		HasSavedPassword: hasSavedPassword,
	}, nil
}

// Credentials is a separate, one-shot method that returns the saved password only
// if the vault is unlocked and the stored binding matches the connection now.
// It is the second bound method allowed to return a secret (the first is RevealSecret).
func (s *RdpService) Credentials(sessionID string) (string, error) {
	s.mu.Lock()
	sess, ok := s.sessions[sessionID]
	if !ok {
		s.mu.Unlock()
		return "", errors.New("session not found")
	}
	if sess.credClaimed {
		s.mu.Unlock()
		return "", errors.New("credentials already retrieved for this session")
	}
	sess.credClaimed = true
	prof := sess.profile
	s.mu.Unlock()

	v := getVault()
	if !v.Status().Unlocked {
		return "", vault.ErrVaultLocked
	}

	binding := bindingOf(prof)
	secret, has, err := v.Get(prof.Name, binding)
	if err != nil {
		return "", err
	}
	if !has {
		return "", errors.New("no saved password for connection")
	}

	return secret, nil
}

// TrustCert delivers the user's accept/reject decision for a pending certificate prompt.
func (s *RdpService) TrustCert(sessionID string, accept bool) bool {
	if s.gateway == nil {
		return false
	}
	return s.gateway.TrustCert(sessionID, accept)
}

// Close terminates any active tunnel, unregisters the gateway session, and frees session state.
func (s *RdpService) Close(sessionID string) error {
	s.mu.Lock()
	sess, ok := s.sessions[sessionID]
	if ok {
		delete(s.sessions, sessionID)
	}
	s.mu.Unlock()

	if ok {
		if sess.tunnel != nil {
			_ = sess.tunnel.Close()
		}
		if s.gateway != nil {
			s.gateway.UnregisterSession(sessionID)
		}
	}
	return nil
}

// CloseAll terminates all active RDP sessions and tunnels.
func (s *RdpService) CloseAll() {
	s.mu.Lock()
	sessions := s.sessions
	s.sessions = make(map[string]*rdpActiveSession)
	s.mu.Unlock()

	for _, sess := range sessions {
		if sess.tunnel != nil {
			_ = sess.tunnel.Close()
		}
		if s.gateway != nil {
			s.gateway.UnregisterSession(sess.id)
		}
	}
}
