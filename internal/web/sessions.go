package web

import (
	"crypto/rand"
	"encoding/hex"
	"sync"

	"github.com/yzh44yzh/bookmeet/internal/domain"
)

// sessions is an in-memory session store mapping a random token to a UserID.
// Sessions are lost on restart (see docs/adr/0003-email-login.md).
type sessions struct {
	mu      sync.RWMutex
	byToken map[string]domain.UserID
}

func newSessions() *sessions {
	return &sessions{byToken: make(map[string]domain.UserID)}
}

func (s *sessions) create(id domain.UserID) string {
	var b [32]byte
	_, _ = rand.Read(b[:]) // crypto/rand.Read never fails
	token := hex.EncodeToString(b[:])

	s.mu.Lock()
	defer s.mu.Unlock()
	s.byToken[token] = id
	return token
}

func (s *sessions) get(token string) (domain.UserID, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.byToken[token]
	return id, ok
}
