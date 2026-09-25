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

// memoryJobRepo implements repository.JobRepository for database-safe isolated testing.
type memoryJobRepo struct {
	mu           sync.RWMutex
	jobs         map[string]*model.Job
	requirements map[string]*model.JobRequirement
	codeSeq      int
}

func newMemoryJobRepo() *memoryJobRepo {
	return &memoryJobRepo{
		jobs:         make(map[string]*model.Job),
		requirements: make(map[string]*model.JobRequirement),
		codeSeq:      0,
	}
}

func (r *memoryJobRepo) Create(ctx context.Context, job *model.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if job.ID == "" {
		job.ID = uuid.New().String()
	}
	job.CreatedAt = time.Now().UTC()
	job.UpdatedAt = time.Now().UTC()
	r.jobs[job.ID] = job
	return nil
}

func (r *memoryJobRepo) GetByID(ctx context.Context, id string, includeRelations bool) (*model.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	job, exists := r.jobs[id]
	if !exists {
		return nil, repository.ErrJobNotFound
	}
	copyJob := *job
	if includeRelations {
		var reqs []model.JobRequirement
		for _, req := range r.requirements {
			if req.JobID == id {
				reqs = append(reqs, *req)
			}
		}
		copyJob.Requirements = reqs
	}
	return &copyJob, nil
}

func (r *memoryJobRepo) List(ctx context.Context, filter repository.JobFilter) (*repository.JobListResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matched []model.Job
	for _, j := range r.jobs {
		// Search
		if filter.Search != "" {
			s := strings.ToLower(filter.Search)
			if !strings.Contains(strings.ToLower(j.Title), s) &&
				!strings.Contains(strings.ToLower(j.Description), s) &&
				!strings.Contains(strings.ToLower(j.Department), s) {
				continue
			}
		}
		// Status
		if filter.Status != "" && string(j.Status) != strings.ToUpper(filter.Status) {
			continue
		}
		// Department
		if filter.Department != "" && !strings.EqualFold(j.Department, filter.Department) {
			continue
		}
		// EmploymentType
		if filter.EmploymentType != "" && string(j.EmploymentType) != strings.ToUpper(filter.EmploymentType) {
			continue
		}
		matched = append(matched, *j)
	}

	total := int64(len(matched))
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	}

	start := (filter.Page - 1) * filter.Limit
	var items []model.Job
	if start < len(matched) {
		end := start + filter.Limit
		if end > len(matched) {
			end = len(matched)
		}
		items = matched[start:end]
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))
	if totalPages == 0 {
		totalPages = 1
	}

	return &repository.JobListResult{
		Items:      items,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *memoryJobRepo) Update(ctx context.Context, job *model.Job) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	job.UpdatedAt = time.Now().UTC()
	r.jobs[job.ID] = job
	return nil
}

func (r *memoryJobRepo) NextJobCode(ctx context.Context) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.codeSeq++
	return fmt.Sprintf("JOB-2026-%04d", r.codeSeq), nil
}

func (r *memoryJobRepo) CreateRequirement(ctx context.Context, req *model.JobRequirement) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if req.ID == "" {
		req.ID = uuid.New().String()
	}
	req.CreatedAt = time.Now().UTC()
	req.UpdatedAt = time.Now().UTC()
	r.requirements[req.ID] = req
	return nil
}

func (r *memoryJobRepo) GetRequirementByID(ctx context.Context, reqID string) (*model.JobRequirement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	req, exists := r.requirements[reqID]
	if !exists {
		return nil, repository.ErrRequirementNotFound
	}
	return req, nil
}

func (r *memoryJobRepo) ListRequirements(ctx context.Context, jobID string) ([]model.JobRequirement, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []model.JobRequirement
	for _, req := range r.requirements {
		if req.JobID == jobID {
			list = append(list, *req)
		}
	}
	return list, nil
}

func (r *memoryJobRepo) UpdateRequirement(ctx context.Context, req *model.JobRequirement) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	req.UpdatedAt = time.Now().UTC()
	r.requirements[req.ID] = req
	return nil
}

func (r *memoryJobRepo) DeleteRequirement(ctx context.Context, jobID, reqID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	req, exists := r.requirements[reqID]
	if !exists || req.JobID != jobID {
		return repository.ErrRequirementNotFound
	}
	delete(r.requirements, reqID)
	return nil
}

func setupJobTestRouter() (*gin.Engine, *memoryJobRepo, service.JWTService, *model.User) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	jwtSvc, _ := service.NewJWTService("secret-for-job-management-testing!!", 1)
	userRepo := newMemoryUserRepo()
	revokedRepo := newMemoryRevokedRepo()
	jobRepo := newMemoryJobRepo()

	testUser := &model.User{
		ID:    "recruiter-uuid-test-1234",
		Name:  "Test Recruiter",
		Email: "recruiter@hirescope.local",
		Role:  model.RoleRecruiter,
	}
	_ = userRepo.Create(context.Background(), testUser)

	jobService := service.NewJobService(jobRepo)
	jobHandler := handler.NewJobHandler(jobService)

	v1 := router.Group("/api/v1")
	jobsGroup := v1.Group("/jobs")
	jobsGroup.Use(middleware.AuthMiddleware(jwtSvc, revokedRepo))
	{
		jobsGroup.POST("", jobHandler.Create)
		jobsGroup.GET("", jobHandler.List)
		jobsGroup.GET("/:id", jobHandler.GetByID)
		jobsGroup.PUT("/:id", jobHandler.Update)
		jobsGroup.PATCH("/:id/status", jobHandler.UpdateStatus)
		jobsGroup.POST("/:id/archive", jobHandler.Archive)

		jobsGroup.GET("/:id/requirements", jobHandler.ListRequirements)
		jobsGroup.POST("/:id/requirements", jobHandler.CreateRequirement)
		jobsGroup.PUT("/:id/requirements/:requirementId", jobHandler.UpdateRequirement)
		jobsGroup.DELETE("/:id/requirements/:requirementId", jobHandler.DeleteRequirement)
	}

	return router, jobRepo, jwtSvc, testUser
}

func TestJobHandler_CreateAndOwnership(t *testing.T) {
	router, _, jwtSvc, testUser := setupJobTestRouter()
	token, _, _ := jwtSvc.GenerateToken(testUser)

	// Attempt to create job with an explicit spoofed created_by
	bodyJSON := `{
		"title": "Senior Go Engineer",
		"description": "Building high-performance recruitment services",
		"department": "Engineering",
		"location": "Jakarta",
		"employment_type": "FULL_TIME",
		"status": "DRAFT",
		"created_by": "spoofed-user-id-attempt"
	}`

	req, _ := http.NewRequest(http.MethodPost, "/api/v1/jobs", bytes.NewBufferString(bodyJSON))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected status 201 Created, got %d. Body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data model.Job `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)

	// Verify ownership: created_by MUST match authenticated JWT user, NOT spoofed value
	if resp.Data.CreatedBy != testUser.ID {
		t.Fatalf("Expected created_by to be %s (authenticated user), got %s", testUser.ID, resp.Data.CreatedBy)
	}
	if resp.Data.Code == "" || !strings.HasPrefix(resp.Data.Code, "JOB-") {
		t.Fatalf("Expected valid job code, got %s", resp.Data.Code)
	}
	if resp.Data.Status != model.JobStatusDraft {
		t.Fatalf("Expected status DRAFT, got %s", resp.Data.Status)
	}
}

func TestJobHandler_ListWithFiltersAndPagination(t *testing.T) {
	router, jobRepo, jwtSvc, testUser := setupJobTestRouter()
	token, _, _ := jwtSvc.GenerateToken(testUser)

	// Seed 3 jobs
	_ = jobRepo.Create(context.Background(), &model.Job{
		ID:             uuid.New().String(),
		Code:           "JOB-2026-0001",
		Title:          "Frontend Developer",
		Description:    "React and TypeScript",
		Department:     "Engineering",
		EmploymentType: model.EmpFullTime,
		Status:         model.JobStatusOpen,
		CreatedBy:      testUser.ID,
	})
	_ = jobRepo.Create(context.Background(), &model.Job{
		ID:             uuid.New().String(),
		Code:           "JOB-2026-0002",
		Title:          "Product Designer",
		Description:    "UI/UX and design systems",
		Department:     "Design",
		EmploymentType: model.EmpContract,
		Status:         model.JobStatusDraft,
		CreatedBy:      testUser.ID,
	})
	_ = jobRepo.Create(context.Background(), &model.Job{
		ID:             uuid.New().String(),
		Code:           "JOB-2026-0003",
		Title:          "Backend Go Developer",
		Description:    "Go, Gin, and PostgreSQL",
		Department:     "Engineering",
		EmploymentType: model.EmpFullTime,
		Status:         model.JobStatusOpen,
		CreatedBy:      testUser.ID,
	})

	// 1. List with search "Go"
	reqSearch, _ := http.NewRequest(http.MethodGet, "/api/v1/jobs?search=Go", nil)
	reqSearch.Header.Set("Authorization", "Bearer "+token)
	wSearch := httptest.NewRecorder()
	router.ServeHTTP(wSearch, reqSearch)
	if wSearch.Code != http.StatusOK {
		t.Fatalf("Expected 200, got %d", wSearch.Code)
	}

	var respSearch struct {
		Data struct {
			Items      []model.Job `json:"items"`
			Pagination struct {
				Total int `json:"total"`
			} `json:"pagination"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wSearch.Body.Bytes(), &respSearch)
	if respSearch.Data.Pagination.Total != 1 || len(respSearch.Data.Items) != 1 {
		t.Fatalf("Expected 1 match for 'Go', got %d", respSearch.Data.Pagination.Total)
	}

	// 2. Filter by status "OPEN"
	reqStatus, _ := http.NewRequest(http.MethodGet, "/api/v1/jobs?status=OPEN", nil)
	reqStatus.Header.Set("Authorization", "Bearer "+token)
	wStatus := httptest.NewRecorder()
	router.ServeHTTP(wStatus, reqStatus)
	var respStatus struct {
		Data struct {
			Pagination struct {
				Total int `json:"total"`
			} `json:"pagination"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wStatus.Body.Bytes(), &respStatus)
	if respStatus.Data.Pagination.Total != 2 {
		t.Fatalf("Expected 2 OPEN jobs, got %d", respStatus.Data.Pagination.Total)
	}
}

func TestJobHandler_StatusTransitions(t *testing.T) {
	router, jobRepo, jwtSvc, testUser := setupJobTestRouter()
	token, _, _ := jwtSvc.GenerateToken(testUser)

	jobID := uuid.New().String()
	_ = jobRepo.Create(context.Background(), &model.Job{
		ID:             jobID,
		Code:           "JOB-2026-0010",
		Title:          "QA Engineer",
		Description:    "Automated testing",
		EmploymentType: model.EmpFullTime,
		Status:         model.JobStatusDraft,
		CreatedBy:      testUser.ID,
	})

	// 1. DRAFT -> CLOSED is invalid (must go through OPEN or ARCHIVED)
	invalidBody := `{"status": "CLOSED"}`
	req1, _ := http.NewRequest(http.MethodPut, "/api/v1/jobs/"+jobID, bytes.NewBufferString(invalidBody))
	req1.Header.Set("Authorization", "Bearer "+token)
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	if w1.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400 for invalid status transition DRAFT -> CLOSED, got %d", w1.Code)
	}

	// 2. DRAFT -> OPEN is valid
	validBody := `{"status": "OPEN"}`
	req2, _ := http.NewRequest(http.MethodPut, "/api/v1/jobs/"+jobID, bytes.NewBufferString(validBody))
	req2.Header.Set("Authorization", "Bearer "+token)
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("Expected status 200 for valid transition DRAFT -> OPEN, got %d. Body: %s", w2.Code, w2.Body.String())
	}

	// 3. Archive endpoint -> status becomes ARCHIVED
	reqArch, _ := http.NewRequest(http.MethodPost, "/api/v1/jobs/"+jobID+"/archive", nil)
	reqArch.Header.Set("Authorization", "Bearer "+token)
	wArch := httptest.NewRecorder()
	router.ServeHTTP(wArch, reqArch)
	if wArch.Code != http.StatusOK {
		t.Fatalf("Expected status 200 for archive, got %d", wArch.Code)
	}

	var respArch struct {
		Data model.Job `json:"data"`
	}
	_ = json.Unmarshal(wArch.Body.Bytes(), &respArch)
	if respArch.Data.Status != model.JobStatusArchived {
		t.Fatalf("Expected status ARCHIVED, got %s", respArch.Data.Status)
	}

	// 4. ARCHIVED -> OPEN is invalid (ARCHIVED is terminal)
	reqReopen, _ := http.NewRequest(http.MethodPut, "/api/v1/jobs/"+jobID, bytes.NewBufferString(validBody))
	reqReopen.Header.Set("Authorization", "Bearer "+token)
	reqReopen.Header.Set("Content-Type", "application/json")
	wReopen := httptest.NewRecorder()
	router.ServeHTTP(wReopen, reqReopen)
	if wReopen.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 for transition from ARCHIVED, got %d", wReopen.Code)
	}
}

func TestJobRequirements_CRUDAndCrossJobProtection(t *testing.T) {
	router, jobRepo, jwtSvc, testUser := setupJobTestRouter()
	token, _, _ := jwtSvc.GenerateToken(testUser)

	jobA_ID := uuid.New().String()
	_ = jobRepo.Create(context.Background(), &model.Job{
		ID:             jobA_ID,
		Code:           "JOB-2026-0020",
		Title:          "Job A",
		Description:    "Description A",
		EmploymentType: model.EmpFullTime,
		Status:         model.JobStatusOpen,
		CreatedBy:      testUser.ID,
	})

	jobB_ID := uuid.New().String()
	_ = jobRepo.Create(context.Background(), &model.Job{
		ID:             jobB_ID,
		Code:           "JOB-2026-0021",
		Title:          "Job B",
		Description:    "Description B",
		EmploymentType: model.EmpFullTime,
		Status:         model.JobStatusOpen,
		CreatedBy:      testUser.ID,
	})

	// 1. Create requirement for Job A
	createReqBody := `{
		"category": "SKILL",
		"requirement": "Go concurrency patterns",
		"importance": "REQUIRED"
	}`
	reqCreate, _ := http.NewRequest(http.MethodPost, "/api/v1/jobs/"+jobA_ID+"/requirements", bytes.NewBufferString(createReqBody))
	reqCreate.Header.Set("Authorization", "Bearer "+token)
	reqCreate.Header.Set("Content-Type", "application/json")
	wCreate := httptest.NewRecorder()
	router.ServeHTTP(wCreate, reqCreate)
	if wCreate.Code != http.StatusCreated {
		t.Fatalf("Expected status 201 Created, got %d. Body: %s", wCreate.Code, wCreate.Body.String())
	}

	var respCreate struct {
		Data model.JobRequirement `json:"data"`
	}
	_ = json.Unmarshal(wCreate.Body.Bytes(), &respCreate)
	reqID := respCreate.Data.ID

	// 2. List requirements for Job A
	reqList, _ := http.NewRequest(http.MethodGet, "/api/v1/jobs/"+jobA_ID+"/requirements", nil)
	reqList.Header.Set("Authorization", "Bearer "+token)
	wList := httptest.NewRecorder()
	router.ServeHTTP(wList, reqList)
	if wList.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", wList.Code)
	}

	// 3. Attempt to update Job A's requirement via Job B's endpoint -> MUST BE REJECTED
	updateReqBody := `{"requirement": "Tampered requirement"}`
	reqCrossUpdate, _ := http.NewRequest(http.MethodPut, "/api/v1/jobs/"+jobB_ID+"/requirements/"+reqID, bytes.NewBufferString(updateReqBody))
	reqCrossUpdate.Header.Set("Authorization", "Bearer "+token)
	reqCrossUpdate.Header.Set("Content-Type", "application/json")
	wCrossUpdate := httptest.NewRecorder()
	router.ServeHTTP(wCrossUpdate, reqCrossUpdate)
	if wCrossUpdate.Code != http.StatusBadRequest {
		t.Fatalf("Expected 400 Bad Request for cross-job requirement tampering, got %d", wCrossUpdate.Code)
	}

	// 4. Attempt to delete Job A's requirement via Job B's endpoint -> MUST BE REJECTED
	reqCrossDelete, _ := http.NewRequest(http.MethodDelete, "/api/v1/jobs/"+jobB_ID+"/requirements/"+reqID, nil)
	reqCrossDelete.Header.Set("Authorization", "Bearer "+token)
	wCrossDelete := httptest.NewRecorder()
	router.ServeHTTP(wCrossDelete, reqCrossDelete)
	if wCrossDelete.Code != http.StatusNotFound && wCrossDelete.Code != http.StatusBadRequest {
		t.Fatalf("Expected 404 or 400 for cross-job deletion, got %d", wCrossDelete.Code)
	}

	// 5. Delete via correct Job A endpoint -> 200 OK
	reqDelete, _ := http.NewRequest(http.MethodDelete, "/api/v1/jobs/"+jobA_ID+"/requirements/"+reqID, nil)
	reqDelete.Header.Set("Authorization", "Bearer "+token)
	wDelete := httptest.NewRecorder()
	router.ServeHTTP(wDelete, reqDelete)
	if wDelete.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for requirement deletion, got %d", wDelete.Code)
	}
}

func TestJobHandler_UnauthenticatedRequestRejected(t *testing.T) {
	router, _, _, _ := setupJobTestRouter()

	req, _ := http.NewRequest(http.MethodGet, "/api/v1/jobs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 for unauthenticated request to /jobs, got %d", w.Code)
	}
}

func TestJobHandler_PublishAndStatusEndpoint(t *testing.T) {
	router, jobRepo, jwtSvc, recruiterUser := setupJobTestRouter()
	recruiterToken, _, _ := jwtSvc.GenerateToken(recruiterUser)

	adminUser := &model.User{
		ID:    "admin-uuid-test-5678",
		Name:  "Test Admin",
		Email: "admin@hirescope.local",
		Role:  model.RoleAdmin,
	}
	adminToken, _, _ := jwtSvc.GenerateToken(adminUser)

	// Helper to create a job in DRAFT status
	createDraftJob := func() string {
		id := uuid.New().String()
		_ = jobRepo.Create(context.Background(), &model.Job{
			ID:             id,
			Code:           "JOB-TEST-" + id[:8],
			Title:          "Principal Systems Engineer",
			Description:    "Systems architecture & cloud infrastructure",
			Department:     "Engineering",
			EmploymentType: model.EmpFullTime,
			Status:         model.JobStatusDraft,
			CreatedBy:      recruiterUser.ID,
		})
		return id
	}

	// 1. DRAFT -> OPEN (Publish) via PATCH /api/v1/jobs/:id/status by Recruiter
	t.Run("Valid DRAFT to OPEN transition by Recruiter", func(t *testing.T) {
		jobID := createDraftJob()
		body := `{"status": "OPEN"}`
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/jobs/"+jobID+"/status", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+recruiterToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d. Body: %s", w.Code, w.Body.String())
		}

		var resp struct {
			Data model.Job `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Data.Status != model.JobStatusOpen {
			t.Fatalf("Expected status OPEN, got %s", resp.Data.Status)
		}
	})

	// 2. Valid DRAFT -> OPEN (Publish) by Admin
	t.Run("Valid DRAFT to OPEN transition by Admin", func(t *testing.T) {
		jobID := createDraftJob()
		body := `{"status": "OPEN"}`
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/jobs/"+jobID+"/status", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+adminToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected 200 OK, got %d", w.Code)
		}

		var resp struct {
			Data model.Job `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		if resp.Data.Status != model.JobStatusOpen {
			t.Fatalf("Expected status OPEN, got %s", resp.Data.Status)
		}
	})

	// 3. Invalid UUID format
	t.Run("Invalid UUID format returns 400", func(t *testing.T) {
		body := `{"status": "OPEN"}`
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/jobs/not-a-uuid/status", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+recruiterToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400 Bad Request, got %d", w.Code)
		}
	})

	// 4. Nonexistent job
	t.Run("Nonexistent job returns 404", func(t *testing.T) {
		randomUUID := uuid.New().String()
		body := `{"status": "OPEN"}`
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/jobs/"+randomUUID+"/status", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+recruiterToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNotFound {
			t.Fatalf("Expected 404 Not Found, got %d", w.Code)
		}
	})

	// 5. Unauthorized request
	t.Run("Unauthorized request without token returns 401", func(t *testing.T) {
		jobID := createDraftJob()
		body := `{"status": "OPEN"}`
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/jobs/"+jobID+"/status", bytes.NewBufferString(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Fatalf("Expected 401 Unauthorized, got %d", w.Code)
		}
	})

	// 6. Invalid status transition (DRAFT -> CLOSED)
	t.Run("Invalid status transition returns 400 INVALID_STATUS_TRANSITION", func(t *testing.T) {
		jobID := createDraftJob()
		body := `{"status": "CLOSED"}`
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/jobs/"+jobID+"/status", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+recruiterToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400 Bad Request, got %d", w.Code)
		}
		if !strings.Contains(w.Body.String(), "INVALID_STATUS_TRANSITION") {
			t.Fatalf("Expected INVALID_STATUS_TRANSITION error code, got %s", w.Body.String())
		}
	})

	// 7. Invalid status value
	t.Run("Invalid status value returns 400", func(t *testing.T) {
		jobID := createDraftJob()
		body := `{"status": "SUPER_STATUS"}`
		req, _ := http.NewRequest(http.MethodPatch, "/api/v1/jobs/"+jobID+"/status", bytes.NewBufferString(body))
		req.Header.Set("Authorization", "Bearer "+recruiterToken)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("Expected 400 Bad Request, got %d", w.Code)
		}
	})
}
