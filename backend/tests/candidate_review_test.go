package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

// memoryJobCandidateRepo implements repository.JobCandidateRepository in-memory for testing.
type memoryJobCandidateRepo struct {
	mu            sync.RWMutex
	jobCandidates map[string]*model.JobCandidate // key: jobID + ":" + candidateID
	candRepo      repository.CandidateRepository
	jobRepo       repository.JobRepository
	users         map[string]*model.User
}

func newMemoryJobCandidateRepo(candRepo repository.CandidateRepository, jobRepo repository.JobRepository) *memoryJobCandidateRepo {
	return &memoryJobCandidateRepo{
		jobCandidates: make(map[string]*model.JobCandidate),
		candRepo:      candRepo,
		jobRepo:       jobRepo,
		users:         make(map[string]*model.User),
	}
}

func (r *memoryJobCandidateRepo) SetUser(u *model.User) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[u.ID] = u
}

func (r *memoryJobCandidateRepo) GetJobCandidate(ctx context.Context, jobID, candidateID string) (*model.JobCandidate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := jobID + ":" + candidateID
	jc, exists := r.jobCandidates[key]
	if !exists {
		return nil, repository.ErrJobCandidateNotFound
	}

	copyJC := *jc
	if r.candRepo != nil {
		cand, _ := r.candRepo.GetByID(ctx, candidateID, true)
		copyJC.Candidate = cand
	}
	if copyJC.ReviewedBy != nil {
		if reviewer, ok := r.users[*copyJC.ReviewedBy]; ok {
			copyJC.Reviewer = reviewer
		}
	}
	return &copyJC, nil
}

func (r *memoryJobCandidateRepo) ListJobCandidates(ctx context.Context, filter repository.JobCandidateFilter) (*repository.JobCandidateListResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matched []model.JobCandidate
	for _, jc := range r.jobCandidates {
		if jc.JobID != filter.JobID {
			continue
		}
		if filter.Status != "" && string(jc.Status) != filter.Status {
			continue
		}

		copyJC := *jc
		if r.candRepo != nil {
			cand, _ := r.candRepo.GetByID(ctx, jc.CandidateID, true)
			copyJC.Candidate = cand
		}
		if copyJC.ReviewedBy != nil {
			if reviewer, ok := r.users[*copyJC.ReviewedBy]; ok {
				copyJC.Reviewer = reviewer
			}
		}

		if filter.Search != "" {
			term := strings.ToLower(filter.Search)
			if copyJC.Candidate != nil {
				nameMatch := strings.Contains(strings.ToLower(copyJC.Candidate.FullName), term)
				emailMatch := strings.Contains(strings.ToLower(copyJC.Candidate.Email), term)
				phoneMatch := strings.Contains(strings.ToLower(copyJC.Candidate.Phone), term)
				if !nameMatch && !emailMatch && !phoneMatch {
					continue
				}
			} else {
				continue
			}
		}

		matched = append(matched, copyJC)
	}

	total := int64(len(matched))
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 {
		limit = 10
	}

	startIndex := (page - 1) * limit
	if startIndex > len(matched) {
		startIndex = len(matched)
	}
	endIndex := startIndex + limit
	if endIndex > len(matched) {
		endIndex = len(matched)
	}

	totalPages := 0
	if limit > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}

	return &repository.JobCandidateListResult{
		Items:      matched[startIndex:endIndex],
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (r *memoryJobCandidateRepo) UpdateStatus(ctx context.Context, jobID, candidateID string, status model.JobCandidateStatus, reviewedBy string, reviewedAt time.Time) (*model.JobCandidate, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := jobID + ":" + candidateID
	jc, exists := r.jobCandidates[key]
	if !exists {
		return nil, repository.ErrJobCandidateNotFound
	}

	jc.Status = status
	jc.ReviewedBy = &reviewedBy
	jc.ReviewedAt = &reviewedAt
	jc.UpdatedAt = time.Now().UTC()

	copyJC := *jc
	if r.candRepo != nil {
		cand, _ := r.candRepo.GetByID(ctx, candidateID, true)
		copyJC.Candidate = cand
	}
	if reviewer, ok := r.users[reviewedBy]; ok {
		copyJC.Reviewer = reviewer
	}
	return &copyJC, nil
}

func (r *memoryJobCandidateRepo) AddJobCandidate(ctx context.Context, jobID, candidateID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := jobID + ":" + candidateID
	if _, exists := r.jobCandidates[key]; exists {
		return nil
	}

	r.jobCandidates[key] = &model.JobCandidate{
		JobID:       jobID,
		CandidateID: candidateID,
		Status:      model.JobCandidateStatusReview,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	return nil
}

func (r *memoryJobCandidateRepo) RemoveJobCandidate(ctx context.Context, jobID, candidateID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	key := jobID + ":" + candidateID
	delete(r.jobCandidates, key)
	return nil
}

func (r *memoryJobCandidateRepo) IsJobCandidateAssociated(ctx context.Context, jobID, candidateID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	key := jobID + ":" + candidateID
	_, exists := r.jobCandidates[key]
	return exists, nil
}

// memoryCandidateNoteRepo implements repository.CandidateNoteRepository in-memory for testing.
type memoryCandidateNoteRepo struct {
	mu    sync.RWMutex
	notes map[string]*model.CandidateNote
	users map[string]*model.User
}

func newMemoryCandidateNoteRepo() *memoryCandidateNoteRepo {
	return &memoryCandidateNoteRepo{
		notes: make(map[string]*model.CandidateNote),
		users: make(map[string]*model.User),
	}
}

func (r *memoryCandidateNoteRepo) SetUser(u *model.User) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[u.ID] = u
}

func (r *memoryCandidateNoteRepo) Create(ctx context.Context, note *model.CandidateNote) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if note.ID == "" {
		note.ID = uuid.New().String()
	}
	now := time.Now().UTC()
	note.CreatedAt = now
	note.UpdatedAt = now

	if author, ok := r.users[note.AuthorID]; ok {
		note.Author = author
	}

	r.notes[note.ID] = note
	return nil
}

func (r *memoryCandidateNoteRepo) GetByID(ctx context.Context, noteID string) (*model.CandidateNote, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	note, exists := r.notes[noteID]
	if !exists {
		return nil, repository.ErrNoteNotFound
	}
	copyNote := *note
	if author, ok := r.users[copyNote.AuthorID]; ok {
		copyNote.Author = author
	}
	return &copyNote, nil
}

func (r *memoryCandidateNoteRepo) ListByJobAndCandidate(ctx context.Context, jobID, candidateID string, page, limit int) (*repository.CandidateNoteListResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var list []model.CandidateNote
	for _, n := range r.notes {
		if n.JobID == jobID && n.CandidateID == candidateID {
			copyN := *n
			if author, ok := r.users[copyN.AuthorID]; ok {
				copyN.Author = author
			}
			list = append(list, copyN)
		}
	}

	// Sort descending by CreatedAt
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].CreatedAt.After(list[i].CreatedAt) {
				list[i], list[j] = list[j], list[i]
			}
		}
	}

	total := int64(len(list))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	start := (page - 1) * limit
	if start > len(list) {
		start = len(list)
	}
	end := start + limit
	if end > len(list) {
		end = len(list)
	}

	totalPages := 0
	if limit > 0 {
		totalPages = int((total + int64(limit) - 1) / int64(limit))
	}

	return &repository.CandidateNoteListResult{
		Items:      list[start:end],
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (r *memoryCandidateNoteRepo) Update(ctx context.Context, note *model.CandidateNote) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.notes[note.ID]
	if !exists {
		return repository.ErrNoteNotFound
	}

	existing.Content = note.Content
	existing.UpdatedAt = time.Now().UTC()
	return nil
}

func (r *memoryCandidateNoteRepo) Delete(ctx context.Context, noteID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.notes[noteID]; !exists {
		return repository.ErrNoteNotFound
	}
	delete(r.notes, noteID)
	return nil
}

// memoryAuditLogRepo implements repository.AuditLogRepository in-memory for testing.
type memoryAuditLogRepo struct {
	mu   sync.RWMutex
	logs []model.AuditLog
}

func newMemoryAuditLogRepo() *memoryAuditLogRepo {
	return &memoryAuditLogRepo{
		logs: make([]model.AuditLog, 0),
	}
}

func (r *memoryAuditLogRepo) Create(ctx context.Context, l *model.AuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if l.ID == "" {
		l.ID = uuid.New().String()
	}
	l.CreatedAt = time.Now().UTC()
	r.logs = append(r.logs, *l)
	return nil
}

func (r *memoryAuditLogRepo) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.logs)
}

func (r *memoryAuditLogRepo) LastLog() *model.AuditLog {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.logs) == 0 {
		return nil
	}
	last := r.logs[len(r.logs)-1]
	return &last
}

func setupCandidateReviewTestRouter() (
	*gin.Engine,
	*memoryJobRepo,
	*memoryCandidateRepo,
	*memoryJobCandidateRepo,
	*memoryCandidateNoteRepo,
	*memoryAuditLogRepo,
	*memoryScreeningRepo,
	service.JWTService,
	*model.User,
	*model.User, // second recruiter
	*model.User, // admin
) {
	gin.SetMode(gin.TestMode)

	jobRepo := newMemoryJobRepo()
	candRepo := newMemoryCandidateRepo(jobRepo)
	screeningRepo := newMemoryScreeningRepo(jobRepo)
	jcRepo := newMemoryJobCandidateRepo(candRepo, jobRepo)
	noteRepo := newMemoryCandidateNoteRepo()
	auditRepo := newMemoryAuditLogRepo()

	jwtSvc, _ := service.NewJWTService("test_secret_for_review_tests_only_12345", 24)
	revokedRepo := newMemoryRevokedRepo()

	recruiter1 := &model.User{
		ID:    uuid.New().String(),
		Name:  "Sarah Jenkins",
		Email: "sarah@hirescope.local",
		Role:  model.RoleRecruiter,
	}
	recruiter2 := &model.User{
		ID:    uuid.New().String(),
		Name:  "Alex Miller",
		Email: "alex@hirescope.local",
		Role:  model.RoleRecruiter,
	}
	admin := &model.User{
		ID:    uuid.New().String(),
		Name:  "Admin Boss",
		Email: "admin@hirescope.local",
		Role:  model.RoleAdmin,
	}

	jcRepo.SetUser(recruiter1)
	jcRepo.SetUser(recruiter2)
	jcRepo.SetUser(admin)

	noteRepo.SetUser(recruiter1)
	noteRepo.SetUser(recruiter2)
	noteRepo.SetUser(admin)

	reviewSvc := service.NewCandidateReviewService(jcRepo, noteRepo, jobRepo, candRepo, screeningRepo, auditRepo)
	reviewHdl := handler.NewCandidateReviewHandler(reviewSvc)

	r := gin.New()
	v1 := r.Group("/api/v1")
	v1.Use(middleware.AuthMiddleware(jwtSvc, revokedRepo))
	{
		jobsGroup := v1.Group("/jobs")
		{
			jobsGroup.GET("/:id/candidates", reviewHdl.ListCandidatesByJob)
			jobsGroup.GET("/:id/candidates/:candidateId/review", reviewHdl.GetReviewDetail)
			jobsGroup.PATCH("/:id/candidates/:candidateId/status", reviewHdl.UpdateStatus)
			jobsGroup.POST("/:id/candidates/:candidateId/notes", reviewHdl.CreateNote)
			jobsGroup.GET("/:id/candidates/:candidateId/notes", reviewHdl.ListNotes)
			jobsGroup.PUT("/:id/candidates/:candidateId/notes/:noteId", reviewHdl.UpdateNote)
			jobsGroup.DELETE("/:id/candidates/:candidateId/notes/:noteId", reviewHdl.DeleteNote)
		}
	}

	return r, jobRepo, candRepo, jcRepo, noteRepo, auditRepo, screeningRepo, jwtSvc, recruiter1, recruiter2, admin
}

// Helper to seed a job and candidate associated together
func seedAssociatedJobAndCandidate(
	t *testing.T,
	jobRepo repository.JobRepository,
	candRepo repository.CandidateRepository,
	jcRepo repository.JobCandidateRepository,
	creatorID string,
) (*model.Job, *model.Candidate) {
	ctx := context.Background()

	job := &model.Job{
		ID:             uuid.New().String(),
		Code:           "JOB-" + uuid.New().String()[:8],
		Title:          "Senior Backend Engineer",
		Description:    "Golang & Microservices",
		Department:     "Engineering",
		Location:       "Remote",
		EmploymentType: model.EmpFullTime,
		Status:         model.JobStatusOpen,
		CreatedBy:      creatorID,
	}
	if err := jobRepo.Create(ctx, job); err != nil {
		t.Fatalf("failed to create job: %v", err)
	}

	cand := &model.Candidate{
		ID:        uuid.New().String(),
		FullName:  "Alex Rivera",
		Email:     "alex.rivera@example.com",
		Phone:     "+1-555-0199",
		Headline:  "Senior Go Architect",
		CreatedBy: creatorID,
	}
	if err := candRepo.Create(ctx, cand); err != nil {
		t.Fatalf("failed to create candidate: %v", err)
	}

	if err := jcRepo.AddJobCandidate(ctx, job.ID, cand.ID); err != nil {
		t.Fatalf("failed to associate candidate: %v", err)
	}

	return job, cand
}

// 1. Test Candidate Default Workflow Status is REVIEW
func TestCandidateReview_DefaultStatusIsReview(t *testing.T) {
	router, jobRepo, candRepo, jcRepo, _, _, _, jwtSvc, recruiter1, _, _ := setupCandidateReviewTestRouter()
	token, _, _ := jwtSvc.GenerateToken(recruiter1)

	job, cand := seedAssociatedJobAndCandidate(t, jobRepo, candRepo, jcRepo, recruiter1.ID)

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/jobs/%s/candidates", job.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data struct {
			Items []struct {
				ID        string `json:"id"`
				FullName  string `json:"full_name"`
				Status    string `json:"status"`
				Candidate struct {
					ID       string `json:"id"`
					FullName string `json:"full_name"`
				} `json:"candidate"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Data.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(resp.Data.Items))
	}
	item := resp.Data.Items[0]
	if item.Status != string(model.JobCandidateStatusReview) {
		t.Errorf("expected default status REVIEW, got %s", item.Status)
	}
	if item.Candidate.FullName != "Alex Rivera" || item.FullName != "Alex Rivera" {
		t.Errorf("expected candidate name Alex Rivera, got %s", item.FullName)
	}
	if item.ID != cand.ID {
		t.Errorf("expected candidate ID %s, got %s", cand.ID, item.ID)
	}
}

// 2. Test Workflow Status Transitions (REVIEW -> SHORTLISTED -> REVIEW -> REJECTED)
func TestCandidateReview_StatusTransitions(t *testing.T) {
	router, jobRepo, candRepo, jcRepo, _, auditRepo, _, jwtSvc, recruiter1, _, _ := setupCandidateReviewTestRouter()
	token, _, _ := jwtSvc.GenerateToken(recruiter1)

	job, cand := seedAssociatedJobAndCandidate(t, jobRepo, candRepo, jcRepo, recruiter1.ID)

	// Step A: Transition to SHORTLISTED
	patchBody := []byte(`{"status":"SHORTLISTED"}`)
	req, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/status", job.ID, cand.ID), bytes.NewBuffer(patchBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on SHORTLISTED, got %d: %s", w.Code, w.Body.String())
	}

	var statusResp struct {
		Data struct {
			Status     string `json:"status"`
			ReviewedBy struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"reviewed_by"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("failed to parse status resp: %v", err)
	}
	if statusResp.Data.Status != "SHORTLISTED" {
		t.Errorf("expected status SHORTLISTED, got %s", statusResp.Data.Status)
	}
	if statusResp.Data.ReviewedBy.ID != recruiter1.ID {
		t.Errorf("expected reviewer ID %s, got %s", recruiter1.ID, statusResp.Data.ReviewedBy.ID)
	}

	// Verify Audit Log
	lastLog := auditRepo.LastLog()
	if lastLog == nil || lastLog.Action != model.AuditActionCandidateStatusChanged {
		t.Errorf("expected CANDIDATE_STATUS_CHANGED audit log, got %+v", lastLog)
	}

	// Step B: Transition to REJECTED
	patchBody = []byte(`{"status":"REJECTED"}`)
	req, _ = http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/status", job.ID, cand.ID), bytes.NewBuffer(patchBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on REJECTED, got %d: %s", w.Code, w.Body.String())
	}
	json.Unmarshal(w.Body.Bytes(), &statusResp)
	if statusResp.Data.Status != "REJECTED" {
		t.Errorf("expected status REJECTED, got %s", statusResp.Data.Status)
	}

	// Step C: Transition back to REVIEW
	patchBody = []byte(`{"status":"REVIEW"}`)
	req, _ = http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/status", job.ID, cand.ID), bytes.NewBuffer(patchBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 on REVIEW, got %d: %s", w.Code, w.Body.String())
	}
	json.Unmarshal(w.Body.Bytes(), &statusResp)
	if statusResp.Data.Status != "REVIEW" {
		t.Errorf("expected status REVIEW, got %s", statusResp.Data.Status)
	}

	// Step D: Invalid status value rejected
	patchBody = []byte(`{"status":"HIRED"}`)
	req, _ = http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/status", job.ID, cand.ID), bytes.NewBuffer(patchBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 on invalid status, got %d", w.Code)
	}
}

// 3. Test Note CRUD and Validation
func TestCandidateReview_NotesCRUD(t *testing.T) {
	router, jobRepo, candRepo, jcRepo, _, auditRepo, _, jwtSvc, recruiter1, _, _ := setupCandidateReviewTestRouter()
	token, _, _ := jwtSvc.GenerateToken(recruiter1)

	job, cand := seedAssociatedJobAndCandidate(t, jobRepo, candRepo, jcRepo, recruiter1.ID)

	// Step A: Create Note with valid content
	noteBody := []byte(`{"content":"Candidate showed strong distributed systems experience in initial screening."}`)
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/notes", job.ID, cand.ID), bytes.NewBuffer(noteBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for note, got %d: %s", w.Code, w.Body.String())
	}

	var createResp struct {
		Data struct {
			ID      string `json:"id"`
			Content string `json:"content"`
			Author  struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"author"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &createResp); err != nil {
		t.Fatalf("failed to decode create note resp: %v", err)
	}
	noteID := createResp.Data.ID
	if noteID == "" {
		t.Fatal("expected non-empty note ID")
	}
	if createResp.Data.Author.ID != recruiter1.ID {
		t.Errorf("expected author ID %s, got %s", recruiter1.ID, createResp.Data.Author.ID)
	}

	// Verify Audit Log
	lastLog := auditRepo.LastLog()
	if lastLog == nil || lastLog.Action != model.AuditActionCandidateNoteCreated {
		t.Errorf("expected CANDIDATE_NOTE_CREATED audit log, got %+v", lastLog)
	}

	// Step B: List Notes
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/notes", job.ID, cand.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK listing notes, got %d: %s", w.Code, w.Body.String())
	}

	var listResp struct {
		Data struct {
			Items []struct {
				ID      string `json:"id"`
				Content string `json:"content"`
			} `json:"items"`
			Total int64 `json:"total"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("failed to parse list notes resp: %v", err)
	}
	if listResp.Data.Total != 1 || len(listResp.Data.Items) != 1 {
		t.Fatalf("expected 1 note, got %d", listResp.Data.Total)
	}
	if listResp.Data.Items[0].ID != noteID {
		t.Errorf("expected note ID %s, got %s", noteID, listResp.Data.Items[0].ID)
	}

	// Step C: Update Note
	updateBody := []byte(`{"content":"Updated: Candidate showed exceptional distributed systems depth."}`)
	req, _ = http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/notes/%s", job.ID, cand.ID, noteID), bytes.NewBuffer(updateBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK updating note, got %d: %s", w.Code, w.Body.String())
	}

	// Step D: Validation - Empty Note rejected
	emptyBody := []byte(`{"content":"   "}`)
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/notes", job.ID, cand.ID), bytes.NewBuffer(emptyBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for whitespace note, got %d", w.Code)
	}

	// Step E: Validation - Oversized Note (>5000 chars) rejected
	longContent := strings.Repeat("A", 5001)
	longBody, _ := json.Marshal(map[string]string{"content": longContent})
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/notes", job.ID, cand.ID), bytes.NewBuffer(longBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for note > 5000 chars, got %d", w.Code)
	}

	// Step F: Delete Note
	req, _ = http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/notes/%s", job.ID, cand.ID, noteID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK deleting note, got %d: %s", w.Code, w.Body.String())
	}

	// Verify Note is gone
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/notes", job.ID, cand.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	json.Unmarshal(w.Body.Bytes(), &listResp)
	if listResp.Data.Total != 0 {
		t.Errorf("expected 0 notes after deletion, got %d", listResp.Data.Total)
	}
}

// 4. Test Note Authorization Guard (Recruiter 1 author vs Recruiter 2 stranger vs Admin)
func TestCandidateReview_NotePermissions(t *testing.T) {
	router, jobRepo, candRepo, jcRepo, _, _, _, jwtSvc, recruiter1, recruiter2, admin := setupCandidateReviewTestRouter()
	token1, _, _ := jwtSvc.GenerateToken(recruiter1)
	token2, _, _ := jwtSvc.GenerateToken(recruiter2)
	tokenAdmin, _, _ := jwtSvc.GenerateToken(admin)

	job, cand := seedAssociatedJobAndCandidate(t, jobRepo, candRepo, jcRepo, recruiter1.ID)

	// Recruiter 1 creates note
	noteBody := []byte(`{"content":"Recruiter 1 initial assessment."}`)
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/notes", job.ID, cand.ID), bytes.NewBuffer(noteBody))
	req.Header.Set("Authorization", "Bearer "+token1)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	noteID := createResp.Data.ID

	// Recruiter 2 tries to update Recruiter 1's note -> 403 Forbidden
	req, _ = http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/notes/%s", job.ID, cand.ID, noteID), bytes.NewBuffer([]byte(`{"content":"Tampered note"}`)))
	req.Header.Set("Authorization", "Bearer "+token2)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden when recruiter modifies another recruiter note, got %d", w.Code)
	}

	// Recruiter 2 tries to delete Recruiter 1's note -> 403 Forbidden
	req, _ = http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/notes/%s", job.ID, cand.ID, noteID), nil)
	req.Header.Set("Authorization", "Bearer "+token2)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden when recruiter deletes another recruiter note, got %d", w.Code)
	}

	// Admin CAN update Recruiter 1's note -> 200 OK
	req, _ = http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/notes/%s", job.ID, cand.ID, noteID), bytes.NewBuffer([]byte(`{"content":"Admin approved edit."}`)))
	req.Header.Set("Authorization", "Bearer "+tokenAdmin)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK when admin updates note, got %d", w.Code)
	}

	// Admin CAN delete Recruiter 1's note -> 200 OK
	req, _ = http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/notes/%s", job.ID, cand.ID, noteID), nil)
	req.Header.Set("Authorization", "Bearer "+tokenAdmin)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 OK when admin deletes note, got %d", w.Code)
	}
}

// 5. Test Cross-Job Note Isolation (prevent accessing/modifying note under another job)
func TestCandidateReview_CrossJobNoteIsolation(t *testing.T) {
	router, jobRepo, candRepo, jcRepo, _, _, _, jwtSvc, recruiter1, _, _ := setupCandidateReviewTestRouter()
	token, _, _ := jwtSvc.GenerateToken(recruiter1)

	job1, cand1 := seedAssociatedJobAndCandidate(t, jobRepo, candRepo, jcRepo, recruiter1.ID)
	job2, _ := seedAssociatedJobAndCandidate(t, jobRepo, candRepo, jcRepo, recruiter1.ID)

	// Note on Job 1
	noteBody := []byte(`{"content":"Job 1 specific feedback"}`)
	req, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/notes", job1.ID, cand1.ID), bytes.NewBuffer(noteBody))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createResp struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &createResp)
	noteID := createResp.Data.ID

	// Attempt to update note via Job 2 URL -> 404 NOTE_NOT_FOUND
	req, _ = http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/notes/%s", job2.ID, cand1.ID, noteID), bytes.NewBuffer([]byte(`{"content":"Cross-job edit"}`)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 NOTE_NOT_FOUND on cross-job note update, got %d", w.Code)
	}

	// Attempt to delete note via Job 2 URL -> 404 NOTE_NOT_FOUND
	req, _ = http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/notes/%s", job2.ID, cand1.ID, noteID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 NOTE_NOT_FOUND on cross-job note deletion, got %d", w.Code)
	}
}

// 6. Test Review Detail Context Snapshot (Job, Candidate, Workflow, Safe Docs, Education, Experience, Skills, Screening, Notes)
func TestCandidateReview_ReviewDetailContext(t *testing.T) {
	router, jobRepo, candRepo, jcRepo, _, _, screeningRepo, jwtSvc, recruiter1, _, _ := setupCandidateReviewTestRouter()
	token, _, _ := jwtSvc.GenerateToken(recruiter1)
	ctx := context.Background()

	job, cand := seedAssociatedJobAndCandidate(t, jobRepo, candRepo, jcRepo, recruiter1.ID)

	// Add profile sections to candidate
	candRepo.CreateSkill(ctx, &model.CandidateSkill{CandidateID: cand.ID, Skill: "Go"})
	candRepo.CreateSkill(ctx, &model.CandidateSkill{CandidateID: cand.ID, Skill: "Docker"})
	candRepo.CreateEducation(ctx, &model.CandidateEducation{CandidateID: cand.ID, Institution: "MIT", Degree: "Bachelor of Science", FieldOfStudy: "Computer Science"})
	candRepo.CreateExperience(ctx, &model.CandidateExperience{CandidateID: cand.ID, Company: "Stripe", Position: "Senior Software Engineer"})
	fn := "resume.pdf"
	mt := "application/pdf"
	sz := int64(102400)
	sp := "/private/storage/cvs/resume_123.pdf"
	candRepo.CreateDocument(ctx, &model.CandidateDocument{
		CandidateID:      cand.ID,
		SourceType:       model.SourceTypePDF,
		OriginalFilename: &fn,
		MimeType:         &mt,
		FileSize:         &sz,
		StoragePath:      &sp, // Should NEVER be exposed in review response
		RawText:          "Full resume text",
	})

	// Save Step 7 Screening Result
	screeningResult := &model.ScreeningResult{
		ID:                     uuid.New().String(),
		JobID:                  job.ID,
		CandidateID:            cand.ID,
		Status:                 model.StatusQualified,
		RequiredMatchCount:     2,
		RequiredPartialCount:   0,
		RequiredMismatchCount:  0,
		RequiredUnknownCount:   0,
		PreferredMatchCount:    0,
		PreferredPartialCount:  0,
		PreferredMismatchCount: 0,
		PreferredUnknownCount:  0,
		EvaluatedAt:            time.Now().UTC(),
	}
	screeningRepo.SaveScreeningResult(ctx, screeningResult, []model.ScreeningMatch{
		{
			ID:                uuid.New().String(),
			ScreeningResultID: screeningResult.ID,
			RequirementID:     uuid.New().String(),
			Status:            model.MatchStatusMatch,
			Evidence:          "Go",
			Reason:            "Exact match found",
			Confidence:        model.ConfidenceHigh,
			Source:            model.SourceSkill,
			MatchedValue:      "Go",
		},
	})

	// Add a note
	reqNote, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/notes", job.ID, cand.ID), bytes.NewBuffer([]byte(`{"content":"Interview scheduled for next Tuesday."}`)))
	reqNote.Header.Set("Authorization", "Bearer "+token)
	reqNote.Header.Set("Content-Type", "application/json")
	wNote := httptest.NewRecorder()
	router.ServeHTTP(wNote, reqNote)

	// Now Fetch Review Detail Endpoint
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/review", job.ID, cand.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for review detail, got %d: %s", w.Code, w.Body.String())
	}

	bodyStr := w.Body.String()
	// Security check: Verify internal storage path is NOT leaked
	if strings.Contains(bodyStr, "/private/storage/cvs/resume_123.pdf") {
		t.Errorf("SECURITY LEAK: review endpoint exposed raw storage_path!")
	}

	var reviewResp struct {
		Data struct {
			Job struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"job"`
			Candidate struct {
				ID       string `json:"id"`
				FullName string `json:"full_name"`
			} `json:"candidate"`
			Workflow struct {
				Status string `json:"status"`
			} `json:"workflow"`
			Documents []struct {
				ID         string  `json:"id"`
				SourceType string  `json:"source_type"`
				FileName   *string `json:"file_name"`
			} `json:"documents"`
			Education []struct {
				Institution string `json:"institution"`
			} `json:"education"`
			Experience []struct {
				Company string `json:"company"`
			} `json:"experience"`
			Skills []struct {
				Skill string `json:"skill"`
			} `json:"skills"`
			Screening *struct {
				ScreeningResult struct {
					Status             string `json:"status"`
					RequiredMatchCount int    `json:"required_match_count"`
				} `json:"screening_result"`
				Matches []any `json:"matches"`
			} `json:"screening"`
			Notes []struct {
				Content string `json:"content"`
			} `json:"notes"`
		} `json:"data"`
	}

	if err := json.Unmarshal(w.Body.Bytes(), &reviewResp); err != nil {
		t.Fatalf("failed to decode review detail response: %v", err)
	}

	d := reviewResp.Data
	if d.Job.Title != "Senior Backend Engineer" {
		t.Errorf("expected job title 'Senior Backend Engineer', got %s", d.Job.Title)
	}
	if d.Candidate.FullName != "Alex Rivera" {
		t.Errorf("expected candidate name 'Alex Rivera', got %s", d.Candidate.FullName)
	}
	if d.Workflow.Status != "REVIEW" {
		t.Errorf("expected workflow status REVIEW, got %s", d.Workflow.Status)
	}
	if len(d.Documents) != 1 || d.Documents[0].FileName == nil || *d.Documents[0].FileName != "resume.pdf" {
		t.Errorf("expected document resume.pdf, got %+v", d.Documents)
	}
	if len(d.Education) != 1 || d.Education[0].Institution != "MIT" {
		t.Errorf("expected education MIT, got %+v", d.Education)
	}
	if len(d.Experience) != 1 || d.Experience[0].Company != "Stripe" {
		t.Errorf("expected experience Stripe, got %+v", d.Experience)
	}
	if len(d.Skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(d.Skills))
	}
	if d.Screening == nil || d.Screening.ScreeningResult.Status != "QUALIFIED" || d.Screening.ScreeningResult.RequiredMatchCount != 2 {
		t.Errorf("expected qualified screening with 2 matches, got %+v", d.Screening)
	}
	if len(d.Notes) != 1 || d.Notes[0].Content != "Interview scheduled for next Tuesday." {
		t.Errorf("expected 1 note, got %+v", d.Notes)
	}
}

// 7. Test Candidate Not Associated with Job returns 404
func TestCandidateReview_UnassociatedCandidate(t *testing.T) {
	router, jobRepo, candRepo, _, _, _, _, jwtSvc, recruiter1, _, _ := setupCandidateReviewTestRouter()
	token, _, _ := jwtSvc.GenerateToken(recruiter1)
	ctx := context.Background()

	job := &model.Job{
		ID:             uuid.New().String(),
		Code:           "JOB-" + uuid.New().String()[:8],
		Title:          "Frontend Developer",
		CreatedBy:      recruiter1.ID,
		Status:         model.JobStatusOpen,
		EmploymentType: model.EmpFullTime,
	}
	jobRepo.Create(ctx, job)

	cand := &model.Candidate{
		ID:        uuid.New().String(),
		FullName:  "Unassociated Candidate",
		CreatedBy: recruiter1.ID,
	}
	candRepo.Create(ctx, cand)

	// Attempt GET review -> 404
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/review", job.ID, cand.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 on unassociated candidate review, got %d", w.Code)
	}

	// Attempt PATCH status -> 404
	req, _ = http.NewRequest(http.MethodPatch, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/status", job.ID, cand.ID), bytes.NewBuffer([]byte(`{"status":"SHORTLISTED"}`)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 on unassociated candidate status update, got %d", w.Code)
	}

	// Attempt POST note -> 404
	req, _ = http.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/notes", job.ID, cand.ID), bytes.NewBuffer([]byte(`{"content":"hello"}`)))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 on unassociated candidate note creation, got %d", w.Code)
	}
}

// 8. Test Candidate List Filtering by Status and Search
func TestCandidateReview_ListFilteringAndSearch(t *testing.T) {
	router, jobRepo, candRepo, jcRepo, _, _, _, jwtSvc, recruiter1, _, _ := setupCandidateReviewTestRouter()
	token, _, _ := jwtSvc.GenerateToken(recruiter1)
	ctx := context.Background()

	job := &model.Job{
		ID:             uuid.New().String(),
		Code:           "JOB-" + uuid.New().String()[:8],
		Title:          "DevOps Engineer",
		CreatedBy:      recruiter1.ID,
		Status:         model.JobStatusOpen,
		EmploymentType: model.EmpFullTime,
	}
	jobRepo.Create(ctx, job)

	// Candidate 1: Shortlisted
	cand1 := &model.Candidate{ID: uuid.New().String(), FullName: "Alice Cloud", Email: "alice@cloud.com", CreatedBy: recruiter1.ID}
	candRepo.Create(ctx, cand1)
	jcRepo.AddJobCandidate(ctx, job.ID, cand1.ID)
	jcRepo.UpdateStatus(ctx, job.ID, cand1.ID, model.JobCandidateStatusShortlisted, recruiter1.ID, time.Now().UTC())

	// Candidate 2: In Review
	cand2 := &model.Candidate{ID: uuid.New().String(), FullName: "Bob Infra", Email: "bob@infra.com", CreatedBy: recruiter1.ID}
	candRepo.Create(ctx, cand2)
	jcRepo.AddJobCandidate(ctx, job.ID, cand2.ID)

	// Candidate 3: Rejected
	cand3 := &model.Candidate{ID: uuid.New().String(), FullName: "Charlie Legacy", Email: "charlie@legacy.com", CreatedBy: recruiter1.ID}
	candRepo.Create(ctx, cand3)
	jcRepo.AddJobCandidate(ctx, job.ID, cand3.ID)
	jcRepo.UpdateStatus(ctx, job.ID, cand3.ID, model.JobCandidateStatusRejected, recruiter1.ID, time.Now().UTC())

	// Filter by status=SHORTLISTED
	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/jobs/%s/candidates?status=SHORTLISTED", job.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp struct {
		Data struct {
			Items []struct {
				FullName string `json:"full_name"`
				Status   string `json:"status"`
			} `json:"items"`
			Pagination struct {
				Total int64 `json:"total"`
			} `json:"pagination"`
		} `json:"data"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.Pagination.Total != 1 || resp.Data.Items[0].FullName != "Alice Cloud" {
		t.Errorf("expected 1 item with Alice Cloud for status=SHORTLISTED, got %+v", resp.Data)
	}

	// Filter by search=Infra
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/jobs/%s/candidates?search=Infra", job.ID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data.Pagination.Total != 1 || resp.Data.Items[0].FullName != "Bob Infra" {
		t.Errorf("expected 1 item with Bob Infra for search=Infra, got %+v", resp.Data)
	}
}
