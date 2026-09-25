package tests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"hirescope/backend/internal/calendar"
	"hirescope/backend/internal/config"
	"hirescope/backend/internal/crypto"
	"hirescope/backend/internal/handler"
	"hirescope/backend/internal/middleware"
	"hirescope/backend/internal/model"
	"hirescope/backend/internal/repository"
	"hirescope/backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type memoryCalendarRepo struct {
	mu          sync.RWMutex
	connections map[string]*model.CalendarConnection
	events      map[string]*model.CalendarEvent
}

func newMemoryCalendarRepo() *memoryCalendarRepo {
	return &memoryCalendarRepo{
		connections: make(map[string]*model.CalendarConnection),
		events:      make(map[string]*model.CalendarEvent),
	}
}

func (r *memoryCalendarRepo) GetConnection(ctx context.Context, userID string, provider model.CalendarProviderType) (*model.CalendarConnection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, c := range r.connections {
		if c.UserID == userID && c.Provider == provider {
			return c, nil
		}
	}
	return nil, repository.ErrConnectionNotFound
}

func (r *memoryCalendarRepo) GetConnectionByID(ctx context.Context, id string) (*model.CalendarConnection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	c, ok := r.connections[id]
	if !ok {
		return nil, repository.ErrConnectionNotFound
	}
	return c, nil
}

func (r *memoryCalendarRepo) ListConnectionsByUserID(ctx context.Context, userID string) ([]model.CalendarConnection, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []model.CalendarConnection
	for _, c := range r.connections {
		if c.UserID == userID {
			result = append(result, *c)
		}
	}
	return result, nil
}

func (r *memoryCalendarRepo) UpsertConnection(ctx context.Context, conn *model.CalendarConnection) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if conn.ID == "" {
		conn.ID = uuid.New().String()
	}
	// Check existing by (user_id, provider)
	for id, c := range r.connections {
		if c.UserID == conn.UserID && c.Provider == conn.Provider {
			c.EncryptedAccessToken = conn.EncryptedAccessToken
			c.EncryptedRefreshToken = conn.EncryptedRefreshToken
			c.TokenExpiresAt = conn.TokenExpiresAt
			c.Status = conn.Status
			c.ProviderEmail = conn.ProviderEmail
			c.ProviderAccountID = conn.ProviderAccountID
			c.Scopes = conn.Scopes
			c.UpdatedAt = time.Now().UTC()
			conn.ID = id
			return nil
		}
	}

	conn.CreatedAt = time.Now().UTC()
	conn.UpdatedAt = time.Now().UTC()
	r.connections[conn.ID] = conn
	return nil
}

func (r *memoryCalendarRepo) UpdateConnectionStatus(ctx context.Context, id string, status model.CalendarConnectionStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	c, ok := r.connections[id]
	if !ok {
		return repository.ErrConnectionNotFound
	}
	c.Status = status
	c.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *memoryCalendarRepo) UpdateConnectionTokens(ctx context.Context, id string, encAccessToken string, encRefreshToken string, expiresAt *time.Time, status model.CalendarConnectionStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	c, ok := r.connections[id]
	if !ok {
		return repository.ErrConnectionNotFound
	}
	c.EncryptedAccessToken = encAccessToken
	if encRefreshToken != "" {
		c.EncryptedRefreshToken = encRefreshToken
	}
	c.TokenExpiresAt = expiresAt
	c.Status = status
	c.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *memoryCalendarRepo) DeleteConnection(ctx context.Context, userID string, provider model.CalendarProviderType) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, c := range r.connections {
		if c.UserID == userID && c.Provider == provider {
			c.Status = model.CalendarConnectionStatusDisconnected
			c.EncryptedAccessToken = ""
			c.EncryptedRefreshToken = ""
			c.UpdatedAt = time.Now().UTC()
			return nil
		}
	}
	return nil
}

func (r *memoryCalendarRepo) GetCalendarEvent(ctx context.Context, connectionID, interviewID string) (*model.CalendarEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, e := range r.events {
		if e.CalendarConnectionID == connectionID && e.InterviewID == interviewID {
			return e, nil
		}
	}
	return nil, repository.ErrCalendarEventNotFound
}

func (r *memoryCalendarRepo) GetCalendarEventByID(ctx context.Context, id string) (*model.CalendarEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	e, ok := r.events[id]
	if !ok {
		return nil, repository.ErrCalendarEventNotFound
	}
	return e, nil
}

func (r *memoryCalendarRepo) ListCalendarEventsByInterviewID(ctx context.Context, interviewID string) ([]model.CalendarEvent, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []model.CalendarEvent
	for _, e := range r.events {
		if e.InterviewID == interviewID {
			result = append(result, *e)
		}
	}
	return result, nil
}

func (r *memoryCalendarRepo) UpsertCalendarEvent(ctx context.Context, event *model.CalendarEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if event.ID == "" {
		event.ID = uuid.New().String()
	}

	for id, e := range r.events {
		if e.CalendarConnectionID == event.CalendarConnectionID && e.InterviewID == event.InterviewID {
			e.ExternalEventID = event.ExternalEventID
			e.ExternalCalendarID = event.ExternalCalendarID
			e.SyncStatus = event.SyncStatus
			e.LastSyncedAt = event.LastSyncedAt
			e.LastError = event.LastError
			e.UpdatedAt = time.Now().UTC()
			event.ID = id
			return nil
		}
	}

	event.CreatedAt = time.Now().UTC()
	event.UpdatedAt = time.Now().UTC()
	r.events[event.ID] = event
	return nil
}

func (r *memoryCalendarRepo) UpdateCalendarEventSyncStatus(ctx context.Context, id string, status model.CalendarSyncStatus, lastSyncedAt *time.Time, lastError *string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	e, ok := r.events[id]
	if !ok {
		return repository.ErrCalendarEventNotFound
	}
	e.SyncStatus = status
	if lastSyncedAt != nil {
		e.LastSyncedAt = lastSyncedAt
	}
	e.LastError = lastError
	e.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *memoryCalendarRepo) DeleteCalendarEvent(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.events, id)
	return nil
}

// -------------------- TESTS --------------------

func TestCalendar_OAuthStateValidationAndCSRF(t *testing.T) {
	sm := calendar.NewStateManager(500 * time.Millisecond)

	userID := "user-recruiter-123"
	token, err := sm.GenerateState(userID, model.CalendarProviderGoogle)
	if err != nil {
		t.Fatalf("failed to generate state: %v", err)
	}
	if len(token) != 64 { // 32 bytes hex encoded
		t.Fatalf("expected 64 chars hex token, got %d", len(token))
	}

	// 1. Provider mismatch check
	_, err = sm.ValidateAndConsumeState(token, model.CalendarProviderMicrosoft)
	if !errors.Is(err, calendar.ErrStateMismatch) {
		t.Fatalf("expected ErrStateMismatch, got %v", err)
	}

	// 2. State was consumed on mismatch, retry should yield ErrInvalidState
	_, err = sm.ValidateAndConsumeState(token, model.CalendarProviderGoogle)
	if !errors.Is(err, calendar.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState on reused token, got %v", err)
	}

	// 3. Normal success consumption
	token2, _ := sm.GenerateState(userID, model.CalendarProviderGoogle)
	consumed, err := sm.ValidateAndConsumeState(token2, model.CalendarProviderGoogle)
	if err != nil {
		t.Fatalf("expected valid state consumption, got %v", err)
	}
	if consumed.UserID != userID || consumed.Provider != model.CalendarProviderGoogle {
		t.Fatalf("state metadata mismatch: %+v", consumed)
	}

	// 4. Expiration check
	token3, _ := sm.GenerateState(userID, model.CalendarProviderMicrosoft)
	time.Sleep(600 * time.Millisecond)
	_, err = sm.ValidateAndConsumeState(token3, model.CalendarProviderMicrosoft)
	if !errors.Is(err, calendar.ErrInvalidState) {
		t.Fatalf("expected expired state to return ErrInvalidState, got %v", err)
	}
}

func TestCalendar_TokenEncryptionAndDecryption(t *testing.T) {
	enc, err := crypto.NewTokenEncryptor("test-secret-key-32-bytes-secure!")
	if err != nil {
		t.Fatalf("failed to init encryptor: %v", err)
	}

	rawAccessToken := "ya29.sample-google-oauth-access-token-live"
	ciphertext, err := enc.Encrypt(rawAccessToken)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	if strings.Contains(ciphertext, "ya29") {
		t.Fatalf("ciphertext must not expose plaintext segments")
	}

	decrypted, err := enc.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}
	if decrypted != rawAccessToken {
		t.Fatalf("expected %s, got %s", rawAccessToken, decrypted)
	}

	// Tampered ciphertext check
	tampered := ciphertext[:len(ciphertext)-4] + "AAAA"
	_, err = enc.Decrypt(tampered)
	if err == nil {
		t.Fatalf("expected decryption error on tampered ciphertext, got nil")
	}
}

func TestCalendar_OAuthCallbackAndConnectionCreation(t *testing.T) {
	repo := newMemoryCalendarRepo()
	intRepo := newMemoryInterviewRepo()
	userRepo := newMemoryUserRepo()
	auditRepo := newMemoryAuditLogRepo()
	enc, _ := crypto.NewTokenEncryptor("calendar-encryption-key-unit-test")
	sm := calendar.NewStateManager(10 * time.Minute)

	googleMock := calendar.NewMockCalendarProvider(model.CalendarProviderGoogle)
	msMock := calendar.NewMockCalendarProvider(model.CalendarProviderMicrosoft)

	cfg := &config.Config{AppEnv: "test"}
	service := service.NewCalendarIntegrationService(
		repo, intRepo, userRepo, auditRepo, enc, sm, googleMock, msMock, cfg,
	)

	userID := "user-recruiter-456"

	// 1. Initiate OAuth
	authURL, err := service.InitiateOAuth(context.Background(), userID, model.CalendarProviderGoogle)
	if err != nil {
		t.Fatalf("failed to initiate OAuth: %v", err)
	}
	if !strings.Contains(authURL, "state=") {
		t.Fatalf("auth URL must contain state query: %s", authURL)
	}

	// Extract state from auth URL
	parts := strings.Split(authURL, "state=")
	stateToken := parts[1]

	// 2. Handle callback
	dto, err := service.HandleOAuthCallback(context.Background(), model.CalendarProviderGoogle, stateToken, "auth-code-123")
	if err != nil {
		t.Fatalf("failed to handle OAuth callback: %v", err)
	}

	if dto.Provider != model.CalendarProviderGoogle {
		t.Fatalf("expected provider GOOGLE, got %s", dto.Provider)
	}
	if dto.Status != model.CalendarConnectionStatusConnected {
		t.Fatalf("expected status CONNECTED, got %s", dto.Status)
	}
	if dto.ProviderEmail != "recruiter@GOOGLE.mock" {
		t.Fatalf("expected recruiter@GOOGLE.mock, got %s", dto.ProviderEmail)
	}

	// 3. Verify database record has encrypted tokens
	stored, err := repo.GetConnection(context.Background(), userID, model.CalendarProviderGoogle)
	if err != nil {
		t.Fatalf("failed to retrieve stored connection: %v", err)
	}
	if stored.EncryptedAccessToken == "" || stored.EncryptedRefreshToken == "" {
		t.Fatalf("tokens must be stored in database")
	}
	if strings.Contains(stored.EncryptedAccessToken, "mock-access-token") {
		t.Fatalf("access token must be encrypted at rest!")
	}

	// 4. Verify audit log was recorded
	var foundAudit bool
	for _, l := range auditRepo.logs {
		if l.Action == model.AuditActionCalendarConnected && l.ActorID == userID {
			foundAudit = true
			break
		}
	}
	if !foundAudit {
		t.Fatalf("expected CALENDAR_CONNECTED audit log")
	}

	// 5. Verify DTO list omits all token data
	conns, err := service.GetConnections(context.Background(), userID)
	if err != nil {
		t.Fatalf("failed to get connections: %v", err)
	}
	if len(conns) != 2 {
		t.Fatalf("expected 2 connections (Google & Microsoft), got %d", len(conns))
	}

	jsonBytes, _ := json.Marshal(conns)
	if strings.Contains(string(jsonBytes), "token") || strings.Contains(string(jsonBytes), "secret") {
		t.Fatalf("tokens must NEVER be exposed in connection list JSON: %s", string(jsonBytes))
	}
}

func TestCalendar_TokenRefreshAndStatusTransitions(t *testing.T) {
	repo := newMemoryCalendarRepo()
	intRepo := newMemoryInterviewRepo()
	userRepo := newMemoryUserRepo()
	auditRepo := newMemoryAuditLogRepo()
	enc, _ := crypto.NewTokenEncryptor("calendar-encryption-key-unit-test")
	sm := calendar.NewStateManager(10 * time.Minute)

	googleMock := calendar.NewMockCalendarProvider(model.CalendarProviderGoogle)
	msMock := calendar.NewMockCalendarProvider(model.CalendarProviderMicrosoft)

	cfg := &config.Config{AppEnv: "test"}
	calService := service.NewCalendarIntegrationService(
		repo, intRepo, userRepo, auditRepo, enc, sm, googleMock, msMock, cfg,
	)

	userID := "user-recruiter-789"
	encOldAccess, _ := enc.Encrypt("old-access-token")
	encOldRefresh, _ := enc.Encrypt("valid-refresh-token")

	// Store an expired connection
	expiredTime := time.Now().UTC().Add(-1 * time.Hour)
	conn := &model.CalendarConnection{
		UserID:                userID,
		Provider:              model.CalendarProviderGoogle,
		ProviderEmail:         "recruiter@google.mock",
		EncryptedAccessToken:  encOldAccess,
		EncryptedRefreshToken: encOldRefresh,
		TokenExpiresAt:        &expiredTime,
		Status:                model.CalendarConnectionStatusConnected,
	}
	_ = repo.UpsertConnection(context.Background(), conn)

	// Create test interview
	now := time.Now().UTC()
	meetingURL := "https://meet.google.com/abc-xyz"
	candidate := &model.Candidate{
		ID:       uuid.New().String(),
		FullName: "Alice Candidate",
		Email:    "alice@example.com",
	}
	job := &model.Job{
		ID:    uuid.New().String(),
		Title: "Principal Engineer",
	}
	interview := &model.Interview{
		ID:             uuid.New().String(),
		JobID:          job.ID,
		CandidateID:    candidate.ID,
		Title:          "Technical Screening",
		Stage:          model.InterviewStageTechnicalInterview,
		InterviewType:  model.InterviewTypeOnline,
		ScheduledStart: now.Add(24 * time.Hour),
		ScheduledEnd:   now.Add(25 * time.Hour),
		Timezone:       "Asia/Jakarta",
		MeetingURL:     &meetingURL,
		Status:         model.InterviewStatusScheduled,
		Candidate:      candidate,
		Job:            job,
	}
	_ = intRepo.Create(context.Background(), interview, nil, nil)

	// Sync should automatically refresh token
	dto, err := calService.SyncInterview(context.Background(), userID, interview.ID, model.CalendarProviderGoogle)
	if err != nil {
		t.Fatalf("expected successful sync after automatic token refresh, got: %v", err)
	}
	if dto.SyncStatus != model.CalendarSyncStatusSynced {
		t.Fatalf("expected SYNCED, got %s", dto.SyncStatus)
	}

	// Verify refreshed token in DB has new expiration in future
	updatedConn, _ := repo.GetConnection(context.Background(), userID, model.CalendarProviderGoogle)
	if updatedConn.TokenExpiresAt == nil || updatedConn.TokenExpiresAt.Before(time.Now().UTC()) {
		t.Fatalf("expected token expiration to be renewed in future")
	}

	// Now test revoked refresh token
	googleMock.RefreshErr = calendar.ErrAuthExpired
	expiredTime2 := time.Now().UTC().Add(-2 * time.Hour)
	updatedConn.TokenExpiresAt = &expiredTime2
	_ = repo.UpsertConnection(context.Background(), updatedConn)

	_, err = calService.SyncInterview(context.Background(), userID, interview.ID, model.CalendarProviderGoogle)
	if err == nil {
		t.Fatalf("expected error on revoked token, got nil")
	}

	// Status must transition to REVOKED
	revokedConn, _ := repo.GetConnection(context.Background(), userID, model.CalendarProviderGoogle)
	if revokedConn.Status != model.CalendarConnectionStatusRevoked {
		t.Fatalf("expected connection status to transition to REVOKED, got %s", revokedConn.Status)
	}
}

func TestCalendar_GoogleAndMicrosoftEventCRUD(t *testing.T) {
	repo := newMemoryCalendarRepo()
	intRepo := newMemoryInterviewRepo()
	userRepo := newMemoryUserRepo()
	auditRepo := newMemoryAuditLogRepo()
	enc, _ := crypto.NewTokenEncryptor("calendar-encryption-key-unit-test")
	sm := calendar.NewStateManager(10 * time.Minute)

	googleMock := calendar.NewMockCalendarProvider(model.CalendarProviderGoogle)
	msMock := calendar.NewMockCalendarProvider(model.CalendarProviderMicrosoft)

	cfg := &config.Config{AppEnv: "test"}
	calService := service.NewCalendarIntegrationService(
		repo, intRepo, userRepo, auditRepo, enc, sm, googleMock, msMock, cfg,
	)

	userID := "user-recruiter-999"
	encAccess, _ := enc.Encrypt("access-token-active")
	futureExp := time.Now().UTC().Add(2 * time.Hour)

	_ = repo.UpsertConnection(context.Background(), &model.CalendarConnection{
		UserID:               userID,
		Provider:             model.CalendarProviderGoogle,
		ProviderEmail:        "user@gmail.com",
		EncryptedAccessToken: encAccess,
		TokenExpiresAt:       &futureExp,
		Status:               model.CalendarConnectionStatusConnected,
	})
	_ = repo.UpsertConnection(context.Background(), &model.CalendarConnection{
		UserID:               userID,
		Provider:             model.CalendarProviderMicrosoft,
		ProviderEmail:        "user@outlook.com",
		EncryptedAccessToken: encAccess,
		TokenExpiresAt:       &futureExp,
		Status:               model.CalendarConnectionStatusConnected,
	})

	candidate := &model.Candidate{
		ID:       uuid.New().String(),
		FullName: "Bob Builder",
		Email:    "bob@builder.com",
	}
	job := &model.Job{
		ID:    uuid.New().String(),
		Title: "Infrastructure Architect",
	}
	now := time.Now().UTC()
	meetingURL := "https://meet.google.com/bob-infra"
	interview := &model.Interview{
		ID:             uuid.New().String(),
		JobID:          job.ID,
		CandidateID:    candidate.ID,
		Title:          "Architecture Review",
		Stage:          model.InterviewStageTechnicalInterview,
		InterviewType:  model.InterviewTypeOnline,
		ScheduledStart: now.Add(48 * time.Hour),
		ScheduledEnd:   now.Add(49 * time.Hour),
		Timezone:       "Asia/Jakarta",
		MeetingURL:     &meetingURL,
		Status:         model.InterviewStatusScheduled,
		Candidate:      candidate,
		Job:            job,
	}
	_ = intRepo.Create(context.Background(), interview, nil, nil)

	// 1. Google Create Event
	gEvent, err := calService.SyncInterview(context.Background(), userID, interview.ID, model.CalendarProviderGoogle)
	if err != nil {
		t.Fatalf("Google sync failed: %v", err)
	}
	if gEvent.SyncStatus != model.CalendarSyncStatusSynced {
		t.Fatalf("expected SYNCED, got %s", gEvent.SyncStatus)
	}
	if googleMock.LastAction != "CREATE" {
		t.Fatalf("expected CREATE action on google mock, got %s", googleMock.LastAction)
	}

	// 2. Microsoft Create Event
	msEvent, err := calService.SyncInterview(context.Background(), userID, interview.ID, model.CalendarProviderMicrosoft)
	if err != nil {
		t.Fatalf("Microsoft sync failed: %v", err)
	}
	if msEvent.SyncStatus != model.CalendarSyncStatusSynced {
		t.Fatalf("expected SYNCED, got %s", msEvent.SyncStatus)
	}
	if msMock.LastAction != "CREATE" {
		t.Fatalf("expected CREATE action on microsoft mock, got %s", msMock.LastAction)
	}

	// 3. Reschedule Hook Updates Both Events
	interview.ScheduledStart = now.Add(50 * time.Hour)
	interview.ScheduledEnd = now.Add(51 * time.Hour)
	err = calService.HandleInterviewRescheduled(context.Background(), interview, userID)
	if err != nil {
		t.Fatalf("Reschedule hook failed: %v", err)
	}
	if googleMock.LastAction != "UPDATE" || msMock.LastAction != "UPDATE" {
		t.Fatalf("expected UPDATE action on both mocks upon reschedule")
	}

	// 4. Cancel Hook Deletes Both Events
	err = calService.HandleInterviewCancelled(context.Background(), interview, userID)
	if err != nil {
		t.Fatalf("Cancel hook failed: %v", err)
	}
	if googleMock.LastAction != "DELETE" || msMock.LastAction != "DELETE" {
		t.Fatalf("expected DELETE action on both mocks upon cancel")
	}
}

func TestCalendar_DuplicatePrevention(t *testing.T) {
	repo := newMemoryCalendarRepo()
	intRepo := newMemoryInterviewRepo()
	userRepo := newMemoryUserRepo()
	auditRepo := newMemoryAuditLogRepo()
	enc, _ := crypto.NewTokenEncryptor("calendar-encryption-key-unit-test")
	sm := calendar.NewStateManager(10 * time.Minute)

	googleMock := calendar.NewMockCalendarProvider(model.CalendarProviderGoogle)
	cfg := &config.Config{AppEnv: "test"}
	calService := service.NewCalendarIntegrationService(
		repo, intRepo, userRepo, auditRepo, enc, sm, googleMock, nil, cfg,
	)

	userID := "user-recruiter-dup"
	encAccess, _ := enc.Encrypt("active-access")
	futureExp := time.Now().UTC().Add(2 * time.Hour)

	_ = repo.UpsertConnection(context.Background(), &model.CalendarConnection{
		UserID:               userID,
		Provider:             model.CalendarProviderGoogle,
		ProviderEmail:        "dup@gmail.com",
		EncryptedAccessToken: encAccess,
		TokenExpiresAt:       &futureExp,
		Status:               model.CalendarConnectionStatusConnected,
	})

	interview := &model.Interview{
		ID:             uuid.New().String(),
		JobID:          uuid.New().String(),
		CandidateID:    uuid.New().String(),
		Title:          "Screening",
		Stage:          model.InterviewStagePhoneScreen,
		InterviewType:  model.InterviewTypePhone,
		ScheduledStart: time.Now().UTC().Add(24 * time.Hour),
		ScheduledEnd:   time.Now().UTC().Add(25 * time.Hour),
		Timezone:       "Asia/Jakarta",
		Status:         model.InterviewStatusScheduled,
	}
	_ = intRepo.Create(context.Background(), interview, nil, nil)

	// First sync creates event
	evt1, err := calService.SyncInterview(context.Background(), userID, interview.ID, model.CalendarProviderGoogle)
	if err != nil {
		t.Fatalf("first sync failed: %v", err)
	}

	// Second sync on same interview must NOT create second event, but update
	evt2, err := calService.SyncInterview(context.Background(), userID, interview.ID, model.CalendarProviderGoogle)
	if err != nil {
		t.Fatalf("second sync failed: %v", err)
	}

	if evt1.ID != evt2.ID {
		t.Fatalf("expected same mapping ID, got %s and %s", evt1.ID, evt2.ID)
	}

	events, _ := repo.ListCalendarEventsByInterviewID(context.Background(), interview.ID)
	if len(events) != 1 {
		t.Fatalf("expected exactly 1 calendar event mapping, got %d", len(events))
	}
}

func TestCalendar_ProviderFailureSafety(t *testing.T) {
	repo := newMemoryCalendarRepo()
	intRepo := newMemoryInterviewRepo()
	userRepo := newMemoryUserRepo()
	auditRepo := newMemoryAuditLogRepo()
	enc, _ := crypto.NewTokenEncryptor("calendar-encryption-key-unit-test")
	sm := calendar.NewStateManager(10 * time.Minute)

	googleMock := calendar.NewMockCalendarProvider(model.CalendarProviderGoogle)
	googleMock.CreateEventErr = fmt.Errorf("Google 503 Service Unavailable")

	cfg := &config.Config{AppEnv: "test"}
	calService := service.NewCalendarIntegrationService(
		repo, intRepo, userRepo, auditRepo, enc, sm, googleMock, nil, cfg,
	)

	userID := "user-recruiter-fail"
	encAccess, _ := enc.Encrypt("active-access")
	futureExp := time.Now().UTC().Add(2 * time.Hour)

	_ = repo.UpsertConnection(context.Background(), &model.CalendarConnection{
		UserID:               userID,
		Provider:             model.CalendarProviderGoogle,
		ProviderEmail:        "fail@gmail.com",
		EncryptedAccessToken: encAccess,
		TokenExpiresAt:       &futureExp,
		Status:               model.CalendarConnectionStatusConnected,
	})

	interview := &model.Interview{
		ID:             uuid.New().String(),
		JobID:          uuid.New().String(),
		CandidateID:    uuid.New().String(),
		Title:          "Screening Failure Test",
		Stage:          model.InterviewStagePhoneScreen,
		InterviewType:  model.InterviewTypePhone,
		ScheduledStart: time.Now().UTC().Add(24 * time.Hour),
		ScheduledEnd:   time.Now().UTC().Add(25 * time.Hour),
		Timezone:       "Asia/Jakarta",
		Status:         model.InterviewStatusScheduled,
	}
	_ = intRepo.Create(context.Background(), interview, nil, nil)

	// Sync fails
	_, err := calService.SyncInterview(context.Background(), userID, interview.ID, model.CalendarProviderGoogle)
	if err == nil {
		t.Fatalf("expected error from failed provider, got nil")
	}

	// Interview must remain valid and intact!
	itw, _ := intRepo.GetByID(context.Background(), interview.ID)
	if itw == nil || itw.Status != model.InterviewStatusScheduled {
		t.Fatalf("interview must remain scheduled and unaffected by external calendar failure")
	}

	// External sync mapping must be recorded as FAILED
	events, _ := repo.ListCalendarEventsByInterviewID(context.Background(), interview.ID)
	if len(events) != 1 || events[0].SyncStatus != model.CalendarSyncStatusFailed {
		t.Fatalf("expected 1 mapping in FAILED status, got %+v", events)
	}

	// Retry succeeds when provider recovers
	googleMock.CreateEventErr = nil
	retried, err := calService.RetrySync(context.Background(), userID, interview.ID, model.CalendarProviderGoogle)
	if err != nil {
		t.Fatalf("expected successful retry after provider recovery, got %v", err)
	}
	if retried.SyncStatus != model.CalendarSyncStatusSynced {
		t.Fatalf("expected SYNCED on retry, got %s", retried.SyncStatus)
	}
}

func TestCalendar_UserIsolation(t *testing.T) {
	repo := newMemoryCalendarRepo()
	intRepo := newMemoryInterviewRepo()
	userRepo := newMemoryUserRepo()
	auditRepo := newMemoryAuditLogRepo()
	enc, _ := crypto.NewTokenEncryptor("calendar-encryption-key-unit-test")
	sm := calendar.NewStateManager(10 * time.Minute)

	googleMock := calendar.NewMockCalendarProvider(model.CalendarProviderGoogle)
	cfg := &config.Config{AppEnv: "test"}
	calService := service.NewCalendarIntegrationService(
		repo, intRepo, userRepo, auditRepo, enc, sm, googleMock, nil, cfg,
	)

	userA := "user-recruiter-A"
	userB := "user-recruiter-B"
	encAccess, _ := enc.Encrypt("active-access")
	futureExp := time.Now().UTC().Add(2 * time.Hour)

	// User A connects Google
	_ = repo.UpsertConnection(context.Background(), &model.CalendarConnection{
		UserID:               userA,
		Provider:             model.CalendarProviderGoogle,
		ProviderEmail:        "userA@gmail.com",
		EncryptedAccessToken: encAccess,
		TokenExpiresAt:       &futureExp,
		Status:               model.CalendarConnectionStatusConnected,
	})

	interview := &model.Interview{
		ID:             uuid.New().String(),
		JobID:          uuid.New().String(),
		CandidateID:    uuid.New().String(),
		Title:          "Interview 1",
		Stage:          model.InterviewStagePhoneScreen,
		InterviewType:  model.InterviewTypePhone,
		ScheduledStart: time.Now().UTC().Add(24 * time.Hour),
		ScheduledEnd:   time.Now().UTC().Add(25 * time.Hour),
		Timezone:       "Asia/Jakarta",
		Status:         model.InterviewStatusScheduled,
	}
	_ = intRepo.Create(context.Background(), interview, nil, nil)

	// User B tries to sync interview using Google without connecting Google
	_, err := calService.SyncInterview(context.Background(), userB, interview.ID, model.CalendarProviderGoogle)
	if !errors.Is(err, service.ErrCalendarNotConnected) {
		t.Fatalf("expected ErrCalendarNotConnected for User B, got: %v", err)
	}

	// User B connection list should show Disconnected
	connsB, _ := calService.GetConnections(context.Background(), userB)
	for _, c := range connsB {
		if c.Provider == model.CalendarProviderGoogle && c.Status == model.CalendarConnectionStatusConnected {
			t.Fatalf("User B must not see User A's connected status!")
		}
	}
}

type memoryRevokedTokenRepo struct{}

func (m *memoryRevokedTokenRepo) Revoke(ctx context.Context, tokenHash string, expiresAt time.Time) error {
	return nil
}
func (m *memoryRevokedTokenRepo) IsRevoked(ctx context.Context, tokenHash string) (bool, error) {
	return false, nil
}
func (m *memoryRevokedTokenRepo) CleanExpired(ctx context.Context) error {
	return nil
}

func TestCalendar_HTTPRouteAuthorization(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := newMemoryCalendarRepo()
	intRepo := newMemoryInterviewRepo()
	userRepo := newMemoryUserRepo()
	auditRepo := newMemoryAuditLogRepo()
	enc, _ := crypto.NewTokenEncryptor("calendar-encryption-key-unit-test")
	sm := calendar.NewStateManager(10 * time.Minute)

	googleMock := calendar.NewMockCalendarProvider(model.CalendarProviderGoogle)
	cfg := &config.Config{AppEnv: "test"}
	calService := service.NewCalendarIntegrationService(
		repo, intRepo, userRepo, auditRepo, enc, sm, googleMock, nil, cfg,
	)

	h := handler.NewCalendarHandler(calService, cfg)
	jwtService, _ := service.NewJWTService("jwt-secret-testing-authorization-key", 24)
	revokedRepo := &memoryRevokedTokenRepo{}

	r := gin.New()
	v1 := r.Group("/api/v1")
	{
		calendarGroup := v1.Group("/calendar")
		{
			calendarGroup.GET("/google/callback", h.CallbackGoogle)

			calendarAuth := calendarGroup.Group("")
			calendarAuth.Use(middleware.AuthMiddleware(jwtService, revokedRepo))
			calendarAuth.Use(middleware.RequireRole(model.RoleAdmin, model.RoleRecruiter))
			{
				calendarAuth.GET("/connections", h.GetConnections)
			}
		}
	}

	// 1. Unauthenticated request to /calendar/connections -> 401
	reqUnauth, _ := http.NewRequest(http.MethodGet, "/api/v1/calendar/connections", nil)
	wUnauth := httptest.NewRecorder()
	r.ServeHTTP(wUnauth, reqUnauth)
	if wUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized, got %d", wUnauth.Code)
	}

	// 2. Authenticated Recruiter request -> 200 OK
	token, _, _ := jwtService.GenerateToken(&model.User{
		ID:    "user-recruiter-1",
		Email: "recruiter@hirescope.internal",
		Role:  model.RoleRecruiter,
	})
	reqAuth, _ := http.NewRequest(http.MethodGet, "/api/v1/calendar/connections", nil)
	reqAuth.Header.Set("Authorization", "Bearer "+token)
	wAuth := httptest.NewRecorder()
	r.ServeHTTP(wAuth, reqAuth)
	if wAuth.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", wAuth.Code, wAuth.Body.String())
	}
}

func TestCalendar_TimezonePreservation(t *testing.T) {
	repo := newMemoryCalendarRepo()
	intRepo := newMemoryInterviewRepo()
	userRepo := newMemoryUserRepo()
	auditRepo := newMemoryAuditLogRepo()
	enc, _ := crypto.NewTokenEncryptor("calendar-encryption-key-unit-test")
	sm := calendar.NewStateManager(10 * time.Minute)

	googleMock := calendar.NewMockCalendarProvider(model.CalendarProviderGoogle)
	cfg := &config.Config{AppEnv: "test"}
	calService := service.NewCalendarIntegrationService(
		repo, intRepo, userRepo, auditRepo, enc, sm, googleMock, nil, cfg,
	)

	userID := "user-recruiter-tz"
	encAccess, _ := enc.Encrypt("access-token-active")
	futureExp := time.Now().UTC().Add(2 * time.Hour)

	_ = repo.UpsertConnection(context.Background(), &model.CalendarConnection{
		UserID:               userID,
		Provider:             model.CalendarProviderGoogle,
		ProviderEmail:        "recruiter@gmail.com",
		EncryptedAccessToken: encAccess,
		TokenExpiresAt:       &futureExp,
		Status:               model.CalendarConnectionStatusConnected,
	})

	// Set interview in Asia/Jakarta timezone: 2026-10-15 14:00 to 15:00
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("failed to load Asia/Jakarta location: %v", err)
	}
	startTime := time.Date(2026, 10, 15, 14, 0, 0, 0, loc)
	endTime := time.Date(2026, 10, 15, 15, 0, 0, 0, loc)

	interview := &model.Interview{
		ID:             uuid.New().String(),
		JobID:          uuid.New().String(),
		CandidateID:    uuid.New().String(),
		Title:          "Senior Architect Interview",
		Stage:          model.InterviewStageTechnicalInterview,
		InterviewType:  model.InterviewTypeOnline,
		ScheduledStart: startTime,
		ScheduledEnd:   endTime,
		Timezone:       "Asia/Jakarta",
		Status:         model.InterviewStatusScheduled,
	}
	_ = intRepo.Create(context.Background(), interview, nil, nil)

	_, err = calService.SyncInterview(context.Background(), userID, interview.ID, model.CalendarProviderGoogle)
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	// Verify event stored in mock has Timezone "Asia/Jakarta"
	var createdEvent *calendar.CalendarEventData
	for _, evt := range googleMock.Events {
		createdEvent = evt
		break
	}
	if createdEvent == nil {
		t.Fatalf("no event was created in mock")
	}

	if createdEvent.Timezone != "Asia/Jakarta" {
		t.Fatalf("expected Timezone Asia/Jakarta, got: %s", createdEvent.Timezone)
	}

	if !createdEvent.Start.Equal(startTime) || !createdEvent.End.Equal(endTime) {
		t.Fatalf("start/end timestamps skewed: expected %v - %v, got %v - %v", startTime, endTime, createdEvent.Start, createdEvent.End)
	}
}
