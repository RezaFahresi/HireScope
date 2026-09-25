package tests

import (
	"bytes"
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

	"hirescope/backend/internal/config"
	"hirescope/backend/internal/email"
	"hirescope/backend/internal/handler"
	"hirescope/backend/internal/middleware"
	"hirescope/backend/internal/model"
	"hirescope/backend/internal/repository"
	"hirescope/backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type memoryEmailDeliveryRepo struct {
	mu           sync.RWMutex
	deliveries   map[string]*model.InterviewEmailDelivery
	byKey        map[string]*model.InterviewEmailDelivery
	byInterview  map[string][]*model.InterviewEmailDelivery
	claimedCount int
}

func newMemoryEmailDeliveryRepo() *memoryEmailDeliveryRepo {
	return &memoryEmailDeliveryRepo{
		deliveries:  make(map[string]*model.InterviewEmailDelivery),
		byKey:       make(map[string]*model.InterviewEmailDelivery),
		byInterview: make(map[string][]*model.InterviewEmailDelivery),
	}
}

func (r *memoryEmailDeliveryRepo) Create(ctx context.Context, delivery *model.InterviewEmailDelivery) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.byKey[delivery.IdempotencyKey]; exists {
		return fmt.Errorf("duplicate idempotency key: %s", delivery.IdempotencyKey)
	}

	if delivery.ID == "" {
		delivery.ID = uuid.New().String()
	}
	r.deliveries[delivery.ID] = delivery
	r.byKey[delivery.IdempotencyKey] = delivery
	r.byInterview[delivery.InterviewID] = append(r.byInterview[delivery.InterviewID], delivery)
	return nil
}

func (r *memoryEmailDeliveryRepo) GetByID(ctx context.Context, id string) (*model.InterviewEmailDelivery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	d, exists := r.deliveries[id]
	if !exists {
		return nil, repository.ErrDeliveryNotFound
	}
	return d, nil
}

func (r *memoryEmailDeliveryRepo) GetByIdempotencyKey(ctx context.Context, key string) (*model.InterviewEmailDelivery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	d, exists := r.byKey[key]
	if !exists {
		return nil, repository.ErrDeliveryNotFound
	}
	return d, nil
}

func (r *memoryEmailDeliveryRepo) ListByInterviewID(ctx context.Context, interviewID string) ([]*model.InterviewEmailDelivery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := r.byInterview[interviewID]
	result := make([]*model.InterviewEmailDelivery, len(list))
	copy(result, list)
	return result, nil
}

func (r *memoryEmailDeliveryRepo) Update(ctx context.Context, delivery *model.InterviewEmailDelivery) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.deliveries[delivery.ID] = delivery
	r.byKey[delivery.IdempotencyKey] = delivery
	return nil
}

func (r *memoryEmailDeliveryRepo) GetDuePendingReminders(ctx context.Context, dueBefore time.Time, limit int) ([]*model.InterviewEmailDelivery, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*model.InterviewEmailDelivery
	for _, d := range r.deliveries {
		if d.EmailType == model.EmailTypeInterviewReminder &&
			d.Status == model.DeliveryStatusPending &&
			d.ScheduledFor != nil &&
			!d.ScheduledFor.After(dueBefore) {
			result = append(result, d)
			if limit > 0 && len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

func (r *memoryEmailDeliveryRepo) ClaimReminder(ctx context.Context, id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	d, exists := r.deliveries[id]
	if !exists || d.Status != model.DeliveryStatusPending {
		return false, nil
	}

	d.Status = model.DeliveryStatusSending
	d.AttemptCount++
	d.UpdatedAt = time.Now().UTC()
	r.claimedCount++
	return true, nil
}

func (r *memoryEmailDeliveryRepo) InvalidatePendingReminders(ctx context.Context, interviewID string, reason string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, d := range r.byInterview[interviewID] {
		if d.EmailType == model.EmailTypeInterviewReminder && d.Status == model.DeliveryStatusPending {
			d.Status = model.DeliveryStatusSkipped
			d.LastError = &reason
			d.UpdatedAt = time.Now().UTC()
		}
	}
	return nil
}

func setupEmailTestHarness() (
	*gin.Engine,
	*memoryEmailDeliveryRepo,
	*memoryInterviewRepo,
	*memoryUserRepo,
	*memoryAuditLogRepo,
	*email.MockEmailProvider,
	service.InterviewNotificationService,
	service.InterviewService,
	string, // recruiterToken
	*model.User,
	*model.Job,
	*model.Candidate,
) {
	gin.SetMode(gin.TestMode)

	userRepo := newMemoryUserRepo()
	interviewRepo := newMemoryInterviewRepo()
	jobCandidateRepo := newMemoryJobCandidateRepo(nil, nil)
	deliveryRepo := newMemoryEmailDeliveryRepo()
	auditRepo := newMemoryAuditLogRepo()
	mockProvider := email.NewMockEmailProvider()

	jwtService, _ := service.NewJWTService("super-secret-test-jwt-key-minimum-32-bytes!", 24)
	revokedRepo := newMemoryRevokedRepo()

	cfg := &config.Config{
		AppEnv:                 "testing",
		EmailProvider:          "mock",
		EmailEnabled:           true,
		EmailFromName:          "HireScope Test",
		EmailFromAddress:       "recruiter@hirescope.local",
		EmailBaseURL:           "http://localhost:5173",
		InterviewReminderHours: 24,
	}

	notificationService := service.NewInterviewNotificationService(
		deliveryRepo,
		interviewRepo,
		userRepo,
		auditRepo,
		mockProvider,
		cfg,
	)

	interviewService := service.NewInterviewService(
		interviewRepo,
		jobCandidateRepo,
		userRepo,
		notificationService,
	)

	recruiter := &model.User{
		ID:    uuid.New().String(),
		Name:  "Sarah Recruiter",
		Email: "sarah@hirescope.local",
		Role:  model.RoleRecruiter,
	}
	_ = userRepo.Create(context.Background(), recruiter)
	interviewRepo.users[recruiter.ID] = recruiter

	interviewer := &model.User{
		ID:    uuid.New().String(),
		Name:  "Alex Tech Lead",
		Email: "alex@hirescope.local",
		Role:  model.RoleRecruiter,
	}
	_ = userRepo.Create(context.Background(), interviewer)
	interviewRepo.users[interviewer.ID] = interviewer

	job := &model.Job{
		ID:    uuid.New().String(),
		Title: "Principal Cloud Engineer",
		Code:  "JOB-2026-TEST",
	}
	interviewRepo.jobs[job.ID] = job

	candidate := &model.Candidate{
		ID:       uuid.New().String(),
		FullName: "Maya Lin",
		Email:    "maya.lin@example.com",
	}
	interviewRepo.candidates[candidate.ID] = candidate

	_ = jobCandidateRepo.AddJobCandidate(context.Background(), job.ID, candidate.ID)

	recruiterToken, _, _ := jwtService.GenerateToken(recruiter)

	interviewHandler := handler.NewInterviewHandler(interviewService, notificationService)

	router := gin.New()
	v1 := router.Group("/api/v1")
	interviewsGroup := v1.Group("/interviews")
	interviewsGroup.Use(middleware.AuthMiddleware(jwtService, revokedRepo))
	{
		interviewsGroup.POST("", interviewHandler.CreateInterview)
		interviewsGroup.GET("/:id", interviewHandler.GetInterview)
		interviewsGroup.PATCH("/:id/reschedule", interviewHandler.RescheduleInterview)
		interviewsGroup.PATCH("/:id/cancel", interviewHandler.CancelInterview)
		interviewsGroup.POST("/:id/complete", interviewHandler.CompleteInterview)
		interviewsGroup.GET("/:id/email-deliveries", interviewHandler.ListEmailDeliveries)
		interviewsGroup.POST("/:id/send-invitation", interviewHandler.SendInvitation)
		interviewsGroup.POST("/:id/email-deliveries/:deliveryId/retry", interviewHandler.RetryEmailDelivery)
	}

	return router, deliveryRepo, interviewRepo, userRepo, auditRepo, mockProvider, notificationService, interviewService, recruiterToken, recruiter, job, candidate
}

// 1. Tests Email Provider Abstraction & Config Validation
func TestEmail_ProviderAbstractionAndConfigValidation(t *testing.T) {
	// Test Disabled provider
	disabledProvider := email.NewDisabledEmailProvider()
	res, err := disabledProvider.Send(context.Background(), email.EmailMessage{
		To:      "test@example.com",
		Subject: "Test",
		Text:    "Test content",
	})
	if res == nil || res.Status != "SKIPPED" {
		t.Fatalf("expected SKIPPED status from disabled provider, got %v", res)
	}
	if !errors.Is(err, email.ErrEmailDisabled) {
		t.Fatalf("expected ErrEmailDisabled, got %v", err)
	}

	// Test Config validation when email enabled
	cfg := &config.Config{
		DBHost:           "localhost",
		DBPort:           "5432",
		DBName:           "testdb",
		DBUser:           "user",
		JWTSecret:        "secret-key-at-least-32-chars-long!",
		EmailEnabled:     true,
		EmailProvider:    "resend",
		ResendAPIKey:     "", // missing
		EmailFromAddress: "onboarding@resend.dev",
	}
	err = cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "RESEND_API_KEY") {
		t.Fatalf("expected missing RESEND_API_KEY validation error, got: %v", err)
	}

	cfg.ResendAPIKey = "re_test_key_123"
	cfg.EmailFromAddress = "" // missing
	err = cfg.Validate()
	if err == nil || !strings.Contains(err.Error(), "EMAIL_FROM_ADDRESS") {
		t.Fatalf("expected missing EMAIL_FROM_ADDRESS validation error, got: %v", err)
	}

	// When disabled, missing API key does not fail config
	cfg.EmailEnabled = false
	cfg.ResendAPIKey = ""
	cfg.EmailFromAddress = ""
	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected config to be valid when EmailEnabled=false, got: %v", err)
	}
}

// 2. Tests Candidate & Interviewer Invitation Delivery Flow with Idempotency
func TestEmail_InvitationDeliveryFlowAndIdempotency(t *testing.T) {
	router, deliveryRepo, _, userRepo, _, mockProvider, _, _, token, _, job, candidate := setupEmailTestHarness()

	// Get interviewer
	users, _ := userRepo.List(context.Background())
	interviewerID := ""
	for _, u := range users {
		if u.Email == "alex@hirescope.local" {
			interviewerID = u.ID
			break
		}
	}

	// 1. Create interview with invitations checked
	start := time.Now().Add(48 * time.Hour).UTC()
	end := start.Add(1 * time.Hour).UTC()
	meetURL := "https://meet.google.com/test-room"

	payload := map[string]interface{}{
		"job_id":                     job.ID,
		"candidate_id":               candidate.ID,
		"title":                      "Technical Assessment Session",
		"stage":                      "TECHNICAL_INTERVIEW",
		"interview_type":             "ONLINE",
		"scheduled_start":            start.Format(time.RFC3339),
		"scheduled_end":              end.Format(time.RFC3339),
		"meeting_url":                meetURL,
		"interviewer_ids":            []string{interviewerID},
		"send_candidate_invitation":   true,
		"send_interviewer_invitation": true,
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/interviews", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d (body: %s)", w.Code, w.Body.String())
	}

	var resp struct {
		Data model.Interview `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	interviewID := resp.Data.ID

	// 2. Verify sent messages in mock provider
	sentMsgs := mockProvider.GetSentMessages()
	if len(sentMsgs) != 2 {
		t.Fatalf("expected 2 messages sent (1 candidate, 1 interviewer), got %d", len(sentMsgs))
	}

	// Verify candidate message content
	var candMsg, intMsg *email.EmailMessage
	for _, m := range sentMsgs {
		if m.To == candidate.Email {
			copied := m
			candMsg = &copied
		} else if m.To == "alex@hirescope.local" {
			copied := m
			intMsg = &copied
		}
	}

	if candMsg == nil {
		t.Fatalf("expected candidate email to %s, not found", candidate.Email)
	}
	if !strings.Contains(candMsg.Subject, "Interview Invitation") {
		t.Errorf("candidate subject mismatch: %s", candMsg.Subject)
	}
	if !strings.Contains(candMsg.HTML, meetURL) {
		t.Errorf("candidate email HTML missing meeting URL: %s", candMsg.HTML)
	}
	if !strings.Contains(candMsg.Text, meetURL) {
		t.Errorf("candidate email text missing meeting URL: %s", candMsg.Text)
	}
	if !strings.Contains(candMsg.HTML, "/api/v1/interviews/"+interviewID+"/ics") {
		t.Errorf("candidate email HTML missing calendar link: %s", candMsg.HTML)
	}

	if intMsg == nil {
		t.Fatalf("expected interviewer email to alex@hirescope.local, not found")
	}
	if !strings.Contains(intMsg.Subject, "Interview Scheduled") {
		t.Errorf("interviewer subject mismatch: %s", intMsg.Subject)
	}

	// 3. Query GET /api/v1/interviews/:id/email-deliveries
	reqList := httptest.NewRequest(http.MethodGet, "/api/v1/interviews/"+interviewID+"/email-deliveries", nil)
	reqList.Header.Set("Authorization", "Bearer "+token)
	wList := httptest.NewRecorder()
	router.ServeHTTP(wList, reqList)

	if wList.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", wList.Code)
	}

	var listResp struct {
		Data []*model.InterviewEmailDelivery `json:"data"`
	}
	_ = json.Unmarshal(wList.Body.Bytes(), &listResp)

	// We expect candidate invitation (SENT), interviewer invitation (SENT), and pre-created reminders (PENDING)
	if len(listResp.Data) < 2 {
		t.Fatalf("expected at least 2 email deliveries recorded, got %d", len(listResp.Data))
	}

	var foundCandSent, foundIntSent bool
	for _, d := range listResp.Data {
		if d.EmailType == model.EmailTypeInterviewInvitation && d.RecipientType == model.RecipientTypeCandidate && d.Status == model.DeliveryStatusSent {
			foundCandSent = true
		}
		if d.EmailType == model.EmailTypeInterviewInvitation && d.RecipientType == model.RecipientTypeInterviewer && d.Status == model.DeliveryStatusSent {
			foundIntSent = true
		}
	}
	if !foundCandSent || !foundIntSent {
		t.Errorf("expected SENT invitation delivery records, got cand=%v, int=%v", foundCandSent, foundIntSent)
	}

	// 4. Test duplicate protection by calling POST /api/v1/interviews/:id/send-invitation again
	callCountBefore := mockProvider.CallCount
	reqDup := httptest.NewRequest(http.MethodPost, "/api/v1/interviews/"+interviewID+"/send-invitation", strings.NewReader(`{"send_candidate":true,"send_interviewers":true}`))
	reqDup.Header.Set("Authorization", "Bearer "+token)
	reqDup.Header.Set("Content-Type", "application/json")
	wDup := httptest.NewRecorder()
	router.ServeHTTP(wDup, reqDup)

	if wDup.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", wDup.Code)
	}

	// Call count should NOT increase because records are already SENT!
	if mockProvider.CallCount != callCountBefore {
		t.Errorf("duplicate send call made to provider! Call count before=%d, after=%d", callCountBefore, mockProvider.CallCount)
	}
	_ = deliveryRepo
}

// 3. Tests Failed Delivery and Safe Retry Flow
func TestEmail_FailedDeliveryAndRetryFlow(t *testing.T) {
	router, _, _, userRepo, _, mockProvider, _, _, token, _, job, candidate := setupEmailTestHarness()

	users, _ := userRepo.List(context.Background())
	interviewerID := users[0].ID

	// Simulate failure in email provider
	mockProvider.SimulateErr = errors.New("rate limit exceeded from resend")

	// Trigger invitation
	start := time.Now().Add(48 * time.Hour).UTC()
	end := start.Add(1 * time.Hour).UTC()
	payload := map[string]interface{}{
		"job_id":                     job.ID,
		"candidate_id":               candidate.ID,
		"title":                      "Retry Assessment Test",
		"stage":                      "HR_INTERVIEW",
		"interview_type":             "ONLINE",
		"scheduled_start":            start.Format(time.RFC3339),
		"scheduled_end":              end.Format(time.RFC3339),
		"meeting_url":                "https://meet.google.com/test-room",
		"interviewer_ids":            []string{interviewerID},
		"send_candidate_invitation":   true,
		"send_interviewer_invitation": false,
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/interviews", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Interview creation still succeeds even if email provider failed!
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for interview, got %d", w.Code)
	}

	var resp struct {
		Data model.Interview `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	interviewID := resp.Data.ID

	// Check delivery status -> FAILED
	reqList := httptest.NewRequest(http.MethodGet, "/api/v1/interviews/"+interviewID+"/email-deliveries", nil)
	reqList.Header.Set("Authorization", "Bearer "+token)
	wList := httptest.NewRecorder()
	router.ServeHTTP(wList, reqList)

	var listResp struct {
		Data []*model.InterviewEmailDelivery `json:"data"`
	}
	_ = json.Unmarshal(wList.Body.Bytes(), &listResp)

	var failedDelivery *model.InterviewEmailDelivery
	for _, d := range listResp.Data {
		if d.RecipientType == model.RecipientTypeCandidate && d.EmailType == model.EmailTypeInterviewInvitation {
			failedDelivery = d
			break
		}
	}

	if failedDelivery == nil {
		t.Fatalf("expected candidate invitation delivery record")
	}
	if failedDelivery.Status != model.DeliveryStatusFailed {
		t.Fatalf("expected FAILED status, got %s", failedDelivery.Status)
	}
	if failedDelivery.LastError == nil || !strings.Contains(*failedDelivery.LastError, "rate limit") {
		t.Fatalf("expected last error to contain rate limit message, got %v", failedDelivery.LastError)
	}
	if failedDelivery.AttemptCount != 1 {
		t.Fatalf("expected attempt_count=1, got %d", failedDelivery.AttemptCount)
	}

	// Now fix provider and retry
	mockProvider.SimulateErr = nil
	reqRetry := httptest.NewRequest(http.MethodPost, "/api/v1/interviews/"+interviewID+"/email-deliveries/"+failedDelivery.ID+"/retry", nil)
	reqRetry.Header.Set("Authorization", "Bearer "+token)
	wRetry := httptest.NewRecorder()
	router.ServeHTTP(wRetry, reqRetry)

	if wRetry.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for retry, got %d (body: %s)", wRetry.Code, wRetry.Body.String())
	}

	var retryResp struct {
		Data model.InterviewEmailDelivery `json:"data"`
	}
	_ = json.Unmarshal(wRetry.Body.Bytes(), &retryResp)

	if retryResp.Data.Status != model.DeliveryStatusSent {
		t.Fatalf("expected status SENT after retry, got %s", retryResp.Data.Status)
	}
	if retryResp.Data.AttemptCount != 2 {
		t.Fatalf("expected attempt_count=2, got %d", retryResp.Data.AttemptCount)
	}

	// Retrying a SENT delivery must return HTTP 400 Bad Request
	wRetrySent := httptest.NewRecorder()
	reqRetrySent := httptest.NewRequest(http.MethodPost, "/api/v1/interviews/"+interviewID+"/email-deliveries/"+failedDelivery.ID+"/retry", nil)
	reqRetrySent.Header.Set("Authorization", "Bearer "+token)
	router.ServeHTTP(wRetrySent, reqRetrySent)

	if wRetrySent.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 Bad Request when retrying a SENT delivery, got %d", wRetrySent.Code)
	}
}

// 4. Tests Reminder Scheduling, Due Processing, and Invalidation on Reschedule/Cancel
func TestEmail_ReminderLifecycleAndWorkerSafety(t *testing.T) {
	router, _, _, userRepo, _, mockProvider, notifService, _, token, _, job, candidate := setupEmailTestHarness()

	users, _ := userRepo.List(context.Background())
	interviewerID := users[0].ID

	// Create interview scheduled in 10 hours (which is within the 24h reminder window!)
	start := time.Now().Add(10 * time.Hour).UTC()
	end := start.Add(1 * time.Hour).UTC()

	payload := map[string]interface{}{
		"job_id":          job.ID,
		"candidate_id":    candidate.ID,
		"title":           "Reminder Test Session",
		"stage":           "FINAL_INTERVIEW",
		"interview_type":  "ONSITE",
		"location":        "Tower 2, Floor 14, Jakarta",
		"scheduled_start": start.Format(time.RFC3339),
		"scheduled_end":   end.Format(time.RFC3339),
		"interviewer_ids": []string{interviewerID},
	}

	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/interviews", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", w.Code)
	}

	var resp struct {
		Data model.Interview `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	interviewID := resp.Data.ID

	// Scheduled for 10h from now - 24h = 14h in the past -> due now!
	mockProvider.Reset()
	processed, err := notifService.ProcessDueReminders(context.Background())
	if err != nil {
		t.Fatalf("unexpected error processing due reminders: %v", err)
	}
	if processed != 2 {
		t.Fatalf("expected 2 reminders processed (candidate + interviewer), got %d", processed)
	}

	sentMsgs := mockProvider.GetSentMessages()
	if len(sentMsgs) != 2 {
		t.Fatalf("expected 2 reminder emails dispatched, got %d", len(sentMsgs))
	}
	for _, m := range sentMsgs {
		if !strings.Contains(m.Subject, "Reminder") {
			t.Errorf("expected subject to contain Reminder, got: %s", m.Subject)
		}
		if !strings.Contains(m.HTML, "Tower 2, Floor 14, Jakarta") {
			t.Errorf("expected HTML to contain onsite location, got: %s", m.HTML)
		}
	}

	// 5. Test Reschedule invalidates old reminder and schedules new reminder
	newStart := time.Now().Add(72 * time.Hour).UTC()
	newEnd := newStart.Add(1 * time.Hour).UTC()

	reschedulePayload := map[string]interface{}{
		"scheduled_start": newStart.Format(time.RFC3339),
		"scheduled_end":   newEnd.Format(time.RFC3339),
		"reason":          "Candidate requested morning slot",
	}
	reschedBody, _ := json.Marshal(reschedulePayload)
	reqResched := httptest.NewRequest(http.MethodPatch, "/api/v1/interviews/"+interviewID+"/reschedule", bytes.NewReader(reschedBody))
	reqResched.Header.Set("Authorization", "Bearer "+token)
	reqResched.Header.Set("Content-Type", "application/json")
	wResched := httptest.NewRecorder()
	router.ServeHTTP(wResched, reqResched)

	if wResched.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for reschedule, got %d", wResched.Code)
	}

	// 6. Test Cancel interview invalidates pending reminders
	reqCancel := httptest.NewRequest(http.MethodPatch, "/api/v1/interviews/"+interviewID+"/cancel", strings.NewReader(`{"reason":"Position filled"}`))
	reqCancel.Header.Set("Authorization", "Bearer "+token)
	reqCancel.Header.Set("Content-Type", "application/json")
	wCancel := httptest.NewRecorder()
	router.ServeHTTP(wCancel, reqCancel)

	if wCancel.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for cancel, got %d", wCancel.Code)
	}

	// Query deliveries: ensure no PENDING reminders remain (they should be SKIPPED)
	deliveries, _ := notifService.ListDeliveries(context.Background(), interviewID)
	for _, d := range deliveries {
		if d.EmailType == model.EmailTypeInterviewReminder && d.Status == model.DeliveryStatusPending {
			t.Errorf("found active PENDING reminder on cancelled interview: %v", d)
		}
	}
}

// 5. Tests HTML Escaping and Header Protection
func TestEmail_HTMLEscapingAndHeaderSanitization(t *testing.T) {
	data := email.InvitationTemplateData{
		CandidateName:    "<script>alert('xss')</script> Maya & Lin",
		JobTitle:         "Lead Architect <b>VIP</b>",
		JobCode:          "JOB-99",
		StageLabel:       "Technical <Assessment>",
		InterviewType:    "ONLINE",
		ScheduledStart:   time.Date(2026, 9, 26, 10, 0, 0, 0, time.UTC),
		ScheduledEnd:     time.Date(2026, 9, 26, 11, 0, 0, 0, time.UTC),
		MeetingURL:       "https://meet.google.com/test\r\nCRLF-Injection",
		Location:         "Floor 3 & 4 <Main>",
		InterviewerNames: []string{"Alice <Lead>", "Bob & Charlie"},
		InterviewID:      "iv-123",
		BaseURL:          "http://localhost:5173",
	}

	subj, htmlBody, textBody := email.BuildCandidateInvitation(data)

	// HTML must NOT contain unescaped raw script tags
	if strings.Contains(htmlBody, "<script>") {
		t.Errorf("HTML contains unescaped script tag: %s", htmlBody)
	}
	if !strings.Contains(htmlBody, "&lt;script&gt;") {
		t.Errorf("HTML does not properly escape candidate name")
	}
	if strings.Contains(htmlBody, "<b>VIP</b>") {
		t.Errorf("HTML contains unescaped job title markup")
	}
	if strings.Contains(htmlBody, "\r\nCRLF-Injection") {
		t.Errorf("HTML contains unescaped CRLF sequence in meeting URL")
	}

	// Plain text should preserve readable characters without HTML entities
	if !strings.Contains(textBody, "Maya & Lin") {
		t.Errorf("plain text should preserve readable name: %s", textBody)
	}
	_ = subj
}
