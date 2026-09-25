package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"hirescope/backend/internal/handler"
	"hirescope/backend/internal/middleware"
	"hirescope/backend/internal/model"
	"hirescope/backend/internal/repository"
	"hirescope/backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type memoryInterviewRepo struct {
	mu           sync.RWMutex
	interviews   map[string]*model.Interview
	interviewers map[string][]model.InterviewInterviewer // key: interview_id
	auditLogs    []model.AuditLog
	users        map[string]*model.User
	jobs         map[string]*model.Job
	candidates   map[string]*model.Candidate
}

func newMemoryInterviewRepo() *memoryInterviewRepo {
	return &memoryInterviewRepo{
		interviews:   make(map[string]*model.Interview),
		interviewers: make(map[string][]model.InterviewInterviewer),
		users:        make(map[string]*model.User),
		jobs:         make(map[string]*model.Job),
		candidates:   make(map[string]*model.Candidate),
	}
}

func (r *memoryInterviewRepo) Create(ctx context.Context, interview *model.Interview, interviewerIDs []string, audit *model.AuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.interviews[interview.ID] = interview

	var ivs []model.InterviewInterviewer
	for _, uid := range interviewerIDs {
		ii := model.InterviewInterviewer{
			ID:          uuid.New().String(),
			InterviewID: interview.ID,
			UserID:      uid,
			CreatedAt:   time.Now().UTC(),
		}
		if u, exists := r.users[uid]; exists {
			ii.User = u
		}
		ivs = append(ivs, ii)
	}
	r.interviewers[interview.ID] = ivs

	if audit != nil {
		r.auditLogs = append(r.auditLogs, *audit)
	}
	return nil
}

func (r *memoryInterviewRepo) GetByID(ctx context.Context, id string) (*model.Interview, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	item, exists := r.interviews[id]
	if !exists {
		return nil, repository.ErrInterviewNotFound
	}

	copyItem := *item
	if j, ok := r.jobs[copyItem.JobID]; ok {
		copyItem.Job = j
	}
	if c, ok := r.candidates[copyItem.CandidateID]; ok {
		copyItem.Candidate = c
	}
	if u, ok := r.users[copyItem.CreatedBy]; ok {
		copyItem.Creator = u
	}
	if copyItem.CompletedBy != nil {
		if u, ok := r.users[*copyItem.CompletedBy]; ok {
			copyItem.CompletedUser = u
		}
	}
	if copyItem.CancelledBy != nil {
		if u, ok := r.users[*copyItem.CancelledBy]; ok {
			copyItem.CancelledUser = u
		}
	}

	copyItem.Interviewers = r.interviewers[id]
	return &copyItem, nil
}

func (r *memoryInterviewRepo) Update(ctx context.Context, interview *model.Interview, interviewerIDs []string, audit *model.AuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.interviews[interview.ID] = interview

	if interviewerIDs != nil {
		var ivs []model.InterviewInterviewer
		for _, uid := range interviewerIDs {
			ii := model.InterviewInterviewer{
				ID:          uuid.New().String(),
				InterviewID: interview.ID,
				UserID:      uid,
				CreatedAt:   time.Now().UTC(),
			}
			if u, exists := r.users[uid]; exists {
				ii.User = u
			}
			ivs = append(ivs, ii)
		}
		r.interviewers[interview.ID] = ivs
	}

	if audit != nil {
		r.auditLogs = append(r.auditLogs, *audit)
	}
	return nil
}

func (r *memoryInterviewRepo) UpdateStatus(ctx context.Context, interview *model.Interview, audit *model.AuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.interviews[interview.ID] = interview
	if audit != nil {
		r.auditLogs = append(r.auditLogs, *audit)
	}
	return nil
}

func (r *memoryInterviewRepo) List(ctx context.Context, filter repository.InterviewFilter) (*repository.InterviewListResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matched []model.Interview
	for _, item := range r.interviews {
		if filter.JobID != "" && item.JobID != filter.JobID {
			continue
		}
		if filter.CandidateID != "" && item.CandidateID != filter.CandidateID {
			continue
		}
		if filter.Status != "" && string(item.Status) != filter.Status {
			continue
		}
		if filter.Stage != "" && string(item.Stage) != filter.Stage {
			continue
		}
		if filter.InterviewType != "" && string(item.InterviewType) != filter.InterviewType {
			continue
		}
		if filter.From != nil && item.ScheduledStart.Before(*filter.From) {
			continue
		}
		if filter.To != nil && item.ScheduledStart.After(*filter.To) {
			continue
		}

		copyItem := *item
		copyItem.Interviewers = r.interviewers[item.ID]
		matched = append(matched, copyItem)
	}

	sort.Slice(matched, func(i, j int) bool {
		return matched[i].ScheduledStart.Before(matched[j].ScheduledStart)
	})

	total := int64(len(matched))
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 {
		limit = 10
	}

	start := (page - 1) * limit
	if start > len(matched) {
		start = len(matched)
	}
	end := start + limit
	if end > len(matched) {
		end = len(matched)
	}

	items := matched[start:end]
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	if totalPages == 0 {
		totalPages = 1
	}

	return &repository.InterviewListResult{
		Items:      items,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (r *memoryInterviewRepo) CheckInterviewerConflict(ctx context.Context, interviewerIDs []string, start, end time.Time, excludeInterviewID string) (string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	interviewerSet := make(map[string]bool)
	for _, id := range interviewerIDs {
		interviewerSet[id] = true
	}

	for _, interview := range r.interviews {
		if interview.ID == excludeInterviewID {
			continue
		}
		if interview.Status != model.InterviewStatusScheduled {
			continue
		}

		// Overlap check
		if interview.ScheduledStart.Before(end) && interview.ScheduledEnd.After(start) {
			assigned := r.interviewers[interview.ID]
			for _, iv := range assigned {
				if interviewerSet[iv.UserID] {
					if u, ok := r.users[iv.UserID]; ok {
						return u.Name, nil
					}
					return iv.UserID, nil
				}
			}
		}
	}

	return "", nil
}

func (r *memoryInterviewRepo) GetCandidateNextInterview(ctx context.Context, candidateID, jobID string) (*model.Interview, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	now := time.Now().UTC()
	var upcoming []*model.Interview

	for _, item := range r.interviews {
		if item.CandidateID != candidateID {
			continue
		}
		if jobID != "" && item.JobID != jobID {
			continue
		}
		if item.Status != model.InterviewStatusScheduled {
			continue
		}
		if item.ScheduledEnd.Before(now) {
			continue
		}
		upcoming = append(upcoming, item)
	}

	if len(upcoming) == 0 {
		return nil, nil
	}

	sort.Slice(upcoming, func(i, j int) bool {
		return upcoming[i].ScheduledStart.Before(upcoming[j].ScheduledStart)
	})

	copyItem := *upcoming[0]
	copyItem.Interviewers = r.interviewers[copyItem.ID]
	return &copyItem, nil
}

func (r *memoryInterviewRepo) ListByCandidateAndJob(ctx context.Context, candidateID, jobID string) ([]model.Interview, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []model.Interview
	for _, item := range r.interviews {
		if item.CandidateID != candidateID {
			continue
		}
		if jobID != "" && item.JobID != jobID {
			continue
		}
		copyItem := *item
		copyItem.Interviewers = r.interviewers[item.ID]
		result = append(result, copyItem)
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].ScheduledStart.After(result[j].ScheduledStart)
	})

	return result, nil
}

// Test harness setup
func setupInterviewTestRouter() (
	*gin.Engine,
	*memoryInterviewRepo,
	*memoryJobCandidateRepo,
	*memoryUserRepo,
	string, // recruiter token
	*model.User, // recruiter
	*model.Job,
	*model.Candidate,
) {
	gin.SetMode(gin.TestMode)

	userRepo := newMemoryUserRepo()
	jwtService, _ := service.NewJWTService("super-secret-test-jwt-key-minimum-32-bytes!", 24)
	revokedRepo := newMemoryRevokedRepo()

	recruiter := &model.User{
		ID:    uuid.New().String(),
		Name:  "Sarah Recruiter",
		Email: "sarah@hirescope.local",
		Role:  model.RoleRecruiter,
	}
	_ = userRepo.Create(context.Background(), recruiter)

	interviewer2 := &model.User{
		ID:    uuid.New().String(),
		Name:  "Alex Tech Lead",
		Email: "alex@hirescope.local",
		Role:  model.RoleRecruiter,
	}
	_ = userRepo.Create(context.Background(), interviewer2)

	job := &model.Job{
		ID:          uuid.New().String(),
		Code:        "JOB-2026-TEST",
		Title:       "Backend Go Engineer",
		Department:  "Engineering",
		Status:      model.JobStatusOpen,
		CreatedBy:   recruiter.ID,
		CreatedAt:   time.Now().UTC(),
	}

	candidate := &model.Candidate{
		ID:        uuid.New().String(),
		FullName:  "Maya Lin",
		Email:     "maya@test.com",
		CreatedBy: recruiter.ID,
		CreatedAt: time.Now().UTC(),
	}

	jobCandidateRepo := newMemoryJobCandidateRepo(nil, nil)
	_ = jobCandidateRepo.AddJobCandidate(context.Background(), job.ID, candidate.ID)

	interviewRepo := newMemoryInterviewRepo()
	interviewRepo.users[recruiter.ID] = recruiter
	interviewRepo.users[interviewer2.ID] = interviewer2
	interviewRepo.jobs[job.ID] = job
	interviewRepo.candidates[candidate.ID] = candidate

	interviewService := service.NewInterviewService(interviewRepo, jobCandidateRepo, userRepo)
	interviewHandler := handler.NewInterviewHandler(interviewService)

	router := gin.New()
	v1 := router.Group("/api/v1")
	{
		interviewsGroup := v1.Group("/interviews")
		interviewsGroup.Use(middleware.AuthMiddleware(jwtService, revokedRepo))
		{
			interviewsGroup.POST("", interviewHandler.CreateInterview)
			interviewsGroup.GET("", interviewHandler.ListInterviews)
			interviewsGroup.GET("/:id", interviewHandler.GetInterview)
			interviewsGroup.GET("/:id/ics", interviewHandler.ExportICS)
			interviewsGroup.PUT("/:id", interviewHandler.UpdateInterview)
			interviewsGroup.PATCH("/:id/reschedule", interviewHandler.RescheduleInterview)
			interviewsGroup.PATCH("/:id/cancel", interviewHandler.CancelInterview)
			interviewsGroup.POST("/:id/complete", interviewHandler.CompleteInterview)
		}

		candidatesGroup := v1.Group("/candidates")
		candidatesGroup.Use(middleware.AuthMiddleware(jwtService, revokedRepo))
		{
			candidatesGroup.GET("/:candidateId/interviews", interviewHandler.GetCandidateInterviews)
			candidatesGroup.GET("/:candidateId/next-interview", interviewHandler.GetCandidateNextInterview)
		}

		usersGroup := v1.Group("/users")
		usersGroup.Use(middleware.AuthMiddleware(jwtService, revokedRepo))
		{
			usersGroup.GET("", interviewHandler.ListUsers)
		}
	}

	token, _, _ := jwtService.GenerateToken(recruiter)
	return router, interviewRepo, jobCandidateRepo, userRepo, token, recruiter, job, candidate
}

func TestInterviewManagement_All25Cases(t *testing.T) {
	router, interviewRepo, _, _, token, recruiter, job, candidate := setupInterviewTestRouter()

	now := time.Now().UTC().Add(24 * time.Hour)
	startTime := now.Truncate(time.Hour)
	endTime := startTime.Add(time.Hour)

	var createdInterviewID string

	// 1. Create interview (Success)
	t.Run("1_CreateInterview_Success", func(t *testing.T) {
		meetingURL := "https://meet.google.com/abc-defg-hij"
		body := map[string]interface{}{
			"job_id":          job.ID,
			"candidate_id":    candidate.ID,
			"title":           "Technical Round 1",
			"stage":           "TECHNICAL_INTERVIEW",
			"interview_type":  "ONLINE",
			"scheduled_start": startTime.Format(time.RFC3339),
			"scheduled_end":   endTime.Format(time.RFC3339),
			"meeting_url":     meetingURL,
			"interviewer_ids": []string{recruiter.ID},
		}
		raw, _ := json.Marshal(body)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/interviews", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
		}

		var res struct {
			Data service.InterviewDTO `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		if res.Data.ID == "" || res.Data.Title != "Technical Round 1" {
			t.Fatalf("unexpected interview created response: %+v", res.Data)
		}
		if res.Data.Status != model.InterviewStatusScheduled {
			t.Fatalf("expected status SCHEDULED, got %s", res.Data.Status)
		}
		createdInterviewID = res.Data.ID
	})

	// 2. Invalid candidate/job association
	t.Run("2_InvalidCandidateJobAssociation", func(t *testing.T) {
		fakeJobID := uuid.New().String()
		body := map[string]interface{}{
			"job_id":          fakeJobID,
			"candidate_id":    candidate.ID,
			"title":           "Fake Job Interview",
			"stage":           "PHONE_SCREEN",
			"interview_type":  "PHONE",
			"scheduled_start": startTime.Add(2 * time.Hour).Format(time.RFC3339),
			"scheduled_end":   endTime.Add(2 * time.Hour).Format(time.RFC3339),
			"interviewer_ids": []string{recruiter.ID},
		}
		raw, _ := json.Marshal(body)

		req, _ := http.NewRequest(http.MethodPost, "/api/v1/interviews", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "CANDIDATE_NOT_ASSOCIATED") {
			t.Fatalf("expected error code CANDIDATE_NOT_ASSOCIATED, got: %s", w.Body.String())
		}
	})

	// 3. Invalid start/end time (end before start or duration > 8 hours)
	t.Run("3_InvalidStartEndTime", func(t *testing.T) {
		// End before start
		body := map[string]interface{}{
			"job_id":          job.ID,
			"candidate_id":    candidate.ID,
			"title":           "Time Warping Interview",
			"stage":           "PHONE_SCREEN",
			"interview_type":  "PHONE",
			"scheduled_start": endTime.Format(time.RFC3339),
			"scheduled_end":   startTime.Format(time.RFC3339),
			"interviewer_ids": []string{recruiter.ID},
		}
		raw, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/interviews", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for end before start, got %d", w.Code)
		}

		// Duration > 8h
		body["scheduled_start"] = startTime.Format(time.RFC3339)
		body["scheduled_end"] = startTime.Add(9 * time.Hour).Format(time.RFC3339)
		raw2, _ := json.Marshal(body)
		req2, _ := http.NewRequest(http.MethodPost, "/api/v1/interviews", bytes.NewReader(raw2))
		req2.Header.Set("Authorization", "Bearer "+token)
		w2 := httptest.NewRecorder()
		router.ServeHTTP(w2, req2)
		if w2.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for duration > 8h, got %d", w2.Code)
		}
	})

	// 4. Invalid interview type
	t.Run("4_InvalidInterviewType", func(t *testing.T) {
		body := map[string]interface{}{
			"job_id":          job.ID,
			"candidate_id":    candidate.ID,
			"title":           "Invalid Type Interview",
			"stage":           "PHONE_SCREEN",
			"interview_type":  "HOLOGRAM",
			"scheduled_start": startTime.Format(time.RFC3339),
			"scheduled_end":   endTime.Format(time.RFC3339),
			"interviewer_ids": []string{recruiter.ID},
		}
		raw, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/interviews", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for invalid interview type, got %d", w.Code)
		}
	})

	// 5. Missing meeting URL for ONLINE
	t.Run("5_MissingMeetingURL_ForOnline", func(t *testing.T) {
		body := map[string]interface{}{
			"job_id":          job.ID,
			"candidate_id":    candidate.ID,
			"title":           "Online Missing URL",
			"stage":           "PHONE_SCREEN",
			"interview_type":  "ONLINE",
			"meeting_url":     "",
			"scheduled_start": startTime.Add(3 * time.Hour).Format(time.RFC3339),
			"scheduled_end":   endTime.Add(3 * time.Hour).Format(time.RFC3339),
			"interviewer_ids": []string{recruiter.ID},
		}
		raw, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/interviews", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for missing meeting URL, got %d", w.Code)
		}
	})

	// 6. Missing location for ONSITE
	t.Run("6_MissingLocation_ForOnsite", func(t *testing.T) {
		body := map[string]interface{}{
			"job_id":          job.ID,
			"candidate_id":    candidate.ID,
			"title":           "Onsite Missing Location",
			"stage":           "FINAL_INTERVIEW",
			"interview_type":  "ONSITE",
			"location":        "",
			"scheduled_start": startTime.Add(4 * time.Hour).Format(time.RFC3339),
			"scheduled_end":   endTime.Add(4 * time.Hour).Format(time.RFC3339),
			"interviewer_ids": []string{recruiter.ID},
		}
		raw, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/interviews", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for missing location, got %d", w.Code)
		}
	})

	// 7. Create with PHONE (Success without meeting URL or location)
	t.Run("7_CreateWithPhone_Success", func(t *testing.T) {
		phoneStart := startTime.Add(5 * time.Hour)
		phoneEnd := endTime.Add(5 * time.Hour)
		body := map[string]interface{}{
			"job_id":          job.ID,
			"candidate_id":    candidate.ID,
			"title":           "Phone Screening Call",
			"stage":           "PHONE_SCREEN",
			"interview_type":  "PHONE",
			"scheduled_start": phoneStart.Format(time.RFC3339),
			"scheduled_end":   phoneEnd.Format(time.RFC3339),
			"interviewer_ids": []string{recruiter.ID},
		}
		raw, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/interviews", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created for PHONE, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 8. Multiple interviewers
	t.Run("8_MultipleInterviewers_Success", func(t *testing.T) {
		multiStart := startTime.Add(7 * time.Hour)
		multiEnd := endTime.Add(7 * time.Hour)
		body := map[string]interface{}{
			"job_id":          job.ID,
			"candidate_id":    candidate.ID,
			"title":           "Panel Technical Interview",
			"stage":           "TECHNICAL_INTERVIEW",
			"interview_type":  "ONLINE",
			"meeting_url":     "https://zoom.us/j/123456789",
			"scheduled_start": multiStart.Format(time.RFC3339),
			"scheduled_end":   multiEnd.Format(time.RFC3339),
			"interviewer_ids": []string{recruiter.ID, recruiter.ID}, // also tests duplicate
		}
		raw, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/interviews", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 10. Interviewer conflict (Sarah Recruiter already has interview at startTime-endTime)
	t.Run("10_InterviewerConflict_Rejected", func(t *testing.T) {
		overlapStart := startTime.Add(30 * time.Minute)
		overlapEnd := endTime.Add(30 * time.Minute)
		body := map[string]interface{}{
			"job_id":          job.ID,
			"candidate_id":    candidate.ID,
			"title":           "Conflicting Interview",
			"stage":           "HR_INTERVIEW",
			"interview_type":  "ONLINE",
			"meeting_url":     "https://meet.google.com/test",
			"scheduled_start": overlapStart.Format(time.RFC3339),
			"scheduled_end":   overlapEnd.Format(time.RFC3339),
			"interviewer_ids": []string{recruiter.ID},
		}
		raw, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/interviews", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusConflict {
			t.Fatalf("expected 409 Conflict, got %d: %s", w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), "INTERVIEWER_CONFLICT") {
			t.Fatalf("expected INTERVIEWER_CONFLICT in body, got: %s", w.Body.String())
		}
	})

	// 11. List pagination
	t.Run("11_ListPagination", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/interviews?page=1&limit=2", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}
		var res struct {
			Data service.InterviewListDTO `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		if res.Data.Limit != 2 || len(res.Data.Items) > 2 {
			t.Fatalf("pagination limit not respected: %+v", res.Data)
		}
	})

	// 12. List filtering (by status=SCHEDULED)
	t.Run("12_ListFiltering", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/interviews?status=SCHEDULED&job_id="+job.ID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}
		var res struct {
			Data service.InterviewListDTO `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		for _, item := range res.Data.Items {
			if item.Status != model.InterviewStatusScheduled {
				t.Fatalf("expected all items to have SCHEDULED status, found: %s", item.Status)
			}
		}
	})

	// 13. Get detail
	t.Run("13_GetDetail", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/interviews/"+createdInterviewID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}
		var res struct {
			Data service.InterviewDTO `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		if res.Data.ID != createdInterviewID {
			t.Fatalf("expected interview ID %s, got %s", createdInterviewID, res.Data.ID)
		}
		if res.Data.Candidate == nil || res.Data.Candidate.FullName != "Maya Lin" {
			t.Fatalf("expected Candidate Maya Lin loaded on interview detail")
		}
	})

	// 14. Update
	t.Run("14_UpdateInterview", func(t *testing.T) {
		body := map[string]interface{}{
			"title":           "Technical Round 1 (Updated Title)",
			"stage":           "TECHNICAL_INTERVIEW",
			"interview_type":  "ONLINE",
			"meeting_url":     "https://meet.google.com/updated-url",
			"scheduled_start": startTime.Format(time.RFC3339),
			"scheduled_end":   endTime.Format(time.RFC3339),
			"interviewer_ids": []string{recruiter.ID},
		}
		raw, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPut, "/api/v1/interviews/"+createdInterviewID, bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}
	})

	// 15. Reschedule
	t.Run("15_RescheduleInterview", func(t *testing.T) {
		newStart := startTime.Add(24 * time.Hour)
		newEnd := endTime.Add(24 * time.Hour)
		reason := "Candidate requested later slot"

		body := map[string]interface{}{
			"scheduled_start": newStart.Format(time.RFC3339),
			"scheduled_end":   newEnd.Format(time.RFC3339),
			"reason":          reason,
		}
		raw, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/interviews/"+createdInterviewID+"/reschedule", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}
		var res struct {
			Data service.InterviewDTO `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		if !res.Data.ScheduledStart.Equal(newStart) {
			t.Fatalf("expected rescheduled start %v, got %v", newStart, res.Data.ScheduledStart)
		}
	})

	// 16 & 17. Cancel with and without cancellation reason
	t.Run("17_Cancel_MissingReason_Fails", func(t *testing.T) {
		body := map[string]interface{}{
			"reason": "",
		}
		raw, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/interviews/"+createdInterviewID+"/cancel", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for empty cancellation reason, got %d", w.Code)
		}
	})

	t.Run("16_Cancel_Success", func(t *testing.T) {
		body := map[string]interface{}{
			"reason": "Position on hold temporarily",
		}
		raw, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/interviews/"+createdInterviewID+"/cancel", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
		}
		var res struct {
			Data service.InterviewDTO `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		if res.Data.Status != model.InterviewStatusCancelled {
			t.Fatalf("expected CANCELLED status, got %s", res.Data.Status)
		}
		if res.Data.CancellationReason == nil || *res.Data.CancellationReason != "Position on hold temporarily" {
			t.Fatalf("expected cancellation reason saved")
		}
	})

	// 18 & 19. Complete and Feedback
	t.Run("18_19_CompleteWithFeedback", func(t *testing.T) {
		// Create a fresh interview to complete
		compStart := startTime.Add(48 * time.Hour)
		compEnd := endTime.Add(48 * time.Hour)
		body := map[string]interface{}{
			"job_id":          job.ID,
			"candidate_id":    candidate.ID,
			"title":           "Manager Assessment",
			"stage":           "MANAGER_INTERVIEW",
			"interview_type":  "ONLINE",
			"meeting_url":     "https://meet.google.com/comp-test",
			"scheduled_start": compStart.Format(time.RFC3339),
			"scheduled_end":   compEnd.Format(time.RFC3339),
			"interviewer_ids": []string{recruiter.ID},
		}
		raw, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/interviews", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		var res struct {
			Data service.InterviewDTO `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		newInterviewID := res.Data.ID

		// Complete interview
		feedback := "Candidate demonstrated strong communication and leadership qualities."
		completeBody := map[string]interface{}{
			"result":   "PASSED",
			"feedback": feedback,
		}
		compRaw, _ := json.Marshal(completeBody)
		compReq, _ := http.NewRequest(http.MethodPost, "/api/v1/interviews/"+newInterviewID+"/complete", bytes.NewReader(compRaw))
		compReq.Header.Set("Authorization", "Bearer "+token)
		compW := httptest.NewRecorder()
		router.ServeHTTP(compW, compReq)

		if compW.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d: %s", compW.Code, compW.Body.String())
		}
		var compRes struct {
			Data service.InterviewDTO `json:"data"`
		}
		_ = json.Unmarshal(compW.Body.Bytes(), &compRes)
		if compRes.Data.Status != model.InterviewStatusCompleted {
			t.Fatalf("expected COMPLETED status, got %s", compRes.Data.Status)
		}
		if compRes.Data.Result == nil || *compRes.Data.Result != "PASSED" {
			t.Fatalf("expected result PASSED, got %v", compRes.Data.Result)
		}
		if compRes.Data.Feedback == nil || *compRes.Data.Feedback != feedback {
			t.Fatalf("expected feedback saved, got %v", compRes.Data.Feedback)
		}
	})

	// 20. Invalid status transition (cannot complete already cancelled interview)
	t.Run("20_InvalidStatusTransition", func(t *testing.T) {
		body := map[string]interface{}{
			"result": "PASSED",
		}
		raw, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/interviews/"+createdInterviewID+"/complete", bytes.NewReader(raw))
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 for completing cancelled interview, got %d", w.Code)
		}
	})

	// 21. Authorization required
	t.Run("21_AuthorizationRequired", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/interviews", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized without token, got %d", w.Code)
		}
	})

	// 22. Invalid UUID
	t.Run("22_InvalidUUID", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/interviews/not-a-valid-uuid", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request for invalid UUID, got %d", w.Code)
		}
	})

	// 23. Not found
	t.Run("23_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New().String()
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/interviews/"+nonExistentID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404 Not Found, got %d", w.Code)
		}
	})

	// 24. Audit logging
	t.Run("24_AuditLoggingRecorded", func(t *testing.T) {
		if len(interviewRepo.auditLogs) == 0 {
			t.Fatalf("expected audit logs to be recorded for interview actions")
		}
	})

	// 25. No sensitive data in audit payload
	t.Run("25_NoSensitiveDataInAuditPayload", func(t *testing.T) {
		for _, log := range interviewRepo.auditLogs {
			lower := strings.ToLower(log.Metadata)
			if strings.Contains(lower, "password") || strings.Contains(lower, "jwt") || strings.Contains(lower, "bearer") {
				t.Fatalf("audit log contains sensitive tokens: %s", log.Metadata)
			}
		}
	})

	// Candidate Review Integration API
	t.Run("CandidateInterviewsHistory", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/candidates/"+candidate.ID+"/interviews?job_id="+job.ID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}
		var res struct {
			Data []service.InterviewDTO `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		if len(res.Data) == 0 {
			t.Fatalf("expected candidate interviews history to contain entries")
		}
	})

	// Candidate Next Interview API
	t.Run("CandidateNextInterview", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/candidates/"+candidate.ID+"/next-interview?job_id="+job.ID, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}
	})

	// Users List API
	t.Run("UsersListForInterviewers", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/users", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}
		var res struct {
			Data []model.UserResponse `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &res)
		if len(res.Data) < 2 {
			t.Fatalf("expected at least 2 users returned, got %d", len(res.Data))
		}
	})
}

func TestInterviewCalendar_ICSExportAndEscaping(t *testing.T) {
	router, interviewRepo, _, _, token, recruiter, job, candidate := setupInterviewTestRouter()

	candidate.FullName = "Maya Lin, MBA"
	job.Title = "Lead Cloud Architect, Platform"

	startTime := time.Date(2026, 9, 26, 3, 0, 0, 0, time.UTC) // 10:00 Jakarta (UTC+7)
	endTime := startTime.Add(time.Hour)                         // 11:00 Jakarta (UTC+7)

	notes := "First note line;\nSecond note line with, commas\\and semicolons;\r\nThird line."
	meetingURL := "https://meet.google.com/abc-defg-hij"

	ivID := uuid.New().String()
	iv := &model.Interview{
		ID:             ivID,
		JobID:          job.ID,
		CandidateID:    candidate.ID,
		Title:          "Technical Round 1",
		Stage:          model.InterviewStageTechnicalInterview,
		InterviewType:  model.InterviewTypeOnline,
		ScheduledStart: startTime,
		ScheduledEnd:   endTime,
		Timezone:       "Asia/Jakarta",
		MeetingURL:     &meetingURL,
		Notes:          &notes,
		Status:         model.InterviewStatusScheduled,
		CreatedBy:      recruiter.ID,
		Job:            job,
		Candidate:      candidate,
	}
	_ = interviewRepo.Create(context.Background(), iv, []string{recruiter.ID}, nil)

	// 1. Valid ICS Export
	t.Run("Valid ICS Export", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/interviews/"+ivID+"/ics", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if !strings.Contains(contentType, "text/calendar") {
			t.Fatalf("expected Content-Type to contain text/calendar, got %s", contentType)
		}

		contentDisposition := w.Header().Get("Content-Disposition")
		if !strings.Contains(contentDisposition, "attachment") || !strings.Contains(contentDisposition, ".ics") {
			t.Fatalf("expected Content-Disposition attachment with .ics, got %s", contentDisposition)
		}

		body := w.Body.String()

		// Verify RFC 5545 essentials
		if !strings.Contains(body, "BEGIN:VCALENDAR") || !strings.Contains(body, "END:VCALENDAR") {
			t.Fatalf("missing VCALENDAR tags in ICS")
		}
		if !strings.Contains(body, "VERSION:2.0") {
			t.Fatalf("missing VERSION:2.0 in ICS")
		}
		if !strings.Contains(body, "BEGIN:VEVENT") || !strings.Contains(body, "END:VEVENT") {
			t.Fatalf("missing VEVENT tags in ICS")
		}

		// Verify UID
		expectedUID := fmt.Sprintf("UID:%s@hirescope.local", ivID)
		if !strings.Contains(body, expectedUID) {
			t.Fatalf("expected UID %s in body", expectedUID)
		}

		// Verify UTC timestamps
		if !strings.Contains(body, "DTSTART:20260926T030000Z") {
			t.Fatalf("expected UTC DTSTART 20260926T030000Z, got body:\n%s", body)
		}
		if !strings.Contains(body, "DTEND:20260926T040000Z") {
			t.Fatalf("expected UTC DTEND 20260926T040000Z, got body:\n%s", body)
		}

		// Verify escaping of commas and semicolons in candidate name / summary
		if !strings.Contains(body, "Maya Lin\\, MBA") {
			t.Fatalf("expected escaped comma in candidate name Maya Lin\\, MBA, got body:\n%s", body)
		}

		// Verify Meeting URL
		if !strings.Contains(body, "URL:https://meet.google.com/abc-defg-hij") {
			t.Fatalf("expected meeting URL in body")
		}

		// Verify Location Online
		if !strings.Contains(body, "LOCATION:Online") {
			t.Fatalf("expected LOCATION:Online in body")
		}

		// Verify newlines in description escaped to \n (no raw newlines breaking lines)
		if strings.Contains(body, "First note line;\n") {
			t.Fatalf("raw unescaped newline found in ICS body")
		}
		if !strings.Contains(body, `First note line\;`) {
			t.Fatalf("expected escaped semicolon in notes")
		}
	})

	// 2. Nonexistent interview returns 404
	t.Run("Nonexistent interview returns 404", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/interviews/"+uuid.New().String()+"/ics", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("expected 404 Not Found, got %d", w.Code)
		}
	})

	// 3. Unauthorized request returns 401
	t.Run("Unauthorized request returns 401", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/v1/interviews/"+ivID+"/ics", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
		}
	})
}

func TestInterviewCalendar_DateRangeValidation(t *testing.T) {
	router, _, _, _, token, _, _, _ := setupInterviewTestRouter()

	// 1. 'to' before 'from' returns 400 INVALID_DATE_RANGE
	t.Run("To before From returns 400", func(t *testing.T) {
		fromStr := "2026-10-01T00:00:00Z"
		toStr := "2026-09-01T00:00:00Z"
		url := fmt.Sprintf("/api/v1/interviews?from=%s&to=%s", fromStr, toStr)
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "INVALID_DATE_RANGE") {
			t.Fatalf("expected INVALID_DATE_RANGE error code, got %s", w.Body.String())
		}
	})

	// 2. Date range > 366 days returns 400 DATE_RANGE_EXCEEDED
	t.Run("Date range over 366 days returns 400", func(t *testing.T) {
		fromStr := "2026-01-01T00:00:00Z"
		toStr := "2027-02-01T00:00:00Z" // > 366 days
		url := fmt.Sprintf("/api/v1/interviews?from=%s&to=%s", fromStr, toStr)
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("expected 400 Bad Request, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "DATE_RANGE_EXCEEDED") {
			t.Fatalf("expected DATE_RANGE_EXCEEDED error code, got %s", w.Body.String())
		}
	})

	// 3. Valid 30-day month range returns 200 OK
	t.Run("Valid 30-day range returns 200", func(t *testing.T) {
		fromStr := "2026-09-01T00:00:00Z"
		toStr := "2026-09-30T23:59:59Z"
		url := fmt.Sprintf("/api/v1/interviews?from=%s&to=%s", fromStr, toStr)
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("expected 200 OK, got %d", w.Code)
		}
	})
}
