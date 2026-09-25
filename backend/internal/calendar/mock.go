package calendar

import (
	"context"
	"fmt"
	"sync"
	"time"

	"hirescope/backend/internal/model"
)

// MockCalendarProvider implements CalendarProvider in memory for automated testing.
type MockCalendarProvider struct {
	mu             sync.Mutex
	providerType   model.CalendarProviderType
	configured     bool
	Events         map[string]*CalendarEventData
	CreateEventErr error
	UpdateEventErr error
	DeleteEventErr error
	RefreshErr     error
	ExchangeErr    error
	NextEventID    string
	LastAction     string
}

// NewMockCalendarProvider constructs a new MockCalendarProvider.
func NewMockCalendarProvider(providerType model.CalendarProviderType) *MockCalendarProvider {
	return &MockCalendarProvider{
		providerType: providerType,
		configured:   true,
		Events:       make(map[string]*CalendarEventData),
		NextEventID:  fmt.Sprintf("mock-event-%s-1", providerType),
	}
}

func (m *MockCalendarProvider) Name() model.CalendarProviderType {
	return m.providerType
}

func (m *MockCalendarProvider) IsConfigured() bool {
	return m.configured
}

func (m *MockCalendarProvider) SetConfigured(val bool) {
	m.configured = val
}

func (m *MockCalendarProvider) GetAuthURL(state string) string {
	if !m.configured {
		return ""
	}
	return fmt.Sprintf("https://mock.auth/%s?state=%s", m.providerType, state)
}

func (m *MockCalendarProvider) ExchangeCode(ctx context.Context, code string) (*TokenResponse, *UserInfo, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.configured {
		return nil, nil, ErrProviderNotConfigured
	}
	if m.ExchangeErr != nil {
		return nil, nil, m.ExchangeErr
	}

	return &TokenResponse{
			AccessToken:  fmt.Sprintf("mock-access-token-%s", m.providerType),
			RefreshToken: fmt.Sprintf("mock-refresh-token-%s", m.providerType),
			ExpiresIn:    3600,
			TokenType:    "Bearer",
		}, &UserInfo{
			ID:    "mock-user-123",
			Email: fmt.Sprintf("recruiter@%s.mock", m.providerType),
			Name:  "Mock Recruiter",
		}, nil
}

func (m *MockCalendarProvider) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.configured {
		return nil, ErrProviderNotConfigured
	}
	if m.RefreshErr != nil {
		return nil, m.RefreshErr
	}

	return &TokenResponse{
		AccessToken:  fmt.Sprintf("mock-refreshed-token-%s", m.providerType),
		RefreshToken: refreshToken,
		ExpiresIn:    3600,
		TokenType:    "Bearer",
	}, nil
}

func (m *MockCalendarProvider) CreateEvent(ctx context.Context, accessToken string, event *CalendarEventData) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.configured {
		return "", ErrProviderNotConfigured
	}
	if m.CreateEventErr != nil {
		return "", m.CreateEventErr
	}

	eventID := m.NextEventID
	if eventID == "" {
		eventID = fmt.Sprintf("mock-evt-%d", time.Now().UnixNano())
	}
	m.Events[eventID] = event
	m.LastAction = "CREATE"

	return eventID, nil
}

func (m *MockCalendarProvider) UpdateEvent(ctx context.Context, accessToken string, externalEventID string, event *CalendarEventData) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.configured {
		return ErrProviderNotConfigured
	}
	if m.UpdateEventErr != nil {
		return m.UpdateEventErr
	}
	if _, ok := m.Events[externalEventID]; !ok {
		return ErrNotFound
	}

	m.Events[externalEventID] = event
	m.LastAction = "UPDATE"
	return nil
}

func (m *MockCalendarProvider) DeleteEvent(ctx context.Context, accessToken string, externalEventID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.configured {
		return ErrProviderNotConfigured
	}
	if m.DeleteEventErr != nil {
		return m.DeleteEventErr
	}

	delete(m.Events, externalEventID)
	m.LastAction = "DELETE"
	return nil
}
