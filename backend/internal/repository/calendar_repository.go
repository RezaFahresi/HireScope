package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"hirescope/backend/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrConnectionNotFound   = errors.New("calendar connection not found")
	ErrCalendarEventNotFound = errors.New("calendar event not found")
)

// CalendarRepository provides database access for external calendar integrations.
type CalendarRepository interface {
	GetConnection(ctx context.Context, userID string, provider model.CalendarProviderType) (*model.CalendarConnection, error)
	GetConnectionByID(ctx context.Context, id string) (*model.CalendarConnection, error)
	ListConnectionsByUserID(ctx context.Context, userID string) ([]model.CalendarConnection, error)
	UpsertConnection(ctx context.Context, conn *model.CalendarConnection) error
	UpdateConnectionStatus(ctx context.Context, id string, status model.CalendarConnectionStatus) error
	UpdateConnectionTokens(ctx context.Context, id string, encAccessToken string, encRefreshToken string, expiresAt *time.Time, status model.CalendarConnectionStatus) error
	DeleteConnection(ctx context.Context, userID string, provider model.CalendarProviderType) error

	GetCalendarEvent(ctx context.Context, connectionID, interviewID string) (*model.CalendarEvent, error)
	GetCalendarEventByID(ctx context.Context, id string) (*model.CalendarEvent, error)
	ListCalendarEventsByInterviewID(ctx context.Context, interviewID string) ([]model.CalendarEvent, error)
	UpsertCalendarEvent(ctx context.Context, event *model.CalendarEvent) error
	UpdateCalendarEventSyncStatus(ctx context.Context, id string, status model.CalendarSyncStatus, lastSyncedAt *time.Time, lastError *string) error
	DeleteCalendarEvent(ctx context.Context, id string) error
}

type calendarRepository struct {
	db *gorm.DB
}

// NewCalendarRepository creates a new CalendarRepository instance.
func NewCalendarRepository(db *gorm.DB) CalendarRepository {
	return &calendarRepository{db: db}
}

func (r *calendarRepository) GetConnection(ctx context.Context, userID string, provider model.CalendarProviderType) (*model.CalendarConnection, error) {
	var conn model.CalendarConnection
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND provider = ?", userID, provider).
		First(&conn).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrConnectionNotFound
		}
		return nil, fmt.Errorf("failed to get calendar connection: %w", err)
	}
	return &conn, nil
}

func (r *calendarRepository) GetConnectionByID(ctx context.Context, id string) (*model.CalendarConnection, error) {
	var conn model.CalendarConnection
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&conn).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrConnectionNotFound
		}
		return nil, fmt.Errorf("failed to get calendar connection by id: %w", err)
	}
	return &conn, nil
}

func (r *calendarRepository) ListConnectionsByUserID(ctx context.Context, userID string) ([]model.CalendarConnection, error) {
	var conns []model.CalendarConnection
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at ASC").
		Find(&conns).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list calendar connections: %w", err)
	}
	return conns, nil
}

func (r *calendarRepository) UpsertConnection(ctx context.Context, conn *model.CalendarConnection) error {
	// Upsert on unique index (user_id, provider)
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "user_id"}, {Name: "provider"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"provider_account_id",
				"provider_email",
				"encrypted_access_token",
				"encrypted_refresh_token",
				"token_expires_at",
				"scopes",
				"status",
				"updated_at",
			}),
		}).
		Create(conn).Error
	if err != nil {
		return fmt.Errorf("failed to upsert calendar connection: %w", err)
	}
	return nil
}

func (r *calendarRepository) UpdateConnectionStatus(ctx context.Context, id string, status model.CalendarConnectionStatus) error {
	err := r.db.WithContext(ctx).
		Model(&model.CalendarConnection{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now().UTC(),
		}).Error
	if err != nil {
		return fmt.Errorf("failed to update connection status: %w", err)
	}
	return nil
}

func (r *calendarRepository) UpdateConnectionTokens(ctx context.Context, id string, encAccessToken string, encRefreshToken string, expiresAt *time.Time, status model.CalendarConnectionStatus) error {
	updates := map[string]interface{}{
		"encrypted_access_token": encAccessToken,
		"token_expires_at":       expiresAt,
		"status":                 status,
		"updated_at":             time.Now().UTC(),
	}
	if encRefreshToken != "" {
		updates["encrypted_refresh_token"] = encRefreshToken
	}

	err := r.db.WithContext(ctx).
		Model(&model.CalendarConnection{}).
		Where("id = ?", id).
		Updates(updates).Error
	if err != nil {
		return fmt.Errorf("failed to update connection tokens: %w", err)
	}
	return nil
}

func (r *calendarRepository) DeleteConnection(ctx context.Context, userID string, provider model.CalendarProviderType) error {
	// Soft disconnect by setting status = DISCONNECTED and clearing tokens
	err := r.db.WithContext(ctx).
		Model(&model.CalendarConnection{}).
		Where("user_id = ? AND provider = ?", userID, provider).
		Updates(map[string]interface{}{
			"status":                  model.CalendarConnectionStatusDisconnected,
			"encrypted_access_token":  "",
			"encrypted_refresh_token": "",
			"updated_at":              time.Now().UTC(),
		}).Error
	if err != nil {
		return fmt.Errorf("failed to disconnect calendar: %w", err)
	}
	return nil
}

func (r *calendarRepository) GetCalendarEvent(ctx context.Context, connectionID, interviewID string) (*model.CalendarEvent, error) {
	var event model.CalendarEvent
	err := r.db.WithContext(ctx).
		Where("calendar_connection_id = ? AND interview_id = ?", connectionID, interviewID).
		First(&event).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCalendarEventNotFound
		}
		return nil, fmt.Errorf("failed to get calendar event: %w", err)
	}
	return &event, nil
}

func (r *calendarRepository) GetCalendarEventByID(ctx context.Context, id string) (*model.CalendarEvent, error) {
	var event model.CalendarEvent
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&event).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCalendarEventNotFound
		}
		return nil, fmt.Errorf("failed to get calendar event by id: %w", err)
	}
	return &event, nil
}

func (r *calendarRepository) ListCalendarEventsByInterviewID(ctx context.Context, interviewID string) ([]model.CalendarEvent, error) {
	var events []model.CalendarEvent
	err := r.db.WithContext(ctx).
		Where("interview_id = ?", interviewID).
		Preload("CalendarConnection").
		Order("created_at ASC").
		Find(&events).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list calendar events for interview: %w", err)
	}
	return events, nil
}

func (r *calendarRepository) UpsertCalendarEvent(ctx context.Context, event *model.CalendarEvent) error {
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "calendar_connection_id"}, {Name: "interview_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"external_event_id",
				"external_calendar_id",
				"sync_status",
				"last_synced_at",
				"last_error",
				"updated_at",
			}),
		}).
		Create(event).Error
	if err != nil {
		return fmt.Errorf("failed to upsert calendar event: %w", err)
	}
	return nil
}

func (r *calendarRepository) UpdateCalendarEventSyncStatus(ctx context.Context, id string, status model.CalendarSyncStatus, lastSyncedAt *time.Time, lastError *string) error {
	updates := map[string]interface{}{
		"sync_status": status,
		"updated_at":  time.Now().UTC(),
	}
	if lastSyncedAt != nil {
		updates["last_synced_at"] = lastSyncedAt
	}
	if lastError != nil {
		updates["last_error"] = lastError
	} else {
		updates["last_error"] = nil
	}

	err := r.db.WithContext(ctx).
		Model(&model.CalendarEvent{}).
		Where("id = ?", id).
		Updates(updates).Error
	if err != nil {
		return fmt.Errorf("failed to update calendar event sync status: %w", err)
	}
	return nil
}

func (r *calendarRepository) DeleteCalendarEvent(ctx context.Context, id string) error {
	err := r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.CalendarEvent{}).Error
	if err != nil {
		return fmt.Errorf("failed to delete calendar event: %w", err)
	}
	return nil
}
