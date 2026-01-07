package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"sync"
	"time"
)

type oauthProvider string

const (
	oauthProviderDrive    oauthProvider = "drive"
	oauthProviderOneDrive oauthProvider = "onedrive"
)

const oauthFlowTTL = 10 * time.Minute

type oauthFlow struct {
	ID            string
	Provider      oauthProvider
	SessionID     string
	CreatedAt     time.Time
	ExpiresAt     time.Time
	ClientID      string
	ClientSecret  string
	Scope         string
	Region        string
	CodeVerifier  string
	CodeChallenge string

	Completed       bool
	CompletedAt     time.Time
	ResultTokenJSON string
	ResultError     string
}

type oauthFlowStore struct {
	mu          sync.Mutex
	lastCleanup time.Time
	flows       map[string]oauthFlow
}

func newOAuthFlowStore() *oauthFlowStore {
	return &oauthFlowStore{flows: make(map[string]oauthFlow)}
}

func (s *oauthFlowStore) put(flow oauthFlow) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.maybeCleanupLocked(time.Now())
	s.flows[flow.ID] = flow
}

func (s *oauthFlowStore) get(id string) (oauthFlow, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.maybeCleanupLocked(time.Now())

	flow, ok := s.flows[id]
	if !ok {
		return oauthFlow{}, false
	}

	if !flow.ExpiresAt.IsZero() && time.Now().After(flow.ExpiresAt) {
		delete(s.flows, id)
		return oauthFlow{}, false
	}

	return flow, true
}

func (s *oauthFlowStore) delete(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.maybeCleanupLocked(time.Now())
	delete(s.flows, id)
}

func (s *oauthFlowStore) setResult(id string, tokenJSON string, resultErr string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.maybeCleanupLocked(time.Now())

	flow, ok := s.flows[id]
	if !ok {
		return
	}
	if flow.Completed {
		return
	}

	now := time.Now()
	flow.Completed = true
	flow.CompletedAt = now
	flow.ResultTokenJSON = tokenJSON
	flow.ResultError = resultErr
	s.flows[id] = flow
}

func (s *oauthFlowStore) maybeCleanupLocked(now time.Time) {
	if s.lastCleanup.IsZero() || now.Sub(s.lastCleanup) > time.Minute {
		s.cleanupLocked(now)
	}
}

func (s *oauthFlowStore) cleanupLocked(now time.Time) {
	for id, flow := range s.flows {
		if !flow.ExpiresAt.IsZero() && now.After(flow.ExpiresAt) {
			delete(s.flows, id)
		}
	}
	s.lastCleanup = now
}

func newPKCE() (verifier string, challenge string, err error) {
	// RFC 7636: code_verifier length is 43-128 characters.
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}

	verifier = base64.RawURLEncoding.EncodeToString(buf)

	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])

	return verifier, challenge, nil
}
