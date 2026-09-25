package calendar

import (
	"context"
	"errors"
	"time"

	"hirescope/backend/internal/model"
)

var (
	ErrProviderNotConfigured = errors.New("PROVIDER_NOT_CONFIGURED")
	ErrAuthExpired           = errors.New("AUTH_EXPIRED")
	ErrPermissionDenied      = errors.New("PERMISSION_DENIED")
	ErrRateLimited           = errors.New("RATE_LIMITED")
	ErrNotFound              = errors.New("NOT_FOUND")
	ErrProviderUnavailable   = errors.New("PROVIDER_UNAVAILABLE")
	ErrInvalidEvent          = errors.New("INVALID_EVENT")
	ErrUnknownProviderError  = errors.New("UNKNOWN_PROVIDER_ERROR")
)

// CalendarAttendee represents an attendee with email and optional name.
type CalendarAttendee struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// CalendarEventData represents universal event data mapped to external calendar formats.
type CalendarEventData struct {
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Start       time.Time          `json:"start"`
	End         time.Time          `json:"end"`
	Timezone    string             `json:"timezone"`
	Location    string             `json:"location"`
	MeetingURL  string             `json:"meeting_url"`
	Attendees   []CalendarAttendee `json:"attendees"`
}

// TokenResponse represents OAuth token response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiresIn    int    `json:"expires_in"` // seconds
	TokenType    string `json:"token_type"`
	Scope        string `json:"scope"`
}

// UserInfo represents identity information fetched from the provider.
type UserInfo struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

// CalendarProvider is the generic abstraction for external calendar integrations.
type CalendarProvider interface {
	Name() model.CalendarProviderType
	IsConfigured() bool
	GetAuthURL(state string) string
	ExchangeCode(ctx context.Context, code string) (*TokenResponse, *UserInfo, error)
	RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error)
	CreateEvent(ctx context.Context, accessToken string, event *CalendarEventData) (externalEventID string, err error)
	UpdateEvent(ctx context.Context, accessToken string, externalEventID string, event *CalendarEventData) error
	DeleteEvent(ctx context.Context, accessToken string, externalEventID string) error
}
