package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"hirescope/backend/internal/calendar"
	"hirescope/backend/internal/config"
	"hirescope/backend/internal/crypto"
	"hirescope/backend/internal/model"
	"hirescope/backend/internal/repository"
)

var (
	ErrCalendarNotConnected = errors.New("calendar provider is not connected")
	ErrSyncFailed           = errors.New("external calendar synchronization failed")
)

// CalendarConnectionDTO represents safe provider connection metadata.
// Tokens and secrets are strictly excluded.
type CalendarConnectionDTO struct {
	ID            string                         `json:"id,omitempty"`
	Provider      model.CalendarProviderType     `json:"provider"`
	ProviderEmail string                         `json:"provider_email,omitempty"`
	Status        model.CalendarConnectionStatus `json:"status"`
	IsConfigured  bool                           `json:"is_configured"`
	ConnectedAt   *time.Time                     `json:"connected_at,omitempty"`
	UpdatedAt     *time.Time                     `json:"updated_at,omitempty"`
}

// InterviewCalendarEventDTO represents external synchronization status for an interview.
type InterviewCalendarEventDTO struct {
	ID                 string                     `json:"id"`
	InterviewID        string                     `json:"interview_id"`
	Provider           model.CalendarProviderType `json:"provider"`
	ExternalEventID    string                     `json:"external_event_id"`
	ExternalCalendarID *string                    `json:"external_calendar_id,omitempty"`
	SyncStatus         model.CalendarSyncStatus   `json:"sync_status"`
	LastSyncedAt       *time.Time                 `json:"last_synced_at,omitempty"`
	LastError          *string                    `json:"last_error,omitempty"`
	CreatedAt          time.Time                  `json:"created_at"`
}

// CalendarIntegrationService defines the service contract for external calendar integrations.
type CalendarIntegrationService interface {
	GetConnections(ctx context.Context, userID string) ([]CalendarConnectionDTO, error)
	InitiateOAuth(ctx context.Context, userID string, provider model.CalendarProviderType) (authURL string, err error)
	HandleOAuthCallback(ctx context.Context, provider model.CalendarProviderType, state, code string) (*CalendarConnectionDTO, error)
	Disconnect(ctx context.Context, userID string, provider model.CalendarProviderType) error

	GetInterviewCalendarEvents(ctx context.Context, userID, interviewID string) ([]InterviewCalendarEventDTO, error)
	SyncInterview(ctx context.Context, userID, interviewID string, provider model.CalendarProviderType) (*InterviewCalendarEventDTO, error)
	RetrySync(ctx context.Context, userID, interviewID string, provider model.CalendarProviderType) (*InterviewCalendarEventDTO, error)

	HandleInterviewRescheduled(ctx context.Context, interview *model.Interview, actorID string) error
	HandleInterviewCancelled(ctx context.Context, interview *model.Interview, actorID string) error
}

type calendarIntegrationService struct {
	repo         repository.CalendarRepository
	interviewRepo repository.InterviewRepository
	userRepo     repository.UserRepository
	auditRepo    repository.AuditLogRepository
	encryptor    *crypto.TokenEncryptor
	stateManager calendar.StateManager
	providers    map[model.CalendarProviderType]calendar.CalendarProvider
	cfg          *config.Config
}

// NewCalendarIntegrationService constructs a new CalendarIntegrationService.
func NewCalendarIntegrationService(
	repo repository.CalendarRepository,
	interviewRepo repository.InterviewRepository,
	userRepo repository.UserRepository,
	auditRepo repository.AuditLogRepository,
	encryptor *crypto.TokenEncryptor,
	stateManager calendar.StateManager,
	googleProvider calendar.CalendarProvider,
	microsoftProvider calendar.CalendarProvider,
	cfg *config.Config,
) CalendarIntegrationService {
	providers := make(map[model.CalendarProviderType]calendar.CalendarProvider)
	if googleProvider != nil {
		providers[model.CalendarProviderGoogle] = googleProvider
	}
	if microsoftProvider != nil {
		providers[model.CalendarProviderMicrosoft] = microsoftProvider
	}

	return &calendarIntegrationService{
		repo:          repo,
		interviewRepo: interviewRepo,
		userRepo:      userRepo,
		auditRepo:     auditRepo,
		encryptor:     encryptor,
		stateManager:  stateManager,
		providers:     providers,
		cfg:           cfg,
	}
}

func (s *calendarIntegrationService) GetConnections(ctx context.Context, userID string) ([]CalendarConnectionDTO, error) {
	storedConns, err := s.repo.ListConnectionsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user connections: %w", err)
	}

	connMap := make(map[model.CalendarProviderType]model.CalendarConnection)
	for _, c := range storedConns {
		connMap[c.Provider] = c
	}

	supported := []model.CalendarProviderType{
		model.CalendarProviderGoogle,
		model.CalendarProviderMicrosoft,
	}

	result := make([]CalendarConnectionDTO, 0, len(supported))
	for _, pType := range supported {
		prov := s.providers[pType]
		isConfigured := prov != nil && prov.IsConfigured()

		dto := CalendarConnectionDTO{
			Provider:     pType,
			IsConfigured: isConfigured,
			Status:       model.CalendarConnectionStatusDisconnected,
		}

		if !isConfigured {
			dto.Status = model.CalendarConnectionStatusNotConfigured
		} else if existing, found := connMap[pType]; found {
			dto.ID = existing.ID
			dto.ProviderEmail = existing.ProviderEmail
			dto.Status = existing.Status
			dto.ConnectedAt = &existing.CreatedAt
			dto.UpdatedAt = &existing.UpdatedAt
		}

		result = append(result, dto)
	}

	return result, nil
}

func (s *calendarIntegrationService) InitiateOAuth(ctx context.Context, userID string, provider model.CalendarProviderType) (string, error) {
	prov, exists := s.providers[provider]
	if !exists || !prov.IsConfigured() {
		return "", calendar.ErrProviderNotConfigured
	}

	state, err := s.stateManager.GenerateState(userID, provider)
	if err != nil {
		return "", fmt.Errorf("failed to generate OAuth state: %w", err)
	}

	authURL := prov.GetAuthURL(state)
	if authURL == "" {
		return "", calendar.ErrProviderNotConfigured
	}

	return authURL, nil
}

func (s *calendarIntegrationService) HandleOAuthCallback(ctx context.Context, provider model.CalendarProviderType, state, code string) (*CalendarConnectionDTO, error) {
	oauthState, err := s.stateManager.ValidateAndConsumeState(state, provider)
	if err != nil {
		return nil, err
	}

	prov, exists := s.providers[provider]
	if !exists || !prov.IsConfigured() {
		return nil, calendar.ErrProviderNotConfigured
	}

	tokenResp, userInfo, err := prov.ExchangeCode(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange authorization code: %w", err)
	}

	encAccessToken, err := s.encryptor.Encrypt(tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt access token: %w", err)
	}

	var encRefreshToken string
	if tokenResp.RefreshToken != "" {
		encRefreshToken, err = s.encryptor.Encrypt(tokenResp.RefreshToken)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt refresh token: %w", err)
		}
	}

	var expiresAt *time.Time
	if tokenResp.ExpiresIn > 0 {
		exp := time.Now().UTC().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		expiresAt = &exp
	}

	conn := &model.CalendarConnection{
		UserID:                oauthState.UserID,
		Provider:              provider,
		ProviderAccountID:     userInfo.ID,
		ProviderEmail:         userInfo.Email,
		EncryptedAccessToken:  encAccessToken,
		EncryptedRefreshToken: encRefreshToken,
		TokenExpiresAt:        expiresAt,
		Scopes:                tokenResp.Scope,
		Status:                model.CalendarConnectionStatusConnected,
		UpdatedAt:             time.Now().UTC(),
	}

	if err := s.repo.UpsertConnection(ctx, conn); err != nil {
		return nil, fmt.Errorf("failed to save connection: %w", err)
	}

	// Create audit log
	meta, _ := json.Marshal(map[string]interface{}{
		"provider":       provider,
		"account_email":  userInfo.Email,
		"account_id":     userInfo.ID,
	})
	_ = s.auditRepo.Create(ctx, &model.AuditLog{
		Action:    model.AuditActionCalendarConnected,
		ActorID:   oauthState.UserID,
		Metadata:  string(meta),
		CreatedAt: time.Now().UTC(),
	})

	return &CalendarConnectionDTO{
		ID:            conn.ID,
		Provider:      conn.Provider,
		ProviderEmail: conn.ProviderEmail,
		Status:        conn.Status,
		IsConfigured:  true,
		ConnectedAt:   &conn.CreatedAt,
		UpdatedAt:     &conn.UpdatedAt,
	}, nil
}

func (s *calendarIntegrationService) Disconnect(ctx context.Context, userID string, provider model.CalendarProviderType) error {
	conn, err := s.repo.GetConnection(ctx, userID, provider)
	if err != nil {
		if errors.Is(err, repository.ErrConnectionNotFound) {
			return nil
		}
		return err
	}

	if err := s.repo.DeleteConnection(ctx, userID, provider); err != nil {
		return err
	}

	meta, _ := json.Marshal(map[string]interface{}{
		"provider":       provider,
		"account_email":  conn.ProviderEmail,
	})
	_ = s.auditRepo.Create(ctx, &model.AuditLog{
		Action:    model.AuditActionCalendarDisconnected,
		ActorID:   userID,
		Metadata:  string(meta),
		CreatedAt: time.Now().UTC(),
	})

	return nil
}

func (s *calendarIntegrationService) GetInterviewCalendarEvents(ctx context.Context, userID, interviewID string) ([]InterviewCalendarEventDTO, error) {
	events, err := s.repo.ListCalendarEventsByInterviewID(ctx, interviewID)
	if err != nil {
		return nil, err
	}

	result := make([]InterviewCalendarEventDTO, 0, len(events))
	for _, e := range events {
		result = append(result, InterviewCalendarEventDTO{
			ID:                 e.ID,
			InterviewID:        e.InterviewID,
			Provider:           e.Provider,
			ExternalEventID:    e.ExternalEventID,
			ExternalCalendarID: e.ExternalCalendarID,
			SyncStatus:         e.SyncStatus,
			LastSyncedAt:       e.LastSyncedAt,
			LastError:          e.LastError,
			CreatedAt:          e.CreatedAt,
		})
	}

	return result, nil
}

func (s *calendarIntegrationService) SyncInterview(ctx context.Context, userID, interviewID string, provider model.CalendarProviderType) (*InterviewCalendarEventDTO, error) {
	conn, err := s.repo.GetConnection(ctx, userID, provider)
	if err != nil || conn.Status != model.CalendarConnectionStatusConnected {
		return nil, ErrCalendarNotConnected
	}

	accessToken, err := s.getValidAccessToken(ctx, conn)
	if err != nil {
		return nil, err
	}

	interview, err := s.interviewRepo.GetByID(ctx, interviewID)
	if err != nil {
		return nil, err
	}

	prov, ok := s.providers[provider]
	if !ok || !prov.IsConfigured() {
		return nil, calendar.ErrProviderNotConfigured
	}

	eventData := s.buildCalendarEventData(interview)

	// Check if mapping already exists
	mapping, err := s.repo.GetCalendarEvent(ctx, conn.ID, interviewID)
	if err == nil && mapping != nil && mapping.ExternalEventID != "" {
		// Event already exists, perform update
		err := prov.UpdateEvent(ctx, accessToken, mapping.ExternalEventID, eventData)
		now := time.Now().UTC()
		if err != nil {
			errStr := err.Error()
			_ = s.repo.UpdateCalendarEventSyncStatus(ctx, mapping.ID, model.CalendarSyncStatusFailed, nil, &errStr)
			s.recordAuditSyncFailed(ctx, userID, interview, provider, errStr)
			return nil, fmt.Errorf("%w: %v", ErrSyncFailed, err)
		}

		_ = s.repo.UpdateCalendarEventSyncStatus(ctx, mapping.ID, model.CalendarSyncStatusSynced, &now, nil)
		s.recordAudit(ctx, model.AuditActionCalendarEventUpdated, userID, interview, provider, model.CalendarSyncStatusSynced)
		return &InterviewCalendarEventDTO{
			ID:                 mapping.ID,
			InterviewID:        interviewID,
			Provider:           provider,
			ExternalEventID:    mapping.ExternalEventID,
			ExternalCalendarID: mapping.ExternalCalendarID,
			SyncStatus:         model.CalendarSyncStatusSynced,
			LastSyncedAt:       &now,
			CreatedAt:          mapping.CreatedAt,
		}, nil
	}

	// Create new event in provider
	extEventID, err := prov.CreateEvent(ctx, accessToken, eventData)
	now := time.Now().UTC()
	if err != nil {
		errStr := err.Error()
		failedMapping := &model.CalendarEvent{
			InterviewID:          interviewID,
			CalendarConnectionID: conn.ID,
			Provider:             provider,
			ExternalEventID:      "",
			SyncStatus:           model.CalendarSyncStatusFailed,
			LastError:            &errStr,
			CreatedAt:            now,
			UpdatedAt:            now,
		}
		_ = s.repo.UpsertCalendarEvent(ctx, failedMapping)
		s.recordAuditSyncFailed(ctx, userID, interview, provider, errStr)
		return nil, fmt.Errorf("%w: %v", ErrSyncFailed, err)
	}

	primaryCal := "primary"
	newMapping := &model.CalendarEvent{
		InterviewID:          interviewID,
		CalendarConnectionID: conn.ID,
		Provider:             provider,
		ExternalEventID:      extEventID,
		ExternalCalendarID:   &primaryCal,
		SyncStatus:           model.CalendarSyncStatusSynced,
		LastSyncedAt:         &now,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if err := s.repo.UpsertCalendarEvent(ctx, newMapping); err != nil {
		return nil, fmt.Errorf("failed to save calendar event mapping: %w", err)
	}

	s.recordAudit(ctx, model.AuditActionCalendarEventCreated, userID, interview, provider, model.CalendarSyncStatusSynced)

	return &InterviewCalendarEventDTO{
		ID:                 newMapping.ID,
		InterviewID:        interviewID,
		Provider:           provider,
		ExternalEventID:    extEventID,
		ExternalCalendarID: newMapping.ExternalCalendarID,
		SyncStatus:         model.CalendarSyncStatusSynced,
		LastSyncedAt:       &now,
		CreatedAt:          now,
	}, nil
}

func (s *calendarIntegrationService) RetrySync(ctx context.Context, userID, interviewID string, provider model.CalendarProviderType) (*InterviewCalendarEventDTO, error) {
	conn, err := s.repo.GetConnection(ctx, userID, provider)
	if err != nil || conn.Status != model.CalendarConnectionStatusConnected {
		return nil, ErrCalendarNotConnected
	}

	accessToken, err := s.getValidAccessToken(ctx, conn)
	if err != nil {
		return nil, err
	}

	interview, err := s.interviewRepo.GetByID(ctx, interviewID)
	if err != nil {
		return nil, err
	}

	prov, ok := s.providers[provider]
	if !ok || !prov.IsConfigured() {
		return nil, calendar.ErrProviderNotConfigured
	}

	eventData := s.buildCalendarEventData(interview)
	mapping, err := s.repo.GetCalendarEvent(ctx, conn.ID, interviewID)
	now := time.Now().UTC()

	if err == nil && mapping != nil && mapping.ExternalEventID != "" {
		// Try update existing event
		updateErr := prov.UpdateEvent(ctx, accessToken, mapping.ExternalEventID, eventData)
		if updateErr == nil {
			_ = s.repo.UpdateCalendarEventSyncStatus(ctx, mapping.ID, model.CalendarSyncStatusSynced, &now, nil)
			s.recordAudit(ctx, model.AuditActionCalendarEventUpdated, userID, interview, provider, model.CalendarSyncStatusSynced)
			return &InterviewCalendarEventDTO{
				ID:                 mapping.ID,
				InterviewID:        interviewID,
				Provider:           provider,
				ExternalEventID:    mapping.ExternalEventID,
				ExternalCalendarID: mapping.ExternalCalendarID,
				SyncStatus:         model.CalendarSyncStatusSynced,
				LastSyncedAt:       &now,
				CreatedAt:          mapping.CreatedAt,
			}, nil
		}

		// If event was deleted in external calendar (404), fall through to recreate
		if !errors.Is(updateErr, calendar.ErrNotFound) {
			errStr := updateErr.Error()
			_ = s.repo.UpdateCalendarEventSyncStatus(ctx, mapping.ID, model.CalendarSyncStatusFailed, nil, &errStr)
			s.recordAuditSyncFailed(ctx, userID, interview, provider, errStr)
			return nil, fmt.Errorf("%w: %v", ErrSyncFailed, updateErr)
		}
	}

	// Create event
	extEventID, createErr := prov.CreateEvent(ctx, accessToken, eventData)
	if createErr != nil {
		errStr := createErr.Error()
		if mapping != nil {
			_ = s.repo.UpdateCalendarEventSyncStatus(ctx, mapping.ID, model.CalendarSyncStatusFailed, nil, &errStr)
		}
		s.recordAuditSyncFailed(ctx, userID, interview, provider, errStr)
		return nil, fmt.Errorf("%w: %v", ErrSyncFailed, createErr)
	}

	primaryCal := "primary"
	savedMapping := &model.CalendarEvent{
		InterviewID:          interviewID,
		CalendarConnectionID: conn.ID,
		Provider:             provider,
		ExternalEventID:      extEventID,
		ExternalCalendarID:   &primaryCal,
		SyncStatus:           model.CalendarSyncStatusSynced,
		LastSyncedAt:         &now,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	if err := s.repo.UpsertCalendarEvent(ctx, savedMapping); err != nil {
		return nil, fmt.Errorf("failed to save calendar event mapping: %w", err)
	}

	s.recordAudit(ctx, model.AuditActionCalendarEventCreated, userID, interview, provider, model.CalendarSyncStatusSynced)

	return &InterviewCalendarEventDTO{
		ID:                 savedMapping.ID,
		InterviewID:        interviewID,
		Provider:           provider,
		ExternalEventID:    extEventID,
		ExternalCalendarID: savedMapping.ExternalCalendarID,
		SyncStatus:         model.CalendarSyncStatusSynced,
		LastSyncedAt:       &now,
		CreatedAt:          now,
	}, nil
}

func (s *calendarIntegrationService) HandleInterviewRescheduled(ctx context.Context, interview *model.Interview, actorID string) error {
	mappings, err := s.repo.ListCalendarEventsByInterviewID(ctx, interview.ID)
	if err != nil || len(mappings) == 0 {
		return nil
	}

	eventData := s.buildCalendarEventData(interview)
	now := time.Now().UTC()

	for _, m := range mappings {
		if m.ExternalEventID == "" || m.SyncStatus == model.CalendarSyncStatusDeleted {
			continue
		}

		conn, err := s.repo.GetConnectionByID(ctx, m.CalendarConnectionID)
		if err != nil || conn.Status != model.CalendarConnectionStatusConnected {
			continue
		}

		prov, ok := s.providers[m.Provider]
		if !ok || !prov.IsConfigured() {
			continue
		}

		accessToken, err := s.getValidAccessToken(ctx, conn)
		if err != nil {
			errStr := err.Error()
			_ = s.repo.UpdateCalendarEventSyncStatus(ctx, m.ID, model.CalendarSyncStatusFailed, nil, &errStr)
			s.recordAuditSyncFailed(ctx, actorID, interview, m.Provider, errStr)
			continue
		}

		if err := prov.UpdateEvent(ctx, accessToken, m.ExternalEventID, eventData); err != nil {
			errStr := err.Error()
			_ = s.repo.UpdateCalendarEventSyncStatus(ctx, m.ID, model.CalendarSyncStatusFailed, nil, &errStr)
			s.recordAuditSyncFailed(ctx, actorID, interview, m.Provider, errStr)
			continue
		}

		_ = s.repo.UpdateCalendarEventSyncStatus(ctx, m.ID, model.CalendarSyncStatusSynced, &now, nil)
		s.recordAudit(ctx, model.AuditActionCalendarEventUpdated, actorID, interview, m.Provider, model.CalendarSyncStatusSynced)
	}

	return nil
}

func (s *calendarIntegrationService) HandleInterviewCancelled(ctx context.Context, interview *model.Interview, actorID string) error {
	mappings, err := s.repo.ListCalendarEventsByInterviewID(ctx, interview.ID)
	if err != nil || len(mappings) == 0 {
		return nil
	}

	for _, m := range mappings {
		if m.ExternalEventID == "" || m.SyncStatus == model.CalendarSyncStatusDeleted {
			continue
		}

		conn, err := s.repo.GetConnectionByID(ctx, m.CalendarConnectionID)
		if err != nil || conn.Status != model.CalendarConnectionStatusConnected {
			continue
		}

		prov, ok := s.providers[m.Provider]
		if !ok || !prov.IsConfigured() {
			continue
		}

		accessToken, err := s.getValidAccessToken(ctx, conn)
		if err != nil {
			errStr := err.Error()
			_ = s.repo.UpdateCalendarEventSyncStatus(ctx, m.ID, model.CalendarSyncStatusFailed, nil, &errStr)
			s.recordAuditSyncFailed(ctx, actorID, interview, m.Provider, errStr)
			continue
		}

		if err := prov.DeleteEvent(ctx, accessToken, m.ExternalEventID); err != nil {
			errStr := err.Error()
			_ = s.repo.UpdateCalendarEventSyncStatus(ctx, m.ID, model.CalendarSyncStatusFailed, nil, &errStr)
			s.recordAuditSyncFailed(ctx, actorID, interview, m.Provider, errStr)
			continue
		}

		_ = s.repo.UpdateCalendarEventSyncStatus(ctx, m.ID, model.CalendarSyncStatusDeleted, nil, nil)
		s.recordAudit(ctx, model.AuditActionCalendarEventDeleted, actorID, interview, m.Provider, model.CalendarSyncStatusDeleted)
	}

	return nil
}

func (s *calendarIntegrationService) getValidAccessToken(ctx context.Context, conn *model.CalendarConnection) (string, error) {
	// Check if token has expired or will expire within 5 minutes
	needsRefresh := conn.TokenExpiresAt != nil && time.Now().UTC().Add(5*time.Minute).After(*conn.TokenExpiresAt)

	if !needsRefresh && conn.EncryptedAccessToken != "" {
		decrypted, err := s.encryptor.Decrypt(conn.EncryptedAccessToken)
		if err == nil && decrypted != "" {
			return decrypted, nil
		}
	}

	// Token needs refresh
	if conn.EncryptedRefreshToken == "" {
		_ = s.repo.UpdateConnectionStatus(ctx, conn.ID, model.CalendarConnectionStatusExpired)
		return "", calendar.ErrAuthExpired
	}

	decryptedRefresh, err := s.encryptor.Decrypt(conn.EncryptedRefreshToken)
	if err != nil || decryptedRefresh == "" {
		_ = s.repo.UpdateConnectionStatus(ctx, conn.ID, model.CalendarConnectionStatusExpired)
		return "", calendar.ErrAuthExpired
	}

	prov, ok := s.providers[conn.Provider]
	if !ok || !prov.IsConfigured() {
		return "", calendar.ErrProviderNotConfigured
	}

	tokenResp, err := prov.RefreshToken(ctx, decryptedRefresh)
	if err != nil {
		if errors.Is(err, calendar.ErrAuthExpired) || errors.Is(err, calendar.ErrPermissionDenied) {
			_ = s.repo.UpdateConnectionStatus(ctx, conn.ID, model.CalendarConnectionStatusRevoked)
		} else {
			_ = s.repo.UpdateConnectionStatus(ctx, conn.ID, model.CalendarConnectionStatusError)
		}
		return "", err
	}

	encAccess, err := s.encryptor.Encrypt(tokenResp.AccessToken)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt new access token: %w", err)
	}

	var encRefresh string
	if tokenResp.RefreshToken != "" {
		encRefresh, _ = s.encryptor.Encrypt(tokenResp.RefreshToken)
	}

	var expiresAt *time.Time
	if tokenResp.ExpiresIn > 0 {
		exp := time.Now().UTC().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
		expiresAt = &exp
	}

	_ = s.repo.UpdateConnectionTokens(ctx, conn.ID, encAccess, encRefresh, expiresAt, model.CalendarConnectionStatusConnected)
	return tokenResp.AccessToken, nil
}

func (s *calendarIntegrationService) buildCalendarEventData(interview *model.Interview) *calendar.CalendarEventData {
	candidateName := "Candidate"
	candidateEmail := ""
	if interview.Candidate != nil {
		candidateName = interview.Candidate.FullName
		candidateEmail = interview.Candidate.Email
	}

	jobTitle := "Position"
	if interview.Job != nil {
		jobTitle = interview.Job.Title
	}

	// External event title format: "Interview — {{candidate_name}} — {{job_title}}"
	title := fmt.Sprintf("Interview — %s — %s", candidateName, jobTitle)

	typeStr := string(interview.InterviewType)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Interview Stage:\n%s\n\n", interview.Stage.DisplayLabel()))
	sb.WriteString(fmt.Sprintf("Candidate:\n%s\n\n", candidateName))
	sb.WriteString(fmt.Sprintf("Position:\n%s\n\n", jobTitle))
	sb.WriteString(fmt.Sprintf("Interview Type:\n%s\n\n", typeStr))

	if interview.InterviewType == model.InterviewTypeOnline && interview.MeetingURL != nil && *interview.MeetingURL != "" {
		sb.WriteString(fmt.Sprintf("Meeting Link:\n%s\n\n", *interview.MeetingURL))
	} else if interview.InterviewType == model.InterviewTypeOnsite && interview.Location != nil && *interview.Location != "" {
		sb.WriteString(fmt.Sprintf("Location:\n%s\n\n", *interview.Location))
	}

	baseURL := "http://localhost:5173"
	if s.cfg != nil && s.cfg.EmailBaseURL != "" {
		baseURL = s.cfg.EmailBaseURL
	}
	sb.WriteString(fmt.Sprintf("HireScope:\n%s/interviews/%s", baseURL, interview.ID))

	var attendees []calendar.CalendarAttendee
	if candidateEmail != "" {
		attendees = append(attendees, calendar.CalendarAttendee{
			Name:  candidateName,
			Email: candidateEmail,
		})
	}

	for _, ii := range interview.Interviewers {
		if ii.User != nil && ii.User.Email != "" {
			attendees = append(attendees, calendar.CalendarAttendee{
				Name:  ii.User.Name,
				Email: ii.User.Email,
			})
		}
	}

	loc := ""
	meetingURL := ""
	if interview.InterviewType == model.InterviewTypeOnline && interview.MeetingURL != nil {
		meetingURL = *interview.MeetingURL
		loc = *interview.MeetingURL
	} else if interview.Location != nil {
		loc = *interview.Location
	}

	tz := interview.Timezone
	if tz == "" {
		tz = "Asia/Jakarta"
	}

	return &calendar.CalendarEventData{
		Title:       title,
		Description: sb.String(),
		Start:       interview.ScheduledStart,
		End:         interview.ScheduledEnd,
		Timezone:    tz,
		Location:    loc,
		MeetingURL:  meetingURL,
		Attendees:   attendees,
	}
}

func (s *calendarIntegrationService) recordAudit(ctx context.Context, action model.AuditAction, actorID string, interview *model.Interview, provider model.CalendarProviderType, status model.CalendarSyncStatus) {
	meta, _ := json.Marshal(map[string]interface{}{
		"provider":     provider,
		"interview_id": interview.ID,
		"sync_status":  status,
	})
	_ = s.auditRepo.Create(ctx, &model.AuditLog{
		Action:      action,
		ActorID:     actorID,
		JobID:       &interview.JobID,
		CandidateID: &interview.CandidateID,
		Metadata:    string(meta),
		CreatedAt:   time.Now().UTC(),
	})
}

func (s *calendarIntegrationService) recordAuditSyncFailed(ctx context.Context, actorID string, interview *model.Interview, provider model.CalendarProviderType, errReason string) {
	meta, _ := json.Marshal(map[string]interface{}{
		"provider":     provider,
		"interview_id": interview.ID,
		"sync_status":  model.CalendarSyncStatusFailed,
		"error":        errReason,
	})
	_ = s.auditRepo.Create(ctx, &model.AuditLog{
		Action:      model.AuditActionCalendarSyncFailed,
		ActorID:     actorID,
		JobID:       &interview.JobID,
		CandidateID: &interview.CandidateID,
		Metadata:    string(meta),
		CreatedAt:   time.Now().UTC(),
	})
}
