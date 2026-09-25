package tests

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"hirescope/backend/internal/extractor"
	"hirescope/backend/internal/handler"
	"hirescope/backend/internal/middleware"
	"hirescope/backend/internal/model"
	"hirescope/backend/internal/repository"
	"hirescope/backend/internal/service"
	"hirescope/backend/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// memoryCandidateRepo implements repository.CandidateRepository in-memory for testing.
type memoryCandidateRepo struct {
	mu            sync.RWMutex
	candidates    map[string]*model.Candidate
	jobCandidates map[string]*model.JobCandidate
	educations    map[string]*model.CandidateEducation
	experiences   map[string]*model.CandidateExperience
	skills        map[string]*model.CandidateSkill
	documents     map[string]*model.CandidateDocument
	jobRepo       repository.JobRepository
}

func newMemoryCandidateRepo(jobRepo repository.JobRepository) *memoryCandidateRepo {
	return &memoryCandidateRepo{
		candidates:    make(map[string]*model.Candidate),
		jobCandidates: make(map[string]*model.JobCandidate),
		educations:    make(map[string]*model.CandidateEducation),
		experiences:   make(map[string]*model.CandidateExperience),
		skills:        make(map[string]*model.CandidateSkill),
		documents:     make(map[string]*model.CandidateDocument),
		jobRepo:       jobRepo,
	}
}

func (r *memoryCandidateRepo) Create(ctx context.Context, c *model.Candidate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	c.CreatedAt = time.Now().UTC()
	c.UpdatedAt = time.Now().UTC()
	r.candidates[c.ID] = c

	for i := range c.Skills {
		s := c.Skills[i]
		if s.ID == "" {
			s.ID = uuid.New().String()
		}
		s.CandidateID = c.ID
		r.skills[s.ID] = &s
	}
	for i := range c.Educations {
		e := c.Educations[i]
		if e.ID == "" {
			e.ID = uuid.New().String()
		}
		e.CandidateID = c.ID
		r.educations[e.ID] = &e
	}
	for i := range c.Experiences {
		exp := c.Experiences[i]
		if exp.ID == "" {
			exp.ID = uuid.New().String()
		}
		exp.CandidateID = c.ID
		r.experiences[exp.ID] = &exp
	}
	return nil
}

func (r *memoryCandidateRepo) GetByID(ctx context.Context, id string, includeRelations bool) (*model.Candidate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, exists := r.candidates[id]
	if !exists {
		return nil, repository.ErrCandidateNotFound
	}
	copyCand := *c
	if includeRelations {
		var edus []model.CandidateEducation
		for _, e := range r.educations {
			if e.CandidateID == id {
				edus = append(edus, *e)
			}
		}
		copyCand.Educations = edus

		var exps []model.CandidateExperience
		for _, exp := range r.experiences {
			if exp.CandidateID == id {
				exps = append(exps, *exp)
			}
		}
		copyCand.Experiences = exps

		var sks []model.CandidateSkill
		for _, s := range r.skills {
			if s.CandidateID == id {
				sks = append(sks, *s)
			}
		}
		copyCand.Skills = sks

		var docs []model.CandidateDocument
		for _, d := range r.documents {
			if d.CandidateID == id {
				docs = append(docs, *d)
			}
		}
		copyCand.Documents = docs
	}
	return &copyCand, nil
}

func (r *memoryCandidateRepo) List(ctx context.Context, filter repository.CandidateFilter) (*repository.CandidateListResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matched []model.Candidate
	for _, c := range r.candidates {
		if filter.Search != "" {
			s := strings.ToLower(filter.Search)
			if !strings.Contains(strings.ToLower(c.FullName), s) &&
				!strings.Contains(strings.ToLower(c.Email), s) &&
				!strings.Contains(strings.ToLower(c.Headline), s) &&
				!strings.Contains(strings.ToLower(c.Summary), s) {
				continue
			}
		}
		if filter.Location != "" && !strings.Contains(strings.ToLower(c.Location), strings.ToLower(filter.Location)) {
			continue
		}
		matched = append(matched, *c)
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

	start := (page - 1) * limit
	end := start + limit
	if start > len(matched) {
		start = len(matched)
	}
	if end > len(matched) {
		end = len(matched)
	}

	paged := matched[start:end]
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	if totalPages == 0 {
		totalPages = 1
	}

	return &repository.CandidateListResult{
		Items:      paged,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (r *memoryCandidateRepo) Update(ctx context.Context, c *model.Candidate) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.candidates[c.ID]; !exists {
		return repository.ErrCandidateNotFound
	}
	c.UpdatedAt = time.Now().UTC()
	r.candidates[c.ID] = c
	return nil
}

func (r *memoryCandidateRepo) AddJobCandidate(ctx context.Context, jobID, candidateID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := jobID + ":" + candidateID
	if _, exists := r.jobCandidates[key]; exists {
		return repository.ErrJobCandidateAlreadyExists
	}
	jc := &model.JobCandidate{
		ID:          uuid.New().String(),
		JobID:       jobID,
		CandidateID: candidateID,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	r.jobCandidates[key] = jc
	return nil
}

func (r *memoryCandidateRepo) RemoveJobCandidate(ctx context.Context, jobID, candidateID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := jobID + ":" + candidateID
	if _, exists := r.jobCandidates[key]; !exists {
		return repository.ErrJobCandidateNotFound
	}
	delete(r.jobCandidates, key)
	return nil
}

func (r *memoryCandidateRepo) IsJobCandidateAssociated(ctx context.Context, jobID, candidateID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := jobID + ":" + candidateID
	_, exists := r.jobCandidates[key]
	return exists, nil
}

func (r *memoryCandidateRepo) ListCandidatesByJob(ctx context.Context, jobID string, page, limit int) (*repository.CandidateListResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matched []model.Candidate
	for _, jc := range r.jobCandidates {
		if jc.JobID == jobID {
			if c, exists := r.candidates[jc.CandidateID]; exists {
				matched = append(matched, *c)
			}
		}
	}

	total := int64(len(matched))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	start := (page - 1) * limit
	end := start + limit
	if start > len(matched) {
		start = len(matched)
	}
	if end > len(matched) {
		end = len(matched)
	}
	paged := matched[start:end]
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	if totalPages == 0 {
		totalPages = 1
	}

	return &repository.CandidateListResult{
		Items:      paged,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (r *memoryCandidateRepo) ListJobsByCandidate(ctx context.Context, candidateID string) ([]model.Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var jobs []model.Job
	for _, jc := range r.jobCandidates {
		if jc.CandidateID == candidateID {
			if j, err := r.jobRepo.GetByID(ctx, jc.JobID, false); err == nil && j != nil {
				jobs = append(jobs, *j)
			}
		}
	}
	return jobs, nil
}

func (r *memoryCandidateRepo) CreateEducation(ctx context.Context, edu *model.CandidateEducation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if edu.ID == "" {
		edu.ID = uuid.New().String()
	}
	edu.CreatedAt = time.Now().UTC()
	edu.UpdatedAt = time.Now().UTC()
	r.educations[edu.ID] = edu
	return nil
}

func (r *memoryCandidateRepo) GetEducationByID(ctx context.Context, id string) (*model.CandidateEducation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	edu, exists := r.educations[id]
	if !exists {
		return nil, repository.ErrEducationNotFound
	}
	return edu, nil
}

func (r *memoryCandidateRepo) ListEducations(ctx context.Context, candidateID string) ([]model.CandidateEducation, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []model.CandidateEducation
	for _, e := range r.educations {
		if e.CandidateID == candidateID {
			list = append(list, *e)
		}
	}
	return list, nil
}

func (r *memoryCandidateRepo) UpdateEducation(ctx context.Context, edu *model.CandidateEducation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.educations[edu.ID]; !exists {
		return repository.ErrEducationNotFound
	}
	edu.UpdatedAt = time.Now().UTC()
	r.educations[edu.ID] = edu
	return nil
}

func (r *memoryCandidateRepo) DeleteEducation(ctx context.Context, candidateID, eduID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	edu, exists := r.educations[eduID]
	if !exists || edu.CandidateID != candidateID {
		return repository.ErrEducationNotFound
	}
	delete(r.educations, eduID)
	return nil
}

func (r *memoryCandidateRepo) CreateExperience(ctx context.Context, exp *model.CandidateExperience) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if exp.ID == "" {
		exp.ID = uuid.New().String()
	}
	exp.CreatedAt = time.Now().UTC()
	exp.UpdatedAt = time.Now().UTC()
	r.experiences[exp.ID] = exp
	return nil
}

func (r *memoryCandidateRepo) GetExperienceByID(ctx context.Context, id string) (*model.CandidateExperience, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	exp, exists := r.experiences[id]
	if !exists {
		return nil, repository.ErrExperienceNotFound
	}
	return exp, nil
}

func (r *memoryCandidateRepo) ListExperiences(ctx context.Context, candidateID string) ([]model.CandidateExperience, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []model.CandidateExperience
	for _, e := range r.experiences {
		if e.CandidateID == candidateID {
			list = append(list, *e)
		}
	}
	return list, nil
}

func (r *memoryCandidateRepo) UpdateExperience(ctx context.Context, exp *model.CandidateExperience) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.experiences[exp.ID]; !exists {
		return repository.ErrExperienceNotFound
	}
	exp.UpdatedAt = time.Now().UTC()
	r.experiences[exp.ID] = exp
	return nil
}

func (r *memoryCandidateRepo) DeleteExperience(ctx context.Context, candidateID, expID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	exp, exists := r.experiences[expID]
	if !exists || exp.CandidateID != candidateID {
		return repository.ErrExperienceNotFound
	}
	delete(r.experiences, expID)
	return nil
}

func (r *memoryCandidateRepo) CreateSkill(ctx context.Context, skill *model.CandidateSkill) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if skill.ID == "" {
		skill.ID = uuid.New().String()
	}
	skill.CreatedAt = time.Now().UTC()
	skill.UpdatedAt = time.Now().UTC()
	r.skills[skill.ID] = skill
	return nil
}

func (r *memoryCandidateRepo) GetSkillByID(ctx context.Context, id string) (*model.CandidateSkill, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, exists := r.skills[id]
	if !exists {
		return nil, repository.ErrSkillNotFound
	}
	return s, nil
}

func (r *memoryCandidateRepo) ListSkills(ctx context.Context, candidateID string) ([]model.CandidateSkill, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []model.CandidateSkill
	for _, s := range r.skills {
		if s.CandidateID == candidateID {
			list = append(list, *s)
		}
	}
	return list, nil
}

func (r *memoryCandidateRepo) UpdateSkill(ctx context.Context, skill *model.CandidateSkill) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.skills[skill.ID]; !exists {
		return repository.ErrSkillNotFound
	}
	skill.UpdatedAt = time.Now().UTC()
	r.skills[skill.ID] = skill
	return nil
}

func (r *memoryCandidateRepo) DeleteSkill(ctx context.Context, candidateID, skillID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, exists := r.skills[skillID]
	if !exists || s.CandidateID != candidateID {
		return repository.ErrSkillNotFound
	}
	delete(r.skills, skillID)
	return nil
}

func (r *memoryCandidateRepo) HasSkill(ctx context.Context, candidateID, normalizedSkill, excludeSkillID string) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, s := range r.skills {
		if s.CandidateID == candidateID && s.NormalizedSkill == normalizedSkill {
			if excludeSkillID == "" || s.ID != excludeSkillID {
				return true, nil
			}
		}
	}
	return false, nil
}

func (r *memoryCandidateRepo) CreateDocument(ctx context.Context, doc *model.CandidateDocument) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if doc.ID == "" {
		doc.ID = uuid.New().String()
	}
	doc.CreatedAt = time.Now().UTC()
	doc.UpdatedAt = time.Now().UTC()
	r.documents[doc.ID] = doc
	return nil
}

func (r *memoryCandidateRepo) GetDocumentByID(ctx context.Context, id string) (*model.CandidateDocument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	doc, exists := r.documents[id]
	if !exists {
		return nil, repository.ErrDocumentNotFound
	}
	return doc, nil
}

func (r *memoryCandidateRepo) ListDocuments(ctx context.Context, candidateID string) ([]model.CandidateDocument, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var list []model.CandidateDocument
	for _, d := range r.documents {
		if d.CandidateID == candidateID {
			list = append(list, *d)
		}
	}
	return list, nil
}

func (r *memoryCandidateRepo) UpdateDocument(ctx context.Context, doc *model.CandidateDocument) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.documents[doc.ID]; !exists {
		return repository.ErrDocumentNotFound
	}
	doc.UpdatedAt = time.Now().UTC()
	r.documents[doc.ID] = doc
	return nil
}

func (r *memoryCandidateRepo) DeleteDocument(ctx context.Context, candidateID, docID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	doc, exists := r.documents[docID]
	if !exists || doc.CandidateID != candidateID {
		return repository.ErrDocumentNotFound
	}
	delete(r.documents, docID)
	return nil
}

// setupCandidateTestRouter builds a full test Gin engine with in-memory candidate and job repositories.
func setupCandidateTestRouter() (*gin.Engine, *memoryCandidateRepo, *memoryJobRepo, service.JWTService, *model.User) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	jwtSvc, _ := service.NewJWTService("secret-for-candidate-testing-12345!", 1)
	userRepo := newMemoryUserRepo()
	revokedRepo := newMemoryRevokedRepo()
	jobRepo := newMemoryJobRepo()
	candRepo := newMemoryCandidateRepo(jobRepo)

	tempDir, _ := os.MkdirTemp("", "hirescope-cand-test-*")
	storageSvc, _ := storage.NewLocalStorageService(tempDir)
	pdfExtractor := extractor.NewPDFTextExtractor()
	docxExtractor := extractor.NewDOCXTextExtractor()

	testRecruiter := &model.User{
		ID:    "recruiter-cand-uuid-1",
		Name:  "Lead Recruiter",
		Email: "recruiter@hirescope.local",
		Role:  model.RoleRecruiter,
	}
	_ = userRepo.Create(context.Background(), testRecruiter)

	candService := service.NewCandidateService(candRepo, jobRepo, storageSvc, pdfExtractor, docxExtractor)
	candHandler := handler.NewCandidateHandler(candService)

	jobService := service.NewJobService(jobRepo)
	jobHandler := handler.NewJobHandler(jobService)

	v1 := router.Group("/api/v1")

	// Jobs routes
	jobsGroup := v1.Group("/jobs")
	jobsGroup.Use(middleware.AuthMiddleware(jwtSvc, revokedRepo))
	{
		jobsGroup.POST("", jobHandler.Create)
		jobsGroup.GET("", jobHandler.List)
		jobsGroup.GET("/:id", jobHandler.GetByID)
		jobsGroup.POST("/:id/candidates", candHandler.AddJobCandidate)
		jobsGroup.GET("/:id/candidates", candHandler.ListCandidatesByJob)
		jobsGroup.DELETE("/:id/candidates/:candidateId", candHandler.RemoveJobCandidate)
	}

	// Candidates routes
	candidatesGroup := v1.Group("/candidates")
	candidatesGroup.Use(middleware.AuthMiddleware(jwtSvc, revokedRepo))
	{
		candidatesGroup.POST("", candHandler.Create)
		candidatesGroup.GET("", candHandler.List)
		candidatesGroup.GET("/:id", candHandler.GetByID)
		candidatesGroup.PUT("/:id", candHandler.Update)

		candidatesGroup.GET("/:id/jobs", candHandler.ListJobsByCandidate)

		candidatesGroup.POST("/:id/educations", candHandler.CreateEducation)
		candidatesGroup.GET("/:id/educations", candHandler.ListEducations)
		candidatesGroup.PUT("/:id/educations/:educationId", candHandler.UpdateEducation)
		candidatesGroup.DELETE("/:id/educations/:educationId", candHandler.DeleteEducation)

		candidatesGroup.POST("/:id/experiences", candHandler.CreateExperience)
		candidatesGroup.GET("/:id/experiences", candHandler.ListExperiences)
		candidatesGroup.PUT("/:id/experiences/:experienceId", candHandler.UpdateExperience)
		candidatesGroup.DELETE("/:id/experiences/:experienceId", candHandler.DeleteExperience)

		candidatesGroup.POST("/:id/skills", candHandler.CreateSkill)
		candidatesGroup.GET("/:id/skills", candHandler.ListSkills)
		candidatesGroup.PUT("/:id/skills/:skillId", candHandler.UpdateSkill)
		candidatesGroup.DELETE("/:id/skills/:skillId", candHandler.DeleteSkill)

		candidatesGroup.POST("/:id/documents", candHandler.CreateDocument)
		candidatesGroup.POST("/:id/documents/upload", candHandler.UploadDocument)
		candidatesGroup.POST("/:id/documents/:documentId/process", candHandler.ProcessDocument)
		candidatesGroup.GET("/:id/documents", candHandler.ListDocuments)
		candidatesGroup.GET("/:id/documents/:documentId", candHandler.GetDocumentByID)
		candidatesGroup.DELETE("/:id/documents/:documentId", candHandler.DeleteDocument)
	}

	return router, candRepo, jobRepo, jwtSvc, testRecruiter
}

// helper to execute authenticated requests
func execAuthReq(router *gin.Engine, method, path, token string, body interface{}) *httptest.ResponseRecorder {
	var bodyBuf *bytes.Buffer
	if body != nil {
		jsonBytes, _ := json.Marshal(body)
		bodyBuf = bytes.NewBuffer(jsonBytes)
	} else {
		bodyBuf = bytes.NewBuffer(nil)
	}
	req, _ := http.NewRequest(method, path, bodyBuf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestCandidate_CreateAndOwnership(t *testing.T) {
	router, _, _, jwtSvc, user := setupCandidateTestRouter()
	token, _, _ := jwtSvc.GenerateToken(user)

	payload := map[string]interface{}{
		"full_name":  "Budi Santoso",
		"email":      "budi.santoso@example.com",
		"phone":      "+628123456789",
		"location":   "Jakarta, Indonesia",
		"headline":   "Senior Business Analyst",
		"summary":    "Over 7 years analyzing financial systems and workflows.",
		"created_by": "spoofed-user-id-attempt", // Must be ignored in favor of JWT creator
	}

	w := execAuthReq(router, http.MethodPost, "/api/v1/candidates", token, payload)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data model.Candidate `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Data.FullName != "Budi Santoso" {
		t.Errorf("expected full_name 'Budi Santoso', got '%s'", resp.Data.FullName)
	}
	if resp.Data.CreatedBy != user.ID {
		t.Errorf("expected created_by to be '%s' from JWT, got '%s'", user.ID, resp.Data.CreatedBy)
	}
}

func TestCandidate_Validation(t *testing.T) {
	router, _, _, jwtSvc, user := setupCandidateTestRouter()
	token, _, _ := jwtSvc.GenerateToken(user)

	// Missing full_name
	w1 := execAuthReq(router, http.MethodPost, "/api/v1/candidates", token, map[string]interface{}{
		"full_name": "",
		"email":     "valid@example.com",
	})
	if w1.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty name, got %d", w1.Code)
	}

	// Invalid email format
	w2 := execAuthReq(router, http.MethodPost, "/api/v1/candidates", token, map[string]interface{}{
		"full_name": "Valid Name",
		"email":     "not-an-email",
	})
	if w2.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid email, got %d", w2.Code)
	}
}

func TestCandidate_ListAndFilter(t *testing.T) {
	router, _, _, jwtSvc, user := setupCandidateTestRouter()
	token, _, _ := jwtSvc.GenerateToken(user)

	// Create 2 candidates
	_ = execAuthReq(router, http.MethodPost, "/api/v1/candidates", token, map[string]interface{}{
		"full_name": "Alice Developer",
		"email":     "alice@example.com",
		"location":  "Bandung",
	})
	_ = execAuthReq(router, http.MethodPost, "/api/v1/candidates", token, map[string]interface{}{
		"full_name": "Bob Analyst",
		"email":     "bob@example.com",
		"location":  "Jakarta",
	})

	// List all
	wAll := execAuthReq(router, http.MethodGet, "/api/v1/candidates", token, nil)
	if wAll.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", wAll.Code)
	}

	var respAll struct {
		Data struct {
			Items      []model.Candidate `json:"items"`
			Pagination map[string]int    `json:"pagination"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wAll.Body.Bytes(), &respAll)
	if len(respAll.Data.Items) != 2 {
		t.Errorf("expected 2 candidates, got %d", len(respAll.Data.Items))
	}

	// Search filter
	wSearch := execAuthReq(router, http.MethodGet, "/api/v1/candidates?search=alice", token, nil)
	var respSearch struct {
		Data struct {
			Items []model.Candidate `json:"items"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wSearch.Body.Bytes(), &respSearch)
	if len(respSearch.Data.Items) != 1 || respSearch.Data.Items[0].FullName != "Alice Developer" {
		t.Errorf("expected 1 result matching 'alice', got %d", len(respSearch.Data.Items))
	}
}

func TestCandidate_Update(t *testing.T) {
	router, _, _, jwtSvc, user := setupCandidateTestRouter()
	token, _, _ := jwtSvc.GenerateToken(user)

	// Create candidate
	wCreate := execAuthReq(router, http.MethodPost, "/api/v1/candidates", token, map[string]interface{}{
		"full_name": "Original Name",
		"location":  "Jakarta",
	})
	var created struct {
		Data model.Candidate `json:"data"`
	}
	_ = json.Unmarshal(wCreate.Body.Bytes(), &created)
	candID := created.Data.ID

	// Update candidate
	newName := "Updated Name"
	newLocation := "Surabaya"
	wUpdate := execAuthReq(router, http.MethodPut, "/api/v1/candidates/"+candID, token, map[string]interface{}{
		"full_name": newName,
		"location":  newLocation,
	})
	if wUpdate.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d: %s", wUpdate.Code, wUpdate.Body.String())
	}

	var updated struct {
		Data model.Candidate `json:"data"`
	}
	_ = json.Unmarshal(wUpdate.Body.Bytes(), &updated)
	if updated.Data.FullName != newName {
		t.Errorf("expected full_name '%s', got '%s'", newName, updated.Data.FullName)
	}
	if updated.Data.Location != newLocation {
		t.Errorf("expected location '%s', got '%s'", newLocation, updated.Data.Location)
	}
}

func TestJobCandidate_AssociationLifecycle(t *testing.T) {
	router, _, jobRepo, jwtSvc, user := setupCandidateTestRouter()
	token, _, _ := jwtSvc.GenerateToken(user)

	// Seed a job
	job := &model.Job{
		ID:             uuid.New().String(),
		Code:           "JOB-2026-0001",
		Title:          "Senior Backend Engineer",
		Description:    "Go microservices",
		Department:     "Engineering",
		Location:       "Jakarta",
		EmploymentType: model.EmpFullTime,
		Status:         model.JobStatusOpen,
		CreatedBy:      user.ID,
	}
	_ = jobRepo.Create(context.Background(), job)

	// Create a candidate
	wCand := execAuthReq(router, http.MethodPost, "/api/v1/candidates", token, map[string]interface{}{
		"full_name": "Charlie Engineer",
		"email":     "charlie@example.com",
	})
	var candResp struct {
		Data model.Candidate `json:"data"`
	}
	_ = json.Unmarshal(wCand.Body.Bytes(), &candResp)
	candID := candResp.Data.ID

	// 1. Link candidate to job
	wLink := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/jobs/%s/candidates", job.ID), token, map[string]interface{}{
		"candidate_id": candID,
	})
	if wLink.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for job-candidate link, got %d: %s", wLink.Code, wLink.Body.String())
	}

	// 2. Duplicate link returns 409 Conflict
	wDup := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/jobs/%s/candidates", job.ID), token, map[string]interface{}{
		"candidate_id": candID,
	})
	if wDup.Code != http.StatusConflict {
		t.Errorf("expected 409 Conflict on duplicate link, got %d", wDup.Code)
	}

	// 3. List candidates for job
	wListCand := execAuthReq(router, http.MethodGet, fmt.Sprintf("/api/v1/jobs/%s/candidates", job.ID), token, nil)
	if wListCand.Code != http.StatusOK {
		t.Fatalf("expected 200 OK listing job candidates, got %d", wListCand.Code)
	}
	var jobCandList struct {
		Data struct {
			Items []model.Candidate `json:"items"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wListCand.Body.Bytes(), &jobCandList)
	if len(jobCandList.Data.Items) != 1 || jobCandList.Data.Items[0].ID != candID {
		t.Errorf("expected candidate '%s' in job candidate list, got %v", candID, jobCandList.Data.Items)
	}

	// 4. List jobs for candidate
	wListJobs := execAuthReq(router, http.MethodGet, fmt.Sprintf("/api/v1/candidates/%s/jobs", candID), token, nil)
	if wListJobs.Code != http.StatusOK {
		t.Fatalf("expected 200 OK listing candidate jobs, got %d", wListJobs.Code)
	}
	var candJobsList struct {
		Data []model.Job `json:"data"`
	}
	_ = json.Unmarshal(wListJobs.Body.Bytes(), &candJobsList)
	if len(candJobsList.Data) != 1 || candJobsList.Data[0].ID != job.ID {
		t.Errorf("expected job '%s' in candidate jobs list, got %v", job.ID, candJobsList.Data)
	}

	// 5. Unlink candidate from job
	wUnlink := execAuthReq(router, http.MethodDelete, fmt.Sprintf("/api/v1/jobs/%s/candidates/%s", job.ID, candID), token, nil)
	if wUnlink.Code != http.StatusOK {
		t.Fatalf("expected 200 OK unlinking candidate, got %d", wUnlink.Code)
	}

	// 6. Verify candidate is unlinked from job
	wListAfter := execAuthReq(router, http.MethodGet, fmt.Sprintf("/api/v1/jobs/%s/candidates", job.ID), token, nil)
	var afterList struct {
		Data struct {
			Items []model.Candidate `json:"items"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wListAfter.Body.Bytes(), &afterList)
	if len(afterList.Data.Items) != 0 {
		t.Errorf("expected 0 candidates after unlink, got %d", len(afterList.Data.Items))
	}

	// 7. Verify candidate still exists independently in system
	wCheckCand := execAuthReq(router, http.MethodGet, fmt.Sprintf("/api/v1/candidates/%s", candID), token, nil)
	if wCheckCand.Code != http.StatusOK {
		t.Errorf("candidate was unexpectedly deleted after job unlinking, code: %d", wCheckCand.Code)
	}
}

func TestEducation_CRUDAndValidation(t *testing.T) {
	router, _, _, jwtSvc, user := setupCandidateTestRouter()
	token, _, _ := jwtSvc.GenerateToken(user)

	// Create candidate
	wCand := execAuthReq(router, http.MethodPost, "/api/v1/candidates", token, map[string]interface{}{
		"full_name": "Edu Test Candidate",
	})
	var candResp struct {
		Data model.Candidate `json:"data"`
	}
	_ = json.Unmarshal(wCand.Body.Bytes(), &candResp)
	candID := candResp.Data.ID

	// 1. Create education with invalid date order (end before start)
	wBadDate := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/candidates/%s/educations", candID), token, map[string]interface{}{
		"institution": "Tech University",
		"degree":      "Bachelor",
		"start_date":  "2022-01-01",
		"end_date":    "2020-01-01",
	})
	if wBadDate.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid date order, got %d", wBadDate.Code)
	}

	// 2. Create valid education
	wCreate := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/candidates/%s/educations", candID), token, map[string]interface{}{
		"institution":    "Universitas Indonesia",
		"degree":         "Bachelor of Computer Science",
		"field_of_study": "Software Engineering",
		"start_date":     "2016-08-01",
		"end_date":       "2020-07-01",
		"description":    "Graduated Cum Laude",
	})
	if wCreate.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", wCreate.Code, wCreate.Body.String())
	}
	var eduResp struct {
		Data model.CandidateEducation `json:"data"`
	}
	_ = json.Unmarshal(wCreate.Body.Bytes(), &eduResp)
	eduID := eduResp.Data.ID

	// 3. List educations
	wList := execAuthReq(router, http.MethodGet, fmt.Sprintf("/api/v1/candidates/%s/educations", candID), token, nil)
	if wList.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", wList.Code)
	}
	var listResp struct {
		Data []model.CandidateEducation `json:"data"`
	}
	_ = json.Unmarshal(wList.Body.Bytes(), &listResp)
	if len(listResp.Data) != 1 {
		t.Errorf("expected 1 education record, got %d", len(listResp.Data))
	}

	// 4. Update education
	updatedInst := "Universitas Indonesia (UI)"
	wUpdate := execAuthReq(router, http.MethodPut, fmt.Sprintf("/api/v1/candidates/%s/educations/%s", candID, eduID), token, map[string]interface{}{
		"institution": updatedInst,
	})
	if wUpdate.Code != http.StatusOK {
		t.Fatalf("expected 200 OK updating education, got %d", wUpdate.Code)
	}

	// 5. Delete education
	wDel := execAuthReq(router, http.MethodDelete, fmt.Sprintf("/api/v1/candidates/%s/educations/%s", candID, eduID), token, nil)
	if wDel.Code != http.StatusOK {
		t.Fatalf("expected 200 OK deleting education, got %d", wDel.Code)
	}
}

func TestExperience_CRUDAndValidation(t *testing.T) {
	router, _, _, jwtSvc, user := setupCandidateTestRouter()
	token, _, _ := jwtSvc.GenerateToken(user)

	// Create candidate
	wCand := execAuthReq(router, http.MethodPost, "/api/v1/candidates", token, map[string]interface{}{
		"full_name": "Exp Test Candidate",
	})
	var candResp struct {
		Data model.Candidate `json:"data"`
	}
	_ = json.Unmarshal(wCand.Body.Bytes(), &candResp)
	candID := candResp.Data.ID

	// 1. Missing company/position
	wBad := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/candidates/%s/experiences", candID), token, map[string]interface{}{
		"company": "",
	})
	if wBad.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty company, got %d", wBad.Code)
	}

	// 2. Create valid experience
	wCreate := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/candidates/%s/experiences", candID), token, map[string]interface{}{
		"company":         "FinTech Corp",
		"position":        "Lead Backend Engineer",
		"location":        "Jakarta",
		"employment_type": "FULL_TIME",
		"start_date":      "2021-01-01",
		"is_current":      true,
		"description":     "Leading 6 engineers in payment gateway migration.",
	})
	if wCreate.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for experience, got %d: %s", wCreate.Code, wCreate.Body.String())
	}
	var expResp struct {
		Data model.CandidateExperience `json:"data"`
	}
	_ = json.Unmarshal(wCreate.Body.Bytes(), &expResp)
	expID := expResp.Data.ID

	// 3. Delete experience
	wDel := execAuthReq(router, http.MethodDelete, fmt.Sprintf("/api/v1/candidates/%s/experiences/%s", candID, expID), token, nil)
	if wDel.Code != http.StatusOK {
		t.Fatalf("expected 200 OK deleting experience, got %d", wDel.Code)
	}
}

func TestSkill_DuplicateCaseInsensitive(t *testing.T) {
	router, _, _, jwtSvc, user := setupCandidateTestRouter()
	token, _, _ := jwtSvc.GenerateToken(user)

	// Create candidate
	wCand := execAuthReq(router, http.MethodPost, "/api/v1/candidates", token, map[string]interface{}{
		"full_name": "Skill Test Candidate",
	})
	var candResp struct {
		Data model.Candidate `json:"data"`
	}
	_ = json.Unmarshal(wCand.Body.Bytes(), &candResp)
	candID := candResp.Data.ID

	// 1. Create skill "Go"
	w1 := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/candidates/%s/skills", candID), token, map[string]interface{}{
		"skill": "Go",
	})
	if w1.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for skill, got %d", w1.Code)
	}

	// 2. Attempt duplicate "go" in different case -> 409 Conflict
	w2 := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/candidates/%s/skills", candID), token, map[string]interface{}{
		"skill": "go",
	})
	if w2.Code != http.StatusConflict {
		t.Errorf("expected 409 Conflict for duplicate case-insensitive skill, got %d", w2.Code)
	}
}

func TestDocument_TextCreationAndRejectionOfPDFAndDOCX(t *testing.T) {
	router, _, _, jwtSvc, user := setupCandidateTestRouter()
	token, _, _ := jwtSvc.GenerateToken(user)

	// Create candidate
	wCand := execAuthReq(router, http.MethodPost, "/api/v1/candidates", token, map[string]interface{}{
		"full_name": "Doc Test Candidate",
	})
	var candResp struct {
		Data model.Candidate `json:"data"`
	}
	_ = json.Unmarshal(wCand.Body.Bytes(), &candResp)
	candID := candResp.Data.ID

	// 1. Explicitly reject PDF in direct JSON creation (requires upload endpoint)
	wPDF := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/candidates/%s/documents", candID), token, map[string]interface{}{
		"source_type": "PDF",
		"raw_text":    "some content",
	})
	if wPDF.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for PDF document in direct JSON creation, got %d", wPDF.Code)
	}
	if !strings.Contains(wPDF.Body.String(), "upload") {
		t.Errorf("expected message referencing upload endpoint, got %s", wPDF.Body.String())
	}

	// 2. Explicitly reject DOCX in direct JSON creation
	wDOCX := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/candidates/%s/documents", candID), token, map[string]interface{}{
		"source_type": "DOCX",
		"raw_text":    "some content",
	})
	if wDOCX.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for DOCX document in direct JSON creation, got %d", wDOCX.Code)
	}

	// 3. Create valid TEXT document
	rawCV := "Budi Santoso\nSenior Business Analyst\nSummary: 7+ years evaluating workflows\nSkills: SQL, Agile, BPMN"
	wText := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/candidates/%s/documents", candID), token, map[string]interface{}{
		"source_type": "TEXT",
		"raw_text":    rawCV,
	})
	if wText.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for TEXT document, got %d: %s", wText.Code, wText.Body.String())
	}

	var docResp struct {
		Data model.CandidateDocument `json:"data"`
	}
	_ = json.Unmarshal(wText.Body.Bytes(), &docResp)
	if docResp.Data.SourceType != model.SourceTypeText {
		t.Errorf("expected source_type 'TEXT', got '%s'", docResp.Data.SourceType)
	}
	if docResp.Data.RawText != rawCV {
		t.Errorf("expected raw_text to match submitted text")
	}

	// 4. Retrieve document by ID
	docID := docResp.Data.ID
	wGetDoc := execAuthReq(router, http.MethodGet, fmt.Sprintf("/api/v1/candidates/%s/documents/%s", candID, docID), token, nil)
	if wGetDoc.Code != http.StatusOK {
		t.Fatalf("expected 200 OK retrieving document, got %d", wGetDoc.Code)
	}

	// 5. Delete document
	wDelDoc := execAuthReq(router, http.MethodDelete, fmt.Sprintf("/api/v1/candidates/%s/documents/%s", candID, docID), token, nil)
	if wDelDoc.Code != http.StatusOK {
		t.Fatalf("expected 200 OK deleting document, got %d", wDelDoc.Code)
	}
}

func TestCandidate_UnauthenticatedForbidden(t *testing.T) {
	router, _, _, _, _ := setupCandidateTestRouter()

	// Candidate listing without token
	w := execAuthReq(router, http.MethodGet, "/api/v1/candidates", "", nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized for unauthenticated request, got %d", w.Code)
	}
}

func createCandidateTestDocx(content string) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	ct, _ := zw.Create("[Content_Types].xml")
	_, _ = ct.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="xml" ContentType="application/xml"/>
</Types>`))

	doc, _ := zw.Create("word/document.xml")
	_, _ = doc.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p><w:r><w:t>` + content + `</w:t></w:r></w:p>
  </w:body>
</w:document>`))

	_ = zw.Close()
	return buf.Bytes()
}

func createCandidateTestPDF(content string) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")

	offsets := make([]int, 6)

	// obj 1: Catalog
	offsets[1] = b.Len()
	b.WriteString("1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n")

	// obj 2: Pages
	offsets[2] = b.Len()
	b.WriteString("2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n")

	// obj 3: Page
	offsets[3] = b.Len()
	b.WriteString("3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>\nendobj\n")

	// obj 4: Stream
	streamContent := "BT /F1 12 Tf 72 712 Td (" + content + ") Tj ET\n"
	offsets[4] = b.Len()
	b.WriteString(fmt.Sprintf("4 0 obj\n<< /Length %d >>\nstream\n%sendstream\nendobj\n", len(streamContent), streamContent))

	// obj 5: Font
	offsets[5] = b.Len()
	b.WriteString("5 0 obj\n<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>\nendobj\n")

	// xref
	xrefOffset := b.Len()
	b.WriteString("xref\n0 6\n0000000000 65535 f \n")
	for i := 1; i <= 5; i++ {
		b.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	b.WriteString(fmt.Sprintf("trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", xrefOffset))

	return b.Bytes()
}

func execMultipartUpload(router *gin.Engine, path, token, fieldName, filename string, fileBytes []byte) *httptest.ResponseRecorder {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	part, _ := w.CreateFormFile(fieldName, filename)
	_, _ = part.Write(fileBytes)
	_ = w.Close()

	req, _ := http.NewRequest(http.MethodPost, path, &b)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestDocument_UploadPDFAndDOCX(t *testing.T) {
	router, _, _, jwtSvc, user := setupCandidateTestRouter()
	token, _, _ := jwtSvc.GenerateToken(user)

	// Create candidate
	wCand := execAuthReq(router, http.MethodPost, "/api/v1/candidates", token, map[string]interface{}{
		"full_name": "Upload Test Candidate",
	})
	var candResp struct {
		Data model.Candidate `json:"data"`
	}
	_ = json.Unmarshal(wCand.Body.Bytes(), &candResp)
	candID := candResp.Data.ID

	// 1. Upload valid PDF
	pdfBytes := createCandidateTestPDF("Candidate Resume Content")
	wPDF := execMultipartUpload(router, fmt.Sprintf("/api/v1/candidates/%s/documents/upload", candID), token, "file", "my_resume.pdf", pdfBytes)
	if wPDF.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created uploading PDF, got %d: %s", wPDF.Code, wPDF.Body.String())
	}
	var pdfDocResp struct {
		Data model.CandidateDocument `json:"data"`
	}
	_ = json.Unmarshal(wPDF.Body.Bytes(), &pdfDocResp)
	if pdfDocResp.Data.SourceType != model.SourceTypePDF {
		t.Errorf("expected source_type 'PDF', got '%s'", pdfDocResp.Data.SourceType)
	}
	if pdfDocResp.Data.StoragePath == nil || *pdfDocResp.Data.StoragePath == "" {
		t.Errorf("expected storage_path to be generated")
	}

	// 2. Upload valid DOCX
	docxBytes := createCandidateTestDocx("Candidate DOCX Content")
	wDOCX := execMultipartUpload(router, fmt.Sprintf("/api/v1/candidates/%s/documents/upload", candID), token, "file", "my_resume.docx", docxBytes)
	if wDOCX.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created uploading DOCX, got %d: %s", wDOCX.Code, wDOCX.Body.String())
	}
	var docxDocResp struct {
		Data model.CandidateDocument `json:"data"`
	}
	_ = json.Unmarshal(wDOCX.Body.Bytes(), &docxDocResp)
	if docxDocResp.Data.SourceType != model.SourceTypeDOCX {
		t.Errorf("expected source_type 'DOCX', got '%s'", docxDocResp.Data.SourceType)
	}
}

func TestDocument_UploadValidationErrors(t *testing.T) {
	router, _, _, jwtSvc, user := setupCandidateTestRouter()
	token, _, _ := jwtSvc.GenerateToken(user)

	wCand := execAuthReq(router, http.MethodPost, "/api/v1/candidates", token, map[string]interface{}{
		"full_name": "Validation Test Candidate",
	})
	var candResp struct {
		Data model.Candidate `json:"data"`
	}
	_ = json.Unmarshal(wCand.Body.Bytes(), &candResp)
	candID := candResp.Data.ID

	// 1. Invalid extension
	w1 := execMultipartUpload(router, fmt.Sprintf("/api/v1/candidates/%s/documents/upload", candID), token, "file", "resume.exe", []byte("executable"))
	if w1.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for .exe file, got %d", w1.Code)
	}

	// 2. Fake PDF (not starting with %PDF-)
	w2 := execMultipartUpload(router, fmt.Sprintf("/api/v1/candidates/%s/documents/upload", candID), token, "file", "fake.pdf", []byte("plain text not a pdf"))
	if w2.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for fake PDF signature, got %d", w2.Code)
	}

	// 3. Fake DOCX (arbitrary zip without document.xml)
	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	f, _ := zw.Create("hello.txt")
	_, _ = f.Write([]byte("not a word document"))
	_ = zw.Close()

	w3 := execMultipartUpload(router, fmt.Sprintf("/api/v1/candidates/%s/documents/upload", candID), token, "file", "fake.docx", zipBuf.Bytes())
	if w3.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for arbitrary zip renamed to .docx, got %d", w3.Code)
	}

	// 4. Missing file form field
	w4 := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/candidates/%s/documents/upload", candID), token, nil)
	if w4.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing file in upload, got %d", w4.Code)
	}
}

func TestDocument_ProcessTEXT_EndToEnd(t *testing.T) {
	router, _, _, jwtSvc, user := setupCandidateTestRouter()
	token, _, _ := jwtSvc.GenerateToken(user)

	// Create candidate initially without headline/summary
	wCand := execAuthReq(router, http.MethodPost, "/api/v1/candidates", token, map[string]interface{}{
		"full_name": "Budi Santoso",
	})
	var candResp struct {
		Data model.Candidate `json:"data"`
	}
	_ = json.Unmarshal(wCand.Body.Bytes(), &candResp)
	candID := candResp.Data.ID

	// Create TEXT document
	cvText := "Budi Santoso\nSenior Business Analyst\nEmail: budi.analyst@example.com\nPhone: +62 812 3456 7890\n\nSUMMARY\nExperienced Business Analyst specializing in financial systems.\n\nWORK EXPERIENCE\n2021 - Present\nBank Mandiri\nSenior Business Analyst\nLed ERP transformation.\n\nEDUCATION\nUniversitas Indonesia\nBachelor of Computer Science\n2015 - 2019\n\nSKILLS\nBPMN, SQL, Agile, JIRA"
	wDoc := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/candidates/%s/documents", candID), token, map[string]interface{}{
		"source_type": "TEXT",
		"raw_text":    cvText,
	})
	var docResp struct {
		Data model.CandidateDocument `json:"data"`
	}
	_ = json.Unmarshal(wDoc.Body.Bytes(), &docResp)
	docID := docResp.Data.ID

	// Process document
	wProc := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/candidates/%s/documents/%s/process", candID, docID), token, nil)
	if wProc.Code != http.StatusOK {
		t.Fatalf("expected 200 OK processing TEXT document, got %d: %s", wProc.Code, wProc.Body.String())
	}

	var procResp struct {
		Data service.ProcessDocumentResult `json:"data"`
	}
	if err := json.Unmarshal(wProc.Body.Bytes(), &procResp); err != nil {
		t.Fatalf("failed to decode process response: %v", err)
	}

	if procResp.Data.Extraction.Status != "SUCCESS" {
		t.Errorf("expected extraction status SUCCESS, got: %s", procResp.Data.Extraction.Status)
	}
	if procResp.Data.Parsing.Status != "COMPLETED" {
		t.Errorf("expected parsing status COMPLETED, got: %s", procResp.Data.Parsing.Status)
	}

	cand := procResp.Data.Candidate
	if cand.Email != "budi.analyst@example.com" {
		t.Errorf("expected email to be populated: '%s'", cand.Email)
	}
	if cand.Headline != "Senior Business Analyst" {
		t.Errorf("expected headline to be populated: '%s'", cand.Headline)
	}
	if len(cand.Skills) < 3 {
		t.Errorf("expected at least 3 skills populated, got %d", len(cand.Skills))
	}
	if len(cand.Educations) != 1 {
		t.Errorf("expected 1 education populated, got %d", len(cand.Educations))
	}
	if len(cand.Experiences) != 1 {
		t.Errorf("expected 1 experience populated, got %d", len(cand.Experiences))
	}
}

func TestDocument_ProcessPDF_EndToEnd(t *testing.T) {
	router, _, _, jwtSvc, user := setupCandidateTestRouter()
	token, _, _ := jwtSvc.GenerateToken(user)

	wCand := execAuthReq(router, http.MethodPost, "/api/v1/candidates", token, map[string]interface{}{
		"full_name": "PDF Candidate",
	})
	var candResp struct {
		Data model.Candidate `json:"data"`
	}
	_ = json.Unmarshal(wCand.Body.Bytes(), &candResp)
	candID := candResp.Data.ID

	// Upload valid PDF
	pdfContent := "Alice Smith email: alice.smith@example.com phone: +62 811 2233 4455"
	pdfBytes := createCandidateTestPDF(pdfContent)
	wUpload := execMultipartUpload(router, fmt.Sprintf("/api/v1/candidates/%s/documents/upload", candID), token, "file", "alice.pdf", pdfBytes)
	if wUpload.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created uploading PDF, got %d: %s", wUpload.Code, wUpload.Body.String())
	}
	var docResp struct {
		Data model.CandidateDocument `json:"data"`
	}
	_ = json.Unmarshal(wUpload.Body.Bytes(), &docResp)
	docID := docResp.Data.ID

	// Process PDF
	wProc := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/candidates/%s/documents/%s/process", candID, docID), token, nil)
	if wProc.Code != http.StatusOK {
		t.Fatalf("expected 200 OK processing PDF, got %d: %s", wProc.Code, wProc.Body.String())
	}

	var procResp struct {
		Data service.ProcessDocumentResult `json:"data"`
	}
	_ = json.Unmarshal(wProc.Body.Bytes(), &procResp)
	if procResp.Data.Candidate.Email != "alice.smith@example.com" {
		t.Errorf("expected email 'alice.smith@example.com', got '%s'", procResp.Data.Candidate.Email)
	}
}

func TestDocument_ProcessDOCX_EndToEnd(t *testing.T) {
	router, _, _, jwtSvc, user := setupCandidateTestRouter()
	token, _, _ := jwtSvc.GenerateToken(user)

	wCand := execAuthReq(router, http.MethodPost, "/api/v1/candidates", token, map[string]interface{}{
		"full_name": "DOCX Candidate",
	})
	var candResp struct {
		Data model.Candidate `json:"data"`
	}
	_ = json.Unmarshal(wCand.Body.Bytes(), &candResp)
	candID := candResp.Data.ID

	// Upload valid DOCX
	docxContent := "Charlie Brown email: charlie@example.com SKILLS Go, Kubernetes, Terraform"
	docxBytes := createCandidateTestDocx(docxContent)
	wUpload := execMultipartUpload(router, fmt.Sprintf("/api/v1/candidates/%s/documents/upload", candID), token, "file", "charlie.docx", docxBytes)
	if wUpload.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created uploading DOCX, got %d: %s", wUpload.Code, wUpload.Body.String())
	}
	var docResp struct {
		Data model.CandidateDocument `json:"data"`
	}
	_ = json.Unmarshal(wUpload.Body.Bytes(), &docResp)
	docID := docResp.Data.ID

	// Process DOCX
	wProc := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/candidates/%s/documents/%s/process", candID, docID), token, nil)
	if wProc.Code != http.StatusOK {
		t.Fatalf("expected 200 OK processing DOCX, got %d: %s", wProc.Code, wProc.Body.String())
	}

	var procResp struct {
		Data service.ProcessDocumentResult `json:"data"`
	}
	_ = json.Unmarshal(wProc.Body.Bytes(), &procResp)
	if procResp.Data.Candidate.Email != "charlie@example.com" {
		t.Errorf("expected email 'charlie@example.com', got '%s'", procResp.Data.Candidate.Email)
	}
}

func TestDocument_ProcessIdempotency(t *testing.T) {
	router, _, _, jwtSvc, user := setupCandidateTestRouter()
	token, _, _ := jwtSvc.GenerateToken(user)

	wCand := execAuthReq(router, http.MethodPost, "/api/v1/candidates", token, map[string]interface{}{
		"full_name": "Idempotent Candidate",
	})
	var candResp struct {
		Data model.Candidate `json:"data"`
	}
	_ = json.Unmarshal(wCand.Body.Bytes(), &candResp)
	candID := candResp.Data.ID

	cvText := "Idempotent Candidate\nEmail: idemp@example.com\n\nEDUCATION\nBandung Institute of Technology\nBachelor\n2018 - 2022\n\nWORK EXPERIENCE\n2022 - Present\nTech Asia\nSoftware Engineer\nDeveloping backend microservices.\n\nSKILLS\nGo, PostgreSQL, Docker"
	wDoc := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/candidates/%s/documents", candID), token, map[string]interface{}{
		"source_type": "TEXT",
		"raw_text":    cvText,
	})
	var docResp struct {
		Data model.CandidateDocument `json:"data"`
	}
	_ = json.Unmarshal(wDoc.Body.Bytes(), &docResp)
	docID := docResp.Data.ID

	// Process 1st time
	wProc1 := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/candidates/%s/documents/%s/process", candID, docID), token, nil)
	if wProc1.Code != http.StatusOK {
		t.Fatalf("expected 200 on 1st process, got %d", wProc1.Code)
	}

	// Process 2nd time (re-processing)
	wProc2 := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/candidates/%s/documents/%s/process", candID, docID), token, nil)
	if wProc2.Code != http.StatusOK {
		t.Fatalf("expected 200 on 2nd process, got %d", wProc2.Code)
	}

	var procResp struct {
		Data service.ProcessDocumentResult `json:"data"`
	}
	_ = json.Unmarshal(wProc2.Body.Bytes(), &procResp)
	cand := procResp.Data.Candidate

	// Verify counts have not doubled
	if len(cand.Skills) != 3 {
		t.Errorf("expected exactly 3 skills after reprocessing, got %d: %v", len(cand.Skills), cand.Skills)
	}
	if len(cand.Educations) != 1 {
		t.Errorf("expected exactly 1 education after reprocessing, got %d", len(cand.Educations))
	}
	if len(cand.Experiences) != 1 {
		t.Errorf("expected exactly 1 experience after reprocessing, got %d", len(cand.Experiences))
	}
}

func TestDocument_ProcessManualDataProtection(t *testing.T) {
	router, _, _, jwtSvc, user := setupCandidateTestRouter()
	token, _, _ := jwtSvc.GenerateToken(user)

	manualEmail := "recruiter.verified@example.com"
	manualPhone := "+62 811 9999 8888"
	manualHeadline := "Manually Verified Senior Architect"

	// Recruiter manually created candidate with specific verified details
	wCand := execAuthReq(router, http.MethodPost, "/api/v1/candidates", token, map[string]interface{}{
		"full_name": "Original Candidate Name",
		"email":     manualEmail,
		"phone":     manualPhone,
		"headline":  manualHeadline,
	})
	var candResp struct {
		Data model.Candidate `json:"data"`
	}
	_ = json.Unmarshal(wCand.Body.Bytes(), &candResp)
	candID := candResp.Data.ID

	// Document contains different email and phone
	cvText := "Extracted Name\nEmail: parsed.different@example.com\nPhone: +1 555 123 4567\n\nSUMMARY\nExtracted summary from CV"
	wDoc := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/candidates/%s/documents", candID), token, map[string]interface{}{
		"source_type": "TEXT",
		"raw_text":    cvText,
	})
	var docResp struct {
		Data model.CandidateDocument `json:"data"`
	}
	_ = json.Unmarshal(wDoc.Body.Bytes(), &docResp)
	docID := docResp.Data.ID

	// Process document
	wProc := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/candidates/%s/documents/%s/process", candID, docID), token, nil)
	if wProc.Code != http.StatusOK {
		t.Fatalf("expected 200 on process, got %d", wProc.Code)
	}

	var procResp struct {
		Data service.ProcessDocumentResult `json:"data"`
	}
	_ = json.Unmarshal(wProc.Body.Bytes(), &procResp)
	cand := procResp.Data.Candidate

	// Verify manual fields were preserved and NOT overwritten
	if cand.Email != manualEmail {
		t.Errorf("manual email was overwritten! expected '%s', got '%s'", manualEmail, cand.Email)
	}
	if cand.Phone != manualPhone {
		t.Errorf("manual phone was overwritten! expected '%s', got '%s'", manualPhone, cand.Phone)
	}
	if cand.Headline != manualHeadline {
		t.Errorf("manual headline was overwritten! expected '%s', got '%s'", manualHeadline, cand.Headline)
	}
	// Verify missing field (summary) was filled
	if cand.Summary != "Extracted summary from CV" {
		t.Errorf("expected missing summary to be filled from CV, got '%s'", cand.Summary)
	}
}

func TestDocument_CandidateMismatch(t *testing.T) {
	router, _, _, jwtSvc, user := setupCandidateTestRouter()
	token, _, _ := jwtSvc.GenerateToken(user)

	// Create candidate 1
	w1 := execAuthReq(router, http.MethodPost, "/api/v1/candidates", token, map[string]interface{}{
		"full_name": "Candidate One",
	})
	var c1 struct {
		Data model.Candidate `json:"data"`
	}
	_ = json.Unmarshal(w1.Body.Bytes(), &c1)

	// Create candidate 2
	w2 := execAuthReq(router, http.MethodPost, "/api/v1/candidates", token, map[string]interface{}{
		"full_name": "Candidate Two",
	})
	var c2 struct {
		Data model.Candidate `json:"data"`
	}
	_ = json.Unmarshal(w2.Body.Bytes(), &c2)

	// Create document for candidate 1
	wDoc := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/candidates/%s/documents", c1.Data.ID), token, map[string]interface{}{
		"source_type": "TEXT",
		"raw_text":    "Resume text for Candidate 1",
	})
	var docResp struct {
		Data model.CandidateDocument `json:"data"`
	}
	_ = json.Unmarshal(wDoc.Body.Bytes(), &docResp)

	// Attempt to process Candidate 1's document under Candidate 2's URL
	wMismatch := execAuthReq(router, http.MethodPost, fmt.Sprintf("/api/v1/candidates/%s/documents/%s/process", c2.Data.ID, docResp.Data.ID), token, nil)
	if wMismatch.Code != http.StatusBadRequest {
		t.Errorf("expected 400 Bad Request for document candidate mismatch, got %d", wMismatch.Code)
	}
}
