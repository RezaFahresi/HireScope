package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CalendarProviderType represents supported third-party calendar providers.
type CalendarProviderType string

const (
	CalendarProviderGoogle    CalendarProviderType = "GOOGLE"
	CalendarProviderMicrosoft CalendarProviderType = "MICROSOFT"
)

// CalendarConnectionStatus represents the authorization lifecycle status of a connection.
type CalendarConnectionStatus string

const (
	CalendarConnectionStatusConnected     CalendarConnectionStatus = "CONNECTED"
	CalendarConnectionStatusExpired       CalendarConnectionStatus = "EXPIRED"
	CalendarConnectionStatusRevoked       CalendarConnectionStatus = "REVOKED"
	CalendarConnectionStatusError         CalendarConnectionStatus = "ERROR"
	CalendarConnectionStatusDisconnected  CalendarConnectionStatus = "DISCONNECTED"
	CalendarConnectionStatusNotConfigured CalendarConnectionStatus = "NOT_CONFIGURED"
)

// CalendarSyncStatus represents synchronization status of a specific interview event.
type CalendarSyncStatus string

const (
	CalendarSyncStatusPending CalendarSyncStatus = "PENDING"
	CalendarSyncStatusSynced  CalendarSyncStatus = "SYNCED"
	CalendarSyncStatusFailed  CalendarSyncStatus = "FAILED"
	CalendarSyncStatusDeleted CalendarSyncStatus = "DELETED"
	CalendarSyncStatusSkipped CalendarSyncStatus = "SKIPPED"
)

// CalendarConnection stores user-scoped OAuth connection credentials and metadata.
type CalendarConnection struct {
	ID                   string                   `gorm:"type:uuid;primaryKey" json:"id"`
	UserID               string                   `gorm:"type:uuid;not null;uniqueIndex:idx_user_provider" json:"user_id"`
	Provider             CalendarProviderType     `gorm:"type:varchar(50);not null;uniqueIndex:idx_user_provider" json:"provider"`
	ProviderAccountID    string                   `gorm:"type:varchar(255)" json:"provider_account_id"`
	ProviderEmail        string                   `gorm:"type:varchar(255)" json:"provider_email"`
	EncryptedAccessToken string                   `gorm:"type:text;not null" json:"-"`
	EncryptedRefreshToken string                  `gorm:"type:text" json:"-"`
	TokenExpiresAt       *time.Time               `gorm:"type:timestamptz" json:"token_expires_at,omitempty"`
	Scopes               string                   `gorm:"type:text" json:"scopes"`
	Status               CalendarConnectionStatus `gorm:"type:varchar(50);not null;default:'CONNECTED';index" json:"status"`
	CreatedAt            time.Time                `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt            time.Time                `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"updated_at"`

	User *User `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
}

// TableName returns the table name for CalendarConnection.
func (CalendarConnection) TableName() string {
	return "calendar_connections"
}

// BeforeCreate assigns a UUID if not provided.
func (c *CalendarConnection) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	return nil
}

// CalendarEvent maps a HireScope interview to an external provider calendar event.
type CalendarEvent struct {
	ID                   string               `gorm:"type:uuid;primaryKey" json:"id"`
	InterviewID          string               `gorm:"type:uuid;not null;uniqueIndex:idx_conn_interview" json:"interview_id"`
	CalendarConnectionID string               `gorm:"type:uuid;not null;uniqueIndex:idx_conn_interview" json:"calendar_connection_id"`
	Provider             CalendarProviderType `gorm:"type:varchar(50);not null;index" json:"provider"`
	ExternalEventID      string               `gorm:"type:varchar(500);not null" json:"external_event_id"`
	ExternalCalendarID   *string              `gorm:"type:varchar(255)" json:"external_calendar_id,omitempty"`
	SyncStatus           CalendarSyncStatus   `gorm:"type:varchar(50);not null;default:'PENDING';index" json:"sync_status"`
	LastSyncedAt         *time.Time           `gorm:"type:timestamptz" json:"last_synced_at,omitempty"`
	LastError            *string              `gorm:"type:text" json:"last_error,omitempty"`
	CreatedAt            time.Time            `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt            time.Time            `gorm:"type:timestamptz;not null;default:CURRENT_TIMESTAMP" json:"updated_at"`

	Interview          *Interview          `gorm:"foreignKey:InterviewID;references:ID" json:"interview,omitempty"`
	CalendarConnection *CalendarConnection `gorm:"foreignKey:CalendarConnectionID;references:ID" json:"calendar_connection,omitempty"`
}

// TableName returns the table name for CalendarEvent.
func (CalendarEvent) TableName() string {
	return "calendar_events"
}

// BeforeCreate assigns a UUID if not provided.
func (e *CalendarEvent) BeforeCreate(tx *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	return nil
}
