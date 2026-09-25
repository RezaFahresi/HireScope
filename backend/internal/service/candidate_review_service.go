package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"hirescope/backend/internal/model"
	"hirescope/backend/internal/repository"

	"github.com/google/uuid"
)

var (
	ErrNoteForbidden = errors.New("you do not have permission to modify this note")
	ErrInvalidStatus = errors.New("invalid candidate workflow status")
	ErrValidation    = errors.New("validation error")
)

// UserSummaryDTO provides safe reviewer/author identity details.
type UserSummaryDTO struct {
	ID    string     `json:"id"`
	Name  string     `json:"name"`
	Email string     `json:"email"`
	Role  model.Role `json:"role,omitempty"`
}

// CandidateSummaryDTO represents safe candidate identity fields in list representations.
type CandidateSummaryDTO struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
	Email    string `json:"email,omitempty"`
	Phone    string `json:"phone,omitempty"`
	Location string `json:"location,omitempty"`
	Headline string `json:"headline,omitempty"`
}

// ScreeningCountsDTO encapsulates partitioned requirement match counters.
type ScreeningCountsDTO struct {
	Match    int `json:"match"`
	Partial  int `json:"partial"`
	Mismatch int `json:"mismatch"`
	Unknown  int `json:"unknown"`
}

// ScreeningSummaryDTO provides a lightweight summary of candidate screening status.
type ScreeningSummaryDTO struct {
	Status      model.ScreeningResultStatus `json:"status"`
	Required    ScreeningCountsDTO          `json:"required"`
	Preferred   ScreeningCountsDTO          `json:"preferred"`
	EvaluatedAt time.Time                   `json:"evaluated_at"`
}

// JobCandidateItemDTO represents a candidate within a job's review list.
type JobCandidateItemDTO struct {
	ID         string                   `json:"id"`
	FullName   string                   `json:"full_name"`
	Email      string                   `json:"email,omitempty"`
	Phone      string                   `json:"phone,omitempty"`
	Location   string                   `json:"location,omitempty"`
	Headline   string                   `json:"headline,omitempty"`
	Candidate  CandidateSummaryDTO      `json:"candidate"`
	Status     model.JobCandidateStatus `json:"status"`
	ReviewedAt *time.Time               `json:"reviewed_at,omitempty"`
	ReviewedBy *UserSummaryDTO          `json:"reviewed_by,omitempty"`
	Screening  *ScreeningSummaryDTO     `json:"screening"`
}

// JobCandidatePageResponse contains paginated job candidates with review metadata.
type JobCandidatePageResponse struct {
	Items      []JobCandidateItemDTO `json:"items"`
	Total      int64                 `json:"total"`
	Page       int                   `json:"page"`
	Limit      int                   `json:"limit"`
	TotalPages int                   `json:"total_pages"`
}

// CandidateNoteDTO represents a job-specific recruiter note.
type CandidateNoteDTO struct {
	ID          string          `json:"id"`
	JobID       string          `json:"job_id"`
	CandidateID string          `json:"candidate_id"`
	Author      *UserSummaryDTO `json:"author,omitempty"`
	Content     string          `json:"content"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// CandidateNotePageResponse encapsulates paginated notes for a candidate under a specific job.
type CandidateNotePageResponse struct {
	Items      []CandidateNoteDTO `json:"items"`
	Total      int64              `json:"total"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
	TotalPages int                `json:"total_pages"`
}

// UpdateStatusResponse represents the response payload after updating a candidate's workflow status.
type UpdateStatusResponse struct {
	JobID       string                   `json:"job_id"`
	CandidateID string                   `json:"candidate_id"`
	Status      model.JobCandidateStatus `json:"status"`
	ReviewedBy  *UserSummaryDTO          `json:"reviewed_by,omitempty"`
	ReviewedAt  *time.Time               `json:"reviewed_at,omitempty"`
}

// DocumentSummaryDTO presents document metadata without exposing internal filesystem paths or raw binary bytes.
type DocumentSummaryDTO struct {
	ID         string                   `json:"id"`
	SourceType model.DocumentSourceType `json:"source_type"`
	FileName   *string                  `json:"file_name,omitempty"`
	MimeType   *string                  `json:"mime_type,omitempty"`
	FileSize   *int64                   `json:"file_size,omitempty"`
	CreatedAt  time.Time                `json:"created_at"`
}

// ReviewDetailResponse provides a complete, aggregated recruiter review context snapshot.
type ReviewDetailResponse struct {
	Job struct {
		ID             string               `json:"id"`
		Title          string               `json:"title"`
		Status         model.JobStatus      `json:"status"`
		Department     string               `json:"department"`
		Location       string               `json:"location"`
		EmploymentType model.EmploymentType `json:"employment_type"`
	} `json:"job"`
	Candidate struct {
		ID       string `json:"id"`
		FullName string `json:"full_name"`
		Email    string `json:"email,omitempty"`
		Phone    string `json:"phone,omitempty"`
		Location string `json:"location,omitempty"`
		Headline string `json:"headline,omitempty"`
		Summary  string `json:"summary,omitempty"`
	} `json:"candidate"`
	Workflow struct {
		Status     model.JobCandidateStatus `json:"status"`
		ReviewedAt *time.Time               `json:"reviewed_at,omitempty"`
		ReviewedBy *UserSummaryDTO          `json:"reviewed_by,omitempty"`
	} `json:"workflow"`
	Documents  []DocumentSummaryDTO        `json:"documents"`
	Education  []model.CandidateEducation  `json:"education"`
	Experience []model.CandidateExperience `json:"experience"`
	Skills     []model.CandidateSkill      `json:"skills"`
	Screening  *ScreeningResponse          `json:"screening"`
	Notes      []CandidateNoteDTO          `json:"notes"`
}

// JobCandidateFilterInput encapsulates query criteria for listing candidates of a job.
type JobCandidateFilterInput struct {
	JobID  string
	Status string
	Search string
	Page   int
	Limit  int
	Sort   string
	Order  string
}

// CandidateReviewService defines the business operations for candidate review and recruiter workflow.
type CandidateReviewService interface {
	GetJobCandidates(ctx context.Context, filter JobCandidateFilterInput) (*JobCandidatePageResponse, error)
	GetReviewDetail(ctx context.Context, jobID, candidateID string) (*ReviewDetailResponse, error)
	UpdateStatus(ctx context.Context, jobID, candidateID string, status model.JobCandidateStatus, reviewerID string) (*UpdateStatusResponse, error)
	CreateNote(ctx context.Context, jobID, candidateID string, authorID string, content string) (*CandidateNoteDTO, error)
	ListNotes(ctx context.Context, jobID, candidateID string, page, limit int) (*CandidateNotePageResponse, error)
	UpdateNote(ctx context.Context, jobID, candidateID, noteID string, userID string, userRole model.Role, content string) (*CandidateNoteDTO, error)
	DeleteNote(ctx context.Context, jobID, candidateID, noteID string, userID string, userRole model.Role) error
}

type candidateReviewService struct {
	jobCandidateRepo  repository.JobCandidateRepository
	candidateNoteRepo repository.CandidateNoteRepository
	jobRepo           repository.JobRepository
	candidateRepo     repository.CandidateRepository
	screeningRepo     repository.ScreeningRepository
	auditRepo         repository.AuditLogRepository
}

// NewCandidateReviewService creates an instance of CandidateReviewService.
func NewCandidateReviewService(
	jobCandidateRepo repository.JobCandidateRepository,
	candidateNoteRepo repository.CandidateNoteRepository,
	jobRepo repository.JobRepository,
	candidateRepo repository.CandidateRepository,
	screeningRepo repository.ScreeningRepository,
	auditRepo repository.AuditLogRepository,
) CandidateReviewService {
	return &candidateReviewService{
		jobCandidateRepo:  jobCandidateRepo,
		candidateNoteRepo: candidateNoteRepo,
		jobRepo:           jobRepo,
		candidateRepo:     candidateRepo,
		screeningRepo:     screeningRepo,
		auditRepo:         auditRepo,
	}
}

func (s *candidateReviewService) logAuditEvent(ctx context.Context, action model.AuditAction, actorID string, jobID, candidateID *string, metaMap map[string]interface{}) {
	if s.auditRepo == nil {
		return
	}
	var metaJSON string
	if metaMap != nil {
		if bytes, err := json.Marshal(metaMap); err == nil {
			metaJSON = string(bytes)
		}
	}
	_ = s.auditRepo.Create(ctx, &model.AuditLog{
		Action:      action,
		ActorID:     actorID,
		JobID:       jobID,
		CandidateID: candidateID,
		Metadata:    metaJSON,
		CreatedAt:   time.Now().UTC(),
	})
}

func (s *candidateReviewService) GetJobCandidates(ctx context.Context, filter JobCandidateFilterInput) (*JobCandidatePageResponse, error) {
	if _, err := uuid.Parse(filter.JobID); err != nil {
		return nil, ErrInvalidUUID
	}

	// Verify job exists
	if _, err := s.jobRepo.GetByID(ctx, filter.JobID, false); err != nil {
		return nil, err
	}

	repoFilter := repository.JobCandidateFilter{
		JobID:  filter.JobID,
		Status: filter.Status,
		Search: filter.Search,
		Page:   filter.Page,
		Limit:  filter.Limit,
		Sort:   filter.Sort,
		Order:  filter.Order,
	}

	result, err := s.jobCandidateRepo.ListJobCandidates(ctx, repoFilter)
	if err != nil {
		return nil, err
	}

	items := make([]JobCandidateItemDTO, 0, len(result.Items))
	for _, jc := range result.Items {
		item := JobCandidateItemDTO{
			Status:     jc.Status,
			ReviewedAt: jc.ReviewedAt,
		}

		if jc.Reviewer != nil {
			item.ReviewedBy = &UserSummaryDTO{
				ID:    jc.Reviewer.ID,
				Name:  jc.Reviewer.Name,
				Email: jc.Reviewer.Email,
				Role:  jc.Reviewer.Role,
			}
		}

		if jc.Candidate != nil {
			item.ID = jc.Candidate.ID
			item.FullName = jc.Candidate.FullName
			item.Email = jc.Candidate.Email
			item.Phone = jc.Candidate.Phone
			item.Location = jc.Candidate.Location
			item.Headline = jc.Candidate.Headline
			item.Candidate = CandidateSummaryDTO{
				ID:       jc.Candidate.ID,
				FullName: jc.Candidate.FullName,
				Email:    jc.Candidate.Email,
				Phone:    jc.Candidate.Phone,
				Location: jc.Candidate.Location,
				Headline: jc.Candidate.Headline,
			}
		}

		// Fetch Step 7 screening result summary if available
		scr, err := s.screeningRepo.GetByCandidateAndJob(ctx, jc.CandidateID, jc.JobID)
		if err == nil && scr != nil {
			item.Screening = &ScreeningSummaryDTO{
				Status: scr.Status,
				Required: ScreeningCountsDTO{
					Match:    scr.RequiredMatchCount,
					Partial:  scr.RequiredPartialCount,
					Mismatch: scr.RequiredMismatchCount,
					Unknown:  scr.RequiredUnknownCount,
				},
				Preferred: ScreeningCountsDTO{
					Match:    scr.PreferredMatchCount,
					Partial:  scr.PreferredPartialCount,
					Mismatch: scr.PreferredMismatchCount,
					Unknown:  scr.PreferredUnknownCount,
				},
				EvaluatedAt: scr.EvaluatedAt,
			}
		}

		items = append(items, item)
	}

	return &JobCandidatePageResponse{
		Items:      items,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *candidateReviewService) GetReviewDetail(ctx context.Context, jobID, candidateID string) (*ReviewDetailResponse, error) {
	if _, err := uuid.Parse(jobID); err != nil {
		return nil, ErrInvalidUUID
	}
	if _, err := uuid.Parse(candidateID); err != nil {
		return nil, ErrInvalidUUID
	}

	// 1. Verify Job exists
	job, err := s.jobRepo.GetByID(ctx, jobID, false)
	if err != nil {
		return nil, err
	}

	// 2. Verify Candidate exists with full background relations
	cand, err := s.candidateRepo.GetByID(ctx, candidateID, true)
	if err != nil {
		return nil, err
	}

	// 3. Verify Job-Candidate relationship
	jc, err := s.jobCandidateRepo.GetJobCandidate(ctx, jobID, candidateID)
	if err != nil {
		return nil, err
	}

	resp := &ReviewDetailResponse{}
	resp.Job.ID = job.ID
	resp.Job.Title = job.Title
	resp.Job.Status = job.Status
	resp.Job.Department = job.Department
	resp.Job.Location = job.Location
	resp.Job.EmploymentType = job.EmploymentType

	resp.Candidate.ID = cand.ID
	resp.Candidate.FullName = cand.FullName
	resp.Candidate.Email = cand.Email
	resp.Candidate.Phone = cand.Phone
	resp.Candidate.Location = cand.Location
	resp.Candidate.Headline = cand.Headline
	resp.Candidate.Summary = cand.Summary

	resp.Workflow.Status = jc.Status
	resp.Workflow.ReviewedAt = jc.ReviewedAt
	if jc.Reviewer != nil {
		resp.Workflow.ReviewedBy = &UserSummaryDTO{
			ID:    jc.Reviewer.ID,
			Name:  jc.Reviewer.Name,
			Email: jc.Reviewer.Email,
			Role:  jc.Reviewer.Role,
		}
	}

	// Safe documents metadata
	resp.Documents = make([]DocumentSummaryDTO, 0, len(cand.Documents))
	for _, doc := range cand.Documents {
		resp.Documents = append(resp.Documents, DocumentSummaryDTO{
			ID:         doc.ID,
			SourceType: doc.SourceType,
			FileName:   doc.OriginalFilename,
			MimeType:   doc.MimeType,
			FileSize:   doc.FileSize,
			CreatedAt:  doc.CreatedAt,
		})
	}

	resp.Education = cand.Educations
	resp.Experience = cand.Experiences
	resp.Skills = cand.Skills

	// Step 7 Screening data
	scr, err := s.screeningRepo.GetByCandidateAndJob(ctx, candidateID, jobID)
	if err == nil && scr != nil {
		resp.Screening = toScreeningResponse(scr)
	}

	// Candidate notes for this job
	notesResult, err := s.candidateNoteRepo.ListByJobAndCandidate(ctx, jobID, candidateID, 1, 100)
	if err == nil && notesResult != nil {
		resp.Notes = make([]CandidateNoteDTO, 0, len(notesResult.Items))
		for _, n := range notesResult.Items {
			dto := CandidateNoteDTO{
				ID:          n.ID,
				JobID:       n.JobID,
				CandidateID: n.CandidateID,
				Content:     n.Content,
				CreatedAt:   n.CreatedAt,
				UpdatedAt:   n.UpdatedAt,
			}
			if n.Author != nil {
				dto.Author = &UserSummaryDTO{
					ID:    n.Author.ID,
					Name:  n.Author.Name,
					Email: n.Author.Email,
					Role:  n.Author.Role,
				}
			}
			resp.Notes = append(resp.Notes, dto)
		}
	}

	return resp, nil
}

func (s *candidateReviewService) UpdateStatus(ctx context.Context, jobID, candidateID string, status model.JobCandidateStatus, reviewerID string) (*UpdateStatusResponse, error) {
	if _, err := uuid.Parse(jobID); err != nil {
		return nil, ErrInvalidUUID
	}
	if _, err := uuid.Parse(candidateID); err != nil {
		return nil, ErrInvalidUUID
	}

	cleanStatus := model.JobCandidateStatus(strings.ToUpper(strings.TrimSpace(string(status))))
	if !cleanStatus.IsValid() {
		return nil, ErrInvalidStatus
	}

	// Verify job and candidate
	if _, err := s.jobRepo.GetByID(ctx, jobID, false); err != nil {
		return nil, err
	}
	if _, err := s.candidateRepo.GetByID(ctx, candidateID, false); err != nil {
		return nil, err
	}

	currentJC, err := s.jobCandidateRepo.GetJobCandidate(ctx, jobID, candidateID)
	if err != nil {
		return nil, err
	}

	oldStatus := currentJC.Status
	now := time.Now().UTC()
	updatedJC, err := s.jobCandidateRepo.UpdateStatus(ctx, jobID, candidateID, cleanStatus, reviewerID, now)
	if err != nil {
		return nil, err
	}

	// Audit logging (safe metadata only, no PII)
	s.logAuditEvent(ctx, model.AuditActionCandidateStatusChanged, reviewerID, &jobID, &candidateID, map[string]interface{}{
		"job_id":       jobID,
		"candidate_id": candidateID,
		"old_status":   string(oldStatus),
		"new_status":   string(cleanStatus),
	})

	resp := &UpdateStatusResponse{
		JobID:       updatedJC.JobID,
		CandidateID: updatedJC.CandidateID,
		Status:      updatedJC.Status,
		ReviewedAt:  updatedJC.ReviewedAt,
	}
	if updatedJC.Reviewer != nil {
		resp.ReviewedBy = &UserSummaryDTO{
			ID:    updatedJC.Reviewer.ID,
			Name:  updatedJC.Reviewer.Name,
			Email: updatedJC.Reviewer.Email,
			Role:  updatedJC.Reviewer.Role,
		}
	}

	return resp, nil
}

func (s *candidateReviewService) CreateNote(ctx context.Context, jobID, candidateID string, authorID string, content string) (*CandidateNoteDTO, error) {
	if _, err := uuid.Parse(jobID); err != nil {
		return nil, ErrInvalidUUID
	}
	if _, err := uuid.Parse(candidateID); err != nil {
		return nil, ErrInvalidUUID
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("%w: note content cannot be empty", ErrValidation)
	}
	if len(content) > 5000 {
		return nil, fmt.Errorf("%w: note content cannot exceed 5000 characters", ErrValidation)
	}

	// Verify job and candidate relationship
	if _, err := s.jobRepo.GetByID(ctx, jobID, false); err != nil {
		return nil, err
	}
	if _, err := s.candidateRepo.GetByID(ctx, candidateID, false); err != nil {
		return nil, err
	}
	associated, err := s.jobCandidateRepo.IsJobCandidateAssociated(ctx, jobID, candidateID)
	if err != nil {
		return nil, err
	}
	if !associated {
		return nil, repository.ErrJobCandidateNotFound
	}

	note := &model.CandidateNote{
		JobID:       jobID,
		CandidateID: candidateID,
		AuthorID:    authorID,
		Content:     content,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := s.candidateNoteRepo.Create(ctx, note); err != nil {
		return nil, err
	}

	s.logAuditEvent(ctx, model.AuditActionCandidateNoteCreated, authorID, &jobID, &candidateID, map[string]interface{}{
		"job_id":       jobID,
		"candidate_id": candidateID,
		"note_id":      note.ID,
	})

	savedNote, err := s.candidateNoteRepo.GetByID(ctx, note.ID)
	if err != nil {
		savedNote = note
	}

	dto := &CandidateNoteDTO{
		ID:          savedNote.ID,
		JobID:       savedNote.JobID,
		CandidateID: savedNote.CandidateID,
		Content:     savedNote.Content,
		CreatedAt:   savedNote.CreatedAt,
		UpdatedAt:   savedNote.UpdatedAt,
	}
	if savedNote.Author != nil {
		dto.Author = &UserSummaryDTO{
			ID:    savedNote.Author.ID,
			Name:  savedNote.Author.Name,
			Email: savedNote.Author.Email,
			Role:  savedNote.Author.Role,
		}
	}
	return dto, nil
}

func (s *candidateReviewService) ListNotes(ctx context.Context, jobID, candidateID string, page, limit int) (*CandidateNotePageResponse, error) {
	if _, err := uuid.Parse(jobID); err != nil {
		return nil, ErrInvalidUUID
	}
	if _, err := uuid.Parse(candidateID); err != nil {
		return nil, ErrInvalidUUID
	}

	associated, err := s.jobCandidateRepo.IsJobCandidateAssociated(ctx, jobID, candidateID)
	if err != nil {
		return nil, err
	}
	if !associated {
		return nil, repository.ErrJobCandidateNotFound
	}

	result, err := s.candidateNoteRepo.ListByJobAndCandidate(ctx, jobID, candidateID, page, limit)
	if err != nil {
		return nil, err
	}

	items := make([]CandidateNoteDTO, 0, len(result.Items))
	for _, n := range result.Items {
		dto := CandidateNoteDTO{
			ID:          n.ID,
			JobID:       n.JobID,
			CandidateID: n.CandidateID,
			Content:     n.Content,
			CreatedAt:   n.CreatedAt,
			UpdatedAt:   n.UpdatedAt,
		}
		if n.Author != nil {
			dto.Author = &UserSummaryDTO{
				ID:    n.Author.ID,
				Name:  n.Author.Name,
				Email: n.Author.Email,
				Role:  n.Author.Role,
			}
		}
		items = append(items, dto)
	}

	return &CandidateNotePageResponse{
		Items:      items,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *candidateReviewService) UpdateNote(ctx context.Context, jobID, candidateID, noteID string, userID string, userRole model.Role, content string) (*CandidateNoteDTO, error) {
	if _, err := uuid.Parse(jobID); err != nil {
		return nil, ErrInvalidUUID
	}
	if _, err := uuid.Parse(candidateID); err != nil {
		return nil, ErrInvalidUUID
	}
	if _, err := uuid.Parse(noteID); err != nil {
		return nil, ErrInvalidUUID
	}

	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("%w: note content cannot be empty", ErrValidation)
	}
	if len(content) > 5000 {
		return nil, fmt.Errorf("%w: note content cannot exceed 5000 characters", ErrValidation)
	}

	note, err := s.candidateNoteRepo.GetByID(ctx, noteID)
	if err != nil {
		return nil, err
	}

	// Verify contextual relationship: Note must belong to this job and candidate
	if note.JobID != jobID || note.CandidateID != candidateID {
		return nil, repository.ErrNoteNotFound
	}

	// Permission guard: Only note author or ADMIN can update
	if userRole != model.RoleAdmin && note.AuthorID != userID {
		return nil, ErrNoteForbidden
	}

	note.Content = content
	if err := s.candidateNoteRepo.Update(ctx, note); err != nil {
		return nil, err
	}

	s.logAuditEvent(ctx, model.AuditActionCandidateNoteUpdated, userID, &jobID, &candidateID, map[string]interface{}{
		"job_id":       jobID,
		"candidate_id": candidateID,
		"note_id":      note.ID,
	})

	updatedNote, err := s.candidateNoteRepo.GetByID(ctx, noteID)
	if err != nil {
		updatedNote = note
	}

	dto := &CandidateNoteDTO{
		ID:          updatedNote.ID,
		JobID:       updatedNote.JobID,
		CandidateID: updatedNote.CandidateID,
		Content:     updatedNote.Content,
		CreatedAt:   updatedNote.CreatedAt,
		UpdatedAt:   updatedNote.UpdatedAt,
	}
	if updatedNote.Author != nil {
		dto.Author = &UserSummaryDTO{
			ID:    updatedNote.Author.ID,
			Name:  updatedNote.Author.Name,
			Email: updatedNote.Author.Email,
			Role:  updatedNote.Author.Role,
		}
	}
	return dto, nil
}

func (s *candidateReviewService) DeleteNote(ctx context.Context, jobID, candidateID, noteID string, userID string, userRole model.Role) error {
	if _, err := uuid.Parse(jobID); err != nil {
		return ErrInvalidUUID
	}
	if _, err := uuid.Parse(candidateID); err != nil {
		return ErrInvalidUUID
	}
	if _, err := uuid.Parse(noteID); err != nil {
		return ErrInvalidUUID
	}

	note, err := s.candidateNoteRepo.GetByID(ctx, noteID)
	if err != nil {
		return err
	}

	// Verify contextual relationship
	if note.JobID != jobID || note.CandidateID != candidateID {
		return repository.ErrNoteNotFound
	}

	// Permission guard: Only note author or ADMIN can delete
	if userRole != model.RoleAdmin && note.AuthorID != userID {
		return ErrNoteForbidden
	}

	if err := s.candidateNoteRepo.Delete(ctx, noteID); err != nil {
		return err
	}

	s.logAuditEvent(ctx, model.AuditActionCandidateNoteDeleted, userID, &jobID, &candidateID, map[string]interface{}{
		"job_id":       jobID,
		"candidate_id": candidateID,
		"note_id":      noteID,
	})

	return nil
}
