package calendar

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"hirescope/backend/internal/model"
)

var (
	ErrInvalidState  = errors.New("invalid or expired OAuth state")
	ErrStateMismatch = errors.New("OAuth state does not match requested provider")
)

// OAuthState holds the context for an ongoing OAuth authorization flow.
type OAuthState struct {
	Token     string
	UserID    string
	Provider  model.CalendarProviderType
	CreatedAt time.Time
	ExpiresAt time.Time
}

// StateManager manages generation and validation of short-lived OAuth states.
type StateManager interface {
	GenerateState(userID string, provider model.CalendarProviderType) (string, error)
	ValidateAndConsumeState(token string, expectedProvider model.CalendarProviderType) (*OAuthState, error)
}

type memoryStateManager struct {
	mu     sync.RWMutex
	states map[string]*OAuthState
	ttl    time.Duration
}

// NewStateManager creates a new StateManager with specified state TTL.
func NewStateManager(ttl time.Duration) StateManager {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	sm := &memoryStateManager{
		states: make(map[string]*OAuthState),
		ttl:    ttl,
	}

	// Periodic cleanup of expired states
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			sm.cleanExpired()
		}
	}()

	return sm
}

func (sm *memoryStateManager) GenerateState(userID string, provider model.CalendarProviderType) (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random state: %w", err)
	}

	token := hex.EncodeToString(bytes)
	now := time.Now().UTC()

	state := &OAuthState{
		Token:     token,
		UserID:    userID,
		Provider:  provider,
		CreatedAt: now,
		ExpiresAt: now.Add(sm.ttl),
	}

	sm.mu.Lock()
	sm.states[token] = state
	sm.mu.Unlock()

	return token, nil
}

func (sm *memoryStateManager) ValidateAndConsumeState(token string, expectedProvider model.CalendarProviderType) (*OAuthState, error) {
	if token == "" {
		return nil, ErrInvalidState
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, exists := sm.states[token]
	if !exists {
		return nil, ErrInvalidState
	}

	// Always delete after first access to ensure one-time usage
	delete(sm.states, token)

	if time.Now().UTC().After(state.ExpiresAt) {
		return nil, ErrInvalidState
	}

	if state.Provider != expectedProvider {
		return nil, ErrStateMismatch
	}

	return state, nil
}

func (sm *memoryStateManager) cleanExpired() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now().UTC()
	for k, v := range sm.states {
		if now.After(v.ExpiresAt) {
			delete(sm.states, k)
		}
	}
}
