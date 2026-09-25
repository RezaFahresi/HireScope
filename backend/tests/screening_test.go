package tests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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

// memoryScreeningRepo implements repository.ScreeningRepository in-memory for database-safe tests.
type memoryScreeningRepo struct {
	mu      sync.RWMutex
	results map[string]*model.ScreeningResult // key: candidateID + ":" + jobID
	byID    map[string]*model.ScreeningResult // key: id
	matches map[string][]model.ScreeningMatch // key: screeningResultID
	jobRepo repository.JobRepository
}

func newMemoryScreeningRepo(jobRepo repository.JobRepository) *memoryScreeningRepo {
	return &memoryScreeningRepo{
		results: make(map[string]*model.ScreeningResult),
		byID:    make(map[string]*model.ScreeningResult),
		matches: make(map[string][]model.ScreeningMatch),
		jobRepo: jobRepo,
	}
}

func (r *memoryScreeningRepo) GetByCandidateAndJob(ctx context.Context, candidateID, jobID string) (*model.ScreeningResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := candidateID + ":" + jobID
	res, found := r.results[key]
	if !found {
		return nil, repository.ErrScreeningResultNotFound
	}
	copyRes := *res
	matches := r.matches[res.ID]
	copyMatches := make([]model.ScreeningMatch, len(matches))
	copy(copyMatches, matches)
	copyRes.Matches = copyMatches
	return &copyRes, nil
}

func (r *memoryScreeningRepo) GetByID(ctx context.Context, id string) (*model.ScreeningResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res, found := r.byID[id]
	if !found {
		return nil, repository.ErrScreeningResultNotFound
	}
	copyRes := *res
	matches := r.matches[res.ID]
	copyMatches := make([]model.ScreeningMatch, len(matches))
	copy(copyMatches, matches)
	copyRes.Matches = copyMatches
	return &copyRes, nil
}

func (r *memoryScreeningRepo) SaveScreeningResult(ctx context.Context, result *model.ScreeningResult, matches []model.ScreeningMatch) (*model.ScreeningResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := result.CandidateID + ":" + result.JobID
	existing, found := r.results[key]
	now := time.Now().UTC()

	if found {
		existing.Status = result.Status
		existing.RequiredMatchCount = result.RequiredMatchCount
		existing.RequiredPartialCount = result.RequiredPartialCount
		existing.RequiredMismatchCount = result.RequiredMismatchCount
		existing.RequiredUnknownCount = result.RequiredUnknownCount
		existing.PreferredMatchCount = result.PreferredMatchCount
		existing.PreferredPartialCount = result.PreferredPartialCount
		existing.PreferredMismatchCount = result.PreferredMismatchCount
		existing.PreferredUnknownCount = result.PreferredUnknownCount
		existing.EvaluatedAt = now
		existing.UpdatedAt = now
		result = existing
	} else {
		if result.ID == "" {
			result.ID = uuid.New().String()
		}
		result.EvaluatedAt = now
		result.CreatedAt = now
		result.UpdatedAt = now
		r.results[key] = result
		r.byID[result.ID] = result
	}

	savedMatches := make([]model.ScreeningMatch, len(matches))
	for i, m := range matches {
		if m.ID == "" {
			m.ID = uuid.New().String()
		}
		m.ScreeningResultID = result.ID
		m.CreatedAt = now
		m.UpdatedAt = now
		if m.Requirement == nil && r.jobRepo != nil {
			req, _ := r.jobRepo.GetRequirementByID(ctx, m.RequirementID)
			m.Requirement = req
		}
		savedMatches[i] = m
	}
	r.matches[result.ID] = savedMatches
	result.Matches = savedMatches

	copyRes := *result
	copyMatches := make([]model.ScreeningMatch, len(savedMatches))
	copy(copyMatches, savedMatches)
	copyRes.Matches = copyMatches
	return &copyRes, nil
}

func setupScreeningTestRouter() (*gin.Engine, *memoryJobRepo, *memoryCandidateRepo, *memoryScreeningRepo, service.JWTService, *model.User) {
	gin.SetMode(gin.TestMode)

	jobRepo := newMemoryJobRepo()
	candRepo := newMemoryCandidateRepo(jobRepo)
	screeningRepo := newMemoryScreeningRepo(jobRepo)

	jwtSvc, _ := service.NewJWTService("test_secret_for_screening_tests_only_12345", 24)
	revokedRepo := newMemoryRevokedRepo()

	screeningEngine := service.NewScreeningEngine()
	screeningSvc := service.NewScreeningService(screeningRepo, jobRepo, candRepo, screeningEngine)
	screeningHdl := handler.NewScreeningHandler(screeningSvc)

	user := &model.User{
		ID:    uuid.New().String(),
		Email: "recruiter@hirescope.local",
		Role:  model.RoleRecruiter,
	}

	r := gin.New()
	v1 := r.Group("/api/v1")
	v1.Use(middleware.AuthMiddleware(jwtSvc, revokedRepo))
	{
		v1.POST("/jobs/:id/candidates/:candidateId/screen", screeningHdl.ScreenCandidate)
		v1.GET("/jobs/:id/candidates/:candidateId/screening", screeningHdl.GetLatestScreening)
	}

	return r, jobRepo, candRepo, screeningRepo, jwtSvc, user
}

func TestScreening_FullQualifiedCandidate(t *testing.T) {
	router, jobRepo, candRepo, _, jwtSvc, user := setupScreeningTestRouter()
	token, _, _ := jwtSvc.GenerateToken(user)
	ctx := context.Background()

	// 1. Create Job with Requirements (4 REQUIRED, 2 PREFERRED)
	job := &model.Job{
		ID:        uuid.New().String(),
		Title:     "Senior Backend Engineer",
		CreatedBy: user.ID,
	}
	_ = jobRepo.Create(ctx, job)

	reqs := []model.JobRequirement{
		{ID: uuid.New().String(), JobID: job.ID, Category: model.CategorySkill, Requirement: "Go", Importance: model.ImportanceRequired},
		{ID: uuid.New().String(), JobID: job.ID, Category: model.CategorySkill, Requirement: "PostgreSQL", Importance: model.ImportanceRequired},
		{ID: uuid.New().String(), JobID: job.ID, Category: model.CategoryExperience, Requirement: "3 years experience", Importance: model.ImportanceRequired},
		{ID: uuid.New().String(), JobID: job.ID, Category: model.CategoryEducation, Requirement: "Bachelor's degree in Computer Science", Importance: model.ImportanceRequired},
		{ID: uuid.New().String(), JobID: job.ID, Category: model.CategorySkill, Requirement: "Docker", Importance: model.ImportancePreferred},
		{ID: uuid.New().String(), JobID: job.ID, Category: model.CategoryLanguage, Requirement: "English", Importance: model.ImportancePreferred},
	}
	for _, req := range reqs {
		rCopy := req
		_ = jobRepo.CreateRequirement(ctx, &rCopy)
	}

	// 2. Create Candidate
	d3YearsAgo := time.Now().UTC().AddDate(-3, -6, 0)
	cand := &model.Candidate{
		ID:        uuid.New().String(),
		FullName:  "Budi Pratama",
		Summary:   "Professional Backend Developer with English working proficiency.",
		CreatedBy: user.ID,
		Skills: []model.CandidateSkill{
			{Skill: "Go", NormalizedSkill: "go"},
			{Skill: "PostgreSQL", NormalizedSkill: "postgresql"},
			{Skill: "Git", NormalizedSkill: "git"},
		},
		Educations: []model.CandidateEducation{
			{
				Institution:  "Institut Teknologi Bandung",
				Degree:       "Bachelor of Computer Science",
				FieldOfStudy: "Computer Science",
			},
		},
		Experiences: []model.CandidateExperience{
			{
				Position:  "Backend Engineer",
				Company:   "Tech Corp",
				StartDate: &d3YearsAgo,
				IsCurrent: true,
			},
		},
	}
	_ = candRepo.Create(ctx, cand)
	// Link candidate to job
	_ = candRepo.AddJobCandidate(ctx, job.ID, cand.ID)

	// 3. Screen Candidate
	w := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/screen", job.ID, cand.ID), token, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK screening, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data service.ScreeningResponse `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	// All 4 REQUIRED match -> QUALIFIED
	if resp.Data.ScreeningResult.Status != model.StatusQualified {
		t.Errorf("expected QUALIFIED status, got %v", resp.Data.ScreeningResult.Status)
	}
	if resp.Data.ScreeningResult.RequiredMatchCount != 4 {
		t.Errorf("expected 4 required matches, got %d", resp.Data.ScreeningResult.RequiredMatchCount)
	}
	if resp.Data.ScreeningResult.RequiredMismatchCount != 0 {
		t.Errorf("expected 0 required mismatch, got %d", resp.Data.ScreeningResult.RequiredMismatchCount)
	}

	// Preferred: English is in summary -> MATCH; Docker is missing -> UNKNOWN
	if resp.Data.ScreeningResult.PreferredMatchCount != 1 {
		t.Errorf("expected 1 preferred match, got %d", resp.Data.ScreeningResult.PreferredMatchCount)
	}
	if resp.Data.ScreeningResult.PreferredUnknownCount != 1 {
		t.Errorf("expected 1 preferred unknown (Docker), got %d", resp.Data.ScreeningResult.PreferredUnknownCount)
	}

	// 4. Retrieve screening via GET endpoint
	wGet := execAuthReq(router, http.MethodGet, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/screening", job.ID, cand.ID), token, nil)
	if wGet.Code != http.StatusOK {
		t.Fatalf("expected 200 OK retrieving screening, got %d: %s", wGet.Code, wGet.Body.String())
	}
}

func TestScreening_OverallStatusTransitions(t *testing.T) {
	router, jobRepo, candRepo, _, jwtSvc, user := setupScreeningTestRouter()
	token, _, _ := jwtSvc.GenerateToken(user)
	ctx := context.Background()

	// Scenario 1: One REQUIRED is PARTIAL (e.g. 5 years required, candidate has 2 years) -> REVIEW
	job1 := &model.Job{ID: uuid.New().String(), Title: "Job 1", CreatedBy: user.ID}
	_ = jobRepo.Create(ctx, job1)
	_ = jobRepo.CreateRequirement(ctx, &model.JobRequirement{ID: uuid.New().String(), JobID: job1.ID, Category: model.CategoryExperience, Requirement: "5 years experience", Importance: model.ImportanceRequired})

	d2YearsAgo := time.Now().UTC().AddDate(-2, 0, 0)
	cand1 := &model.Candidate{
		ID:        uuid.New().String(),
		FullName:  "Cand 1",
		CreatedBy: user.ID,
		Experiences: []model.CandidateExperience{
			{Position: "Dev", Company: "C", StartDate: &d2YearsAgo, IsCurrent: true},
		},
	}
	_ = candRepo.Create(ctx, cand1)
	_ = candRepo.AddJobCandidate(ctx, job1.ID, cand1.ID)

	w1 := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/screen", job1.ID, cand1.ID), token, nil)
	var resp1 struct {
		Data service.ScreeningResponse `json:"data"`
	}
	_ = json.Unmarshal(w1.Body.Bytes(), &resp1)
	if resp1.Data.ScreeningResult.Status != model.StatusReview {
		t.Errorf("expected REVIEW for required partial, got %v", resp1.Data.ScreeningResult.Status)
	}

	// Scenario 2: One REQUIRED is UNKNOWN (e.g. missing skill) -> REVIEW
	job2 := &model.Job{ID: uuid.New().String(), Title: "Job 2", CreatedBy: user.ID}
	_ = jobRepo.Create(ctx, job2)
	_ = jobRepo.CreateRequirement(ctx, &model.JobRequirement{ID: uuid.New().String(), JobID: job2.ID, Category: model.CategorySkill, Requirement: "Elixir", Importance: model.ImportanceRequired})

	cand2 := &model.Candidate{ID: uuid.New().String(), FullName: "Cand 2", CreatedBy: user.ID}
	_ = candRepo.Create(ctx, cand2)
	_ = candRepo.AddJobCandidate(ctx, job2.ID, cand2.ID)

	w2 := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/screen", job2.ID, cand2.ID), token, nil)
	var resp2 struct {
		Data service.ScreeningResponse `json:"data"`
	}
	_ = json.Unmarshal(w2.Body.Bytes(), &resp2)
	if resp2.Data.ScreeningResult.Status != model.StatusReview {
		t.Errorf("expected REVIEW for required unknown, got %v", resp2.Data.ScreeningResult.Status)
	}

	// Scenario 3: One REQUIRED is MISMATCH (e.g. Required Master, Candidate only has Bachelor) -> NOT_QUALIFIED
	job3 := &model.Job{ID: uuid.New().String(), Title: "Job 3", CreatedBy: user.ID}
	_ = jobRepo.Create(ctx, job3)
	_ = jobRepo.CreateRequirement(ctx, &model.JobRequirement{ID: uuid.New().String(), JobID: job3.ID, Category: model.CategoryEducation, Requirement: "Master's degree", Importance: model.ImportanceRequired})

	cand3 := &model.Candidate{
		ID:        uuid.New().String(),
		FullName:  "Cand 3",
		CreatedBy: user.ID,
		Educations: []model.CandidateEducation{
			{Institution: "UI", Degree: "Bachelor of Science", FieldOfStudy: "Physics"},
		},
	}
	_ = candRepo.Create(ctx, cand3)
	_ = candRepo.AddJobCandidate(ctx, job3.ID, cand3.ID)

	w3 := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/screen", job3.ID, cand3.ID), token, nil)
	var resp3 struct {
		Data service.ScreeningResponse `json:"data"`
	}
	_ = json.Unmarshal(w3.Body.Bytes(), &resp3)
	if resp3.Data.ScreeningResult.Status != model.StatusNotQualified {
		t.Errorf("expected NOT_QUALIFIED for required mismatch, got %v", resp3.Data.ScreeningResult.Status)
	}

	// Scenario 4: PREFERRED mismatch must NOT automatically cause NOT_QUALIFIED!
	job4 := &model.Job{ID: uuid.New().String(), Title: "Job 4", CreatedBy: user.ID}
	_ = jobRepo.Create(ctx, job4)
	_ = jobRepo.CreateRequirement(ctx, &model.JobRequirement{ID: uuid.New().String(), JobID: job4.ID, Category: model.CategorySkill, Requirement: "Go", Importance: model.ImportanceRequired})
	// PREFERRED Master's degree
	_ = jobRepo.CreateRequirement(ctx, &model.JobRequirement{ID: uuid.New().String(), JobID: job4.ID, Category: model.CategoryEducation, Requirement: "Master's degree", Importance: model.ImportancePreferred})

	cand4 := &model.Candidate{
		ID:        uuid.New().String(),
		FullName:  "Cand 4",
		CreatedBy: user.ID,
		Skills:    []model.CandidateSkill{{Skill: "Go", NormalizedSkill: "go"}},
		Educations: []model.CandidateEducation{
			{Institution: "ITB", Degree: "Bachelor of Engineering"},
		},
	}
	_ = candRepo.Create(ctx, cand4)
	_ = candRepo.AddJobCandidate(ctx, job4.ID, cand4.ID)

	w4 := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/screen", job4.ID, cand4.ID), token, nil)
	var resp4 struct {
		Data service.ScreeningResponse `json:"data"`
	}
	_ = json.Unmarshal(w4.Body.Bytes(), &resp4)
	if resp4.Data.ScreeningResult.Status != model.StatusQualified {
		t.Errorf("expected QUALIFIED despite PREFERRED education mismatch, got %v", resp4.Data.ScreeningResult.Status)
	}
	if resp4.Data.ScreeningResult.PreferredMismatchCount != 1 {
		t.Errorf("expected 1 preferred mismatch count, got %d", resp4.Data.ScreeningResult.PreferredMismatchCount)
	}
}

func TestScreening_IdempotencyAndNoDuplicates(t *testing.T) {
	router, jobRepo, candRepo, _, jwtSvc, user := setupScreeningTestRouter()
	token, _, _ := jwtSvc.GenerateToken(user)
	ctx := context.Background()

	job := &model.Job{ID: uuid.New().String(), Title: "Idempotency Job", CreatedBy: user.ID}
	_ = jobRepo.Create(ctx, job)
	_ = jobRepo.CreateRequirement(ctx, &model.JobRequirement{ID: uuid.New().String(), JobID: job.ID, Category: model.CategorySkill, Requirement: "Go", Importance: model.ImportanceRequired})
	_ = jobRepo.CreateRequirement(ctx, &model.JobRequirement{ID: uuid.New().String(), JobID: job.ID, Category: model.CategorySkill, Requirement: "Docker", Importance: model.ImportanceRequired})

	cand := &model.Candidate{
		ID:        uuid.New().String(),
		FullName:  "Idempotent Candidate",
		CreatedBy: user.ID,
		Skills:    []model.CandidateSkill{{Skill: "Go", NormalizedSkill: "go"}},
	}
	_ = candRepo.Create(ctx, cand)
	_ = candRepo.AddJobCandidate(ctx, job.ID, cand.ID)

	// First screen
	w1 := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/screen", job.ID, cand.ID), token, nil)
	var resp1 struct {
		Data service.ScreeningResponse `json:"data"`
	}
	_ = json.Unmarshal(w1.Body.Bytes(), &resp1)
	firstID := resp1.Data.ScreeningResult.ID
	if len(resp1.Data.Matches) != 2 {
		t.Fatalf("expected 2 matches on first run, got %d", len(resp1.Data.Matches))
	}

	// Second screen on same candidate and job
	w2 := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/screen", job.ID, cand.ID), token, nil)
	var resp2 struct {
		Data service.ScreeningResponse `json:"data"`
	}
	_ = json.Unmarshal(w2.Body.Bytes(), &resp2)
	secondID := resp2.Data.ScreeningResult.ID

	if firstID != secondID {
		t.Errorf("expected same screening result ID on re-screening, got %s vs %s", firstID, secondID)
	}
	if len(resp2.Data.Matches) != 2 {
		t.Errorf("idempotency violation: expected 2 matches on second run, got %d", len(resp2.Data.Matches))
	}
}

func TestScreening_CandidateNotAssociatedRejected(t *testing.T) {
	router, jobRepo, candRepo, _, jwtSvc, user := setupScreeningTestRouter()
	token, _, _ := jwtSvc.GenerateToken(user)
	ctx := context.Background()

	job := &model.Job{ID: uuid.New().String(), Title: "Unlinked Job", CreatedBy: user.ID}
	_ = jobRepo.Create(ctx, job)

	cand := &model.Candidate{ID: uuid.New().String(), FullName: "Unlinked Cand", CreatedBy: user.ID}
	_ = candRepo.Create(ctx, cand)
	// Do NOT link candidate to job!

	w := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/screen", job.ID, cand.ID), token, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for unassociated candidate, got %d: %s", w.Code, w.Body.String())
	}
}

func TestScreening_NotFoundAndInvalidUUID(t *testing.T) {
	router, _, _, _, jwtSvc, user := setupScreeningTestRouter()
	token, _, _ := jwtSvc.GenerateToken(user)

	validUUID1 := uuid.New().String()
	validUUID2 := uuid.New().String()

	// 1. Invalid UUID
	w1 := execAuthReq(router, http.MethodPost, "/api/v1/jobs/bad-uuid/candidates/invalid-uuid/screen", token, nil)
	if w1.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad UUID, got %d", w1.Code)
	}

	// 2. Non-existent Job
	w2 := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/screen", validUUID1, validUUID2), token, nil)
	if w2.Code != http.StatusNotFound {
		t.Errorf("expected 404 for non-existent job, got %d", w2.Code)
	}

	// 3. GET screening when none exists
	w3 := execAuthReq(router, http.MethodGet, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/screening", validUUID1, validUUID2), token, nil)
	if w3.Code != http.StatusNotFound {
		t.Errorf("expected 404 for missing screening result, got %d", w3.Code)
	}
}

func TestScreening_AuthenticationRequired(t *testing.T) {
	router, _, _, _, _, _ := setupScreeningTestRouter()

	jobID := uuid.New().String()
	candID := uuid.New().String()

	// POST without token
	wPost := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/screen", jobID, candID), "", nil)
	if wPost.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated screen, got %d", wPost.Code)
	}

	// GET without token
	wGet := execAuthReq(router, http.MethodGet, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s/screening", jobID, candID), "", nil)
	if wGet.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated GET screening, got %d", wGet.Code)
	}
}
