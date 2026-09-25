package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"hirescope/backend/internal/model"
	"hirescope/backend/internal/repository"
)

var (
	ErrInvalidStatusTransition = errors.New("invalid status transition")
	ErrTitleRequired           = errors.New("job title is required")
	ErrDescriptionRequired     = errors.New("job description is required")
	ErrInvalidEmploymentType   = errors.New("invalid employment type")
	ErrInvalidJobStatus        = errors.New("invalid job status")
	ErrInvalidCategory         = errors.New("invalid requirement category")
	ErrRequirementTextRequired = errors.New("requirement text is required")
	ErrInvalidImportance       = errors.New("invalid requirement importance")
	ErrCrossJobRequirement     = errors.New("requirement does not belong to specified job")
)

// CreateJobInput encapsulates parameters for creating a new recruitment vacancy.
type CreateJobInput struct {
	Title          string               `json:"title"`
	Description    string               `json:"description"`
	Department     string               `json:"department"`
	Location       string               `json:"location"`
	EmploymentType model.EmploymentType `json:"employment_type"`
	Status         model.JobStatus      `json:"status"`
	ClosingDate    *string              `json:"closing_date"`
}

// UpdateJobInput encapsulates parameters for updating an existing vacancy.
type UpdateJobInput struct {
	Title          *string               `json:"title"`
	Description    *string               `json:"description"`
	Department     *string               `json:"department"`
	Location       *string               `json:"location"`
	EmploymentType *model.EmploymentType `json:"employment_type"`
	Status         *model.JobStatus      `json:"status"`
	ClosingDate    *string               `json:"closing_date"`
}

// CreateRequirementInput encapsulates parameters for adding a requirement.
type CreateRequirementInput struct {
	Category    model.RequirementCategory   `json:"category"`
	Requirement string                      `json:"requirement"`
	Importance  model.RequirementImportance `json:"importance"`
}

// UpdateRequirementInput encapsulates parameters for updating a requirement.
type UpdateRequirementInput struct {
	Category    *model.RequirementCategory   `json:"category"`
	Requirement *string                      `json:"requirement"`
	Importance  *model.RequirementImportance `json:"importance"`
}

// JobService defines business operations for job vacancies and requirements.
type JobService interface {
	CreateJob(ctx context.Context, creatorID string, input CreateJobInput) (*model.Job, error)
	GetJobByID(ctx context.Context, id string) (*model.Job, error)
	ListJobs(ctx context.Context, filter repository.JobFilter) (*repository.JobListResult, error)
	UpdateJob(ctx context.Context, id string, input UpdateJobInput) (*model.Job, error)
	ArchiveJob(ctx context.Context, id string) (*model.Job, error)

	CreateRequirement(ctx context.Context, jobID string, input CreateRequirementInput) (*model.JobRequirement, error)
	ListRequirements(ctx context.Context, jobID string) ([]model.JobRequirement, error)
	UpdateRequirement(ctx context.Context, jobID, reqID string, input UpdateRequirementInput) (*model.JobRequirement, error)
	DeleteRequirement(ctx context.Context, jobID, reqID string) error
}

type jobService struct {
	jobRepo repository.JobRepository
}

// NewJobService creates a new JobService instance.
func NewJobService(jobRepo repository.JobRepository) JobService {
	return &jobService{jobRepo: jobRepo}
}

func (s *jobService) CreateJob(ctx context.Context, creatorID string, input CreateJobInput) (*model.Job, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, ErrTitleRequired
	}
	if len(title) > 255 {
		return nil, errors.New("job title exceeds maximum length of 255 characters")
	}

	description := strings.TrimSpace(input.Description)
	if description == "" {
		return nil, ErrDescriptionRequired
	}

	if !input.EmploymentType.IsValid() {
		return nil, ErrInvalidEmploymentType
	}

	status := input.Status
	if status == "" {
		status = model.JobStatusDraft
	}
	if !status.IsValid() {
		return nil, ErrInvalidJobStatus
	}

	var closingDate *time.Time
	if input.ClosingDate != nil && strings.TrimSpace(*input.ClosingDate) != "" {
		t, err := time.Parse("2006-01-02", strings.TrimSpace(*input.ClosingDate))
		if err != nil {
			return nil, fmt.Errorf("invalid closing_date format, expected YYYY-MM-DD: %w", err)
		}
		closingDate = &t
	}

	code, err := s.jobRepo.NextJobCode(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate job code: %w", err)
	}

	job := &model.Job{
		Code:           code,
		Title:          title,
		Description:    description,
		Department:     strings.TrimSpace(input.Department),
		Location:       strings.TrimSpace(input.Location),
		EmploymentType: input.EmploymentType,
		Status:         status,
		ClosingDate:    closingDate,
		CreatedBy:      creatorID,
	}

	if err := s.jobRepo.Create(ctx, job); err != nil {
		return nil, err
	}

	return s.jobRepo.GetByID(ctx, job.ID, true)
}

func (s *jobService) GetJobByID(ctx context.Context, id string) (*model.Job, error) {
	if strings.TrimSpace(id) == "" {
		return nil, repository.ErrJobNotFound
	}
	return s.jobRepo.GetByID(ctx, id, true)
}

func (s *jobService) ListJobs(ctx context.Context, filter repository.JobFilter) (*repository.JobListResult, error) {
	return s.jobRepo.List(ctx, filter)
}

func (s *jobService) UpdateJob(ctx context.Context, id string, input UpdateJobInput) (*model.Job, error) {
	job, err := s.jobRepo.GetByID(ctx, id, false)
	if err != nil {
		return nil, err
	}

	// Validate title if supplied
	if input.Title != nil {
		t := strings.TrimSpace(*input.Title)
		if t == "" {
			return nil, ErrTitleRequired
		}
		if len(t) > 255 {
			return nil, errors.New("job title exceeds maximum length of 255 characters")
		}
		job.Title = t
	}

	// Validate description if supplied
	if input.Description != nil {
		d := strings.TrimSpace(*input.Description)
		if d == "" {
			return nil, ErrDescriptionRequired
		}
		job.Description = d
	}

	if input.Department != nil {
		job.Department = strings.TrimSpace(*input.Department)
	}

	if input.Location != nil {
		job.Location = strings.TrimSpace(*input.Location)
	}

	if input.EmploymentType != nil {
		if !input.EmploymentType.IsValid() {
			return nil, ErrInvalidEmploymentType
		}
		job.EmploymentType = *input.EmploymentType
	}

	// Validate status transition if requested
	if input.Status != nil {
		targetStatus := *input.Status
		if !targetStatus.IsValid() {
			return nil, ErrInvalidJobStatus
		}
		if !job.Status.CanTransitionTo(targetStatus) {
			return nil, fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidStatusTransition, job.Status, targetStatus)
		}
		job.Status = targetStatus
	}

	if input.ClosingDate != nil {
		rawDate := strings.TrimSpace(*input.ClosingDate)
		if rawDate == "" {
			job.ClosingDate = nil
		} else {
			t, err := time.Parse("2006-01-02", rawDate)
			if err != nil {
				return nil, fmt.Errorf("invalid closing_date format, expected YYYY-MM-DD: %w", err)
			}
			job.ClosingDate = &t
		}
	}

	if err := s.jobRepo.Update(ctx, job); err != nil {
		return nil, err
	}

	return s.jobRepo.GetByID(ctx, job.ID, true)
}

func (s *jobService) ArchiveJob(ctx context.Context, id string) (*model.Job, error) {
	job, err := s.jobRepo.GetByID(ctx, id, false)
	if err != nil {
		return nil, err
	}

	if job.Status == model.JobStatusArchived {
		// Idempotent: already archived
		return s.jobRepo.GetByID(ctx, id, true)
	}

	if !job.Status.CanTransitionTo(model.JobStatusArchived) {
		return nil, fmt.Errorf("%w: cannot transition from %s to %s", ErrInvalidStatusTransition, job.Status, model.JobStatusArchived)
	}

	job.Status = model.JobStatusArchived
	if err := s.jobRepo.Update(ctx, job); err != nil {
		return nil, err
	}

	return s.jobRepo.GetByID(ctx, job.ID, true)
}

func (s *jobService) CreateRequirement(ctx context.Context, jobID string, input CreateRequirementInput) (*model.JobRequirement, error) {
	// Verify job exists
	if _, err := s.jobRepo.GetByID(ctx, jobID, false); err != nil {
		return nil, err
	}

	reqText := strings.TrimSpace(input.Requirement)
	if reqText == "" {
		return nil, ErrRequirementTextRequired
	}
	if len(reqText) > 500 {
		return nil, errors.New("requirement text exceeds maximum length of 500 characters")
	}

	if !input.Category.IsValid() {
		return nil, ErrInvalidCategory
	}

	importance := input.Importance
	if importance == "" {
		importance = model.ImportanceRequired
	}
	if !importance.IsValid() {
		return nil, ErrInvalidImportance
	}

	req := &model.JobRequirement{
		JobID:       jobID,
		Category:    input.Category,
		Requirement: reqText,
		Importance:  importance,
	}

	if err := s.jobRepo.CreateRequirement(ctx, req); err != nil {
		return nil, err
	}

	return req, nil
}

func (s *jobService) ListRequirements(ctx context.Context, jobID string) ([]model.JobRequirement, error) {
	if _, err := s.jobRepo.GetByID(ctx, jobID, false); err != nil {
		return nil, err
	}
	return s.jobRepo.ListRequirements(ctx, jobID)
}

func (s *jobService) UpdateRequirement(ctx context.Context, jobID, reqID string, input UpdateRequirementInput) (*model.JobRequirement, error) {
	// Verify job exists
	if _, err := s.jobRepo.GetByID(ctx, jobID, false); err != nil {
		return nil, err
	}

	req, err := s.jobRepo.GetRequirementByID(ctx, reqID)
	if err != nil {
		return nil, err
	}

	// Verify requirement belongs to the specified job
	if req.JobID != jobID {
		return nil, ErrCrossJobRequirement
	}

	if input.Category != nil {
		if !input.Category.IsValid() {
			return nil, ErrInvalidCategory
		}
		req.Category = *input.Category
	}

	if input.Requirement != nil {
		t := strings.TrimSpace(*input.Requirement)
		if t == "" {
			return nil, ErrRequirementTextRequired
		}
		if len(t) > 500 {
			return nil, errors.New("requirement text exceeds maximum length of 500 characters")
		}
		req.Requirement = t
	}

	if input.Importance != nil {
		if !input.Importance.IsValid() {
			return nil, ErrInvalidImportance
		}
		req.Importance = *input.Importance
	}

	if err := s.jobRepo.UpdateRequirement(ctx, req); err != nil {
		return nil, err
	}

	return req, nil
}

func (s *jobService) DeleteRequirement(ctx context.Context, jobID, reqID string) error {
	// Verify job exists
	if _, err := s.jobRepo.GetByID(ctx, jobID, false); err != nil {
		return err
	}

	// Verify requirement exists and belongs to the specified job
	req, err := s.jobRepo.GetRequirementByID(ctx, reqID)
	if err != nil {
		return err
	}
	if req.JobID != jobID {
		return ErrCrossJobRequirement
	}

	return s.jobRepo.DeleteRequirement(ctx, jobID, reqID)
}
