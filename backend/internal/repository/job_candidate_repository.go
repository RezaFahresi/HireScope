package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"hirescope/backend/internal/model"

	"gorm.io/gorm"
)

// JobCandidateFilter specifies criteria for querying job candidates.
type JobCandidateFilter struct {
	JobID  string
	Status string
	Search string
	Page   int
	Limit  int
	Sort   string
	Order  string
}

// JobCandidateListResult encapsulates paginated job-candidate associations with metadata.
type JobCandidateListResult struct {
	Items      []model.JobCandidate `json:"items"`
	Total      int64                `json:"total"`
	Page       int                  `json:"page"`
	Limit      int                  `json:"limit"`
	TotalPages int                  `json:"total_pages"`
}

// JobCandidateRepository defines database access for job-candidate associations and workflow state.
type JobCandidateRepository interface {
	GetJobCandidate(ctx context.Context, jobID, candidateID string) (*model.JobCandidate, error)
	ListJobCandidates(ctx context.Context, filter JobCandidateFilter) (*JobCandidateListResult, error)
	UpdateStatus(ctx context.Context, jobID, candidateID string, status model.JobCandidateStatus, reviewedBy string, reviewedAt time.Time) (*model.JobCandidate, error)
	AddJobCandidate(ctx context.Context, jobID, candidateID string) error
	RemoveJobCandidate(ctx context.Context, jobID, candidateID string) error
	IsJobCandidateAssociated(ctx context.Context, jobID, candidateID string) (bool, error)
}

type gormJobCandidateRepository struct {
	db *gorm.DB
}

// NewJobCandidateRepository creates a new GORM-backed JobCandidateRepository.
func NewJobCandidateRepository(db *gorm.DB) JobCandidateRepository {
	return &gormJobCandidateRepository{db: db}
}

func (r *gormJobCandidateRepository) GetJobCandidate(ctx context.Context, jobID, candidateID string) (*model.JobCandidate, error) {
	var jc model.JobCandidate
	err := r.db.WithContext(ctx).
		Preload("Candidate").
		Preload("Candidate.Skills").
		Preload("Reviewer").
		Where("job_id = ? AND candidate_id = ?", jobID, candidateID).
		First(&jc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJobCandidateNotFound
		}
		return nil, err
	}
	return &jc, nil
}

func (r *gormJobCandidateRepository) ListJobCandidates(ctx context.Context, filter JobCandidateFilter) (*JobCandidateListResult, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	} else if filter.Limit > 100 {
		filter.Limit = 100
	}

	query := r.db.WithContext(ctx).Model(&model.JobCandidate{}).Where("job_id = ?", filter.JobID)

	// Workflow status filter
	if strings.TrimSpace(filter.Status) != "" {
		st := strings.ToUpper(strings.TrimSpace(filter.Status))
		query = query.Where("status = ?", st)
	}

	// Search filter across candidate fields using subquery
	if strings.TrimSpace(filter.Search) != "" {
		s := "%" + strings.ToLower(strings.TrimSpace(filter.Search)) + "%"
		query = query.Where("candidate_id IN (SELECT id FROM candidates WHERE LOWER(full_name) LIKE ? OR LOWER(email) LIKE ? OR LOWER(headline) LIKE ? OR LOWER(location) LIKE ?)", s, s, s, s)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Sorting
	orderClause := "created_at DESC"
	sortField := strings.ToLower(strings.TrimSpace(filter.Sort))
	orderDirection := strings.ToUpper(strings.TrimSpace(filter.Order))
	if orderDirection != "ASC" && orderDirection != "DESC" {
		orderDirection = "DESC"
	}

	switch sortField {
	case "created_at", "updated_at", "reviewed_at", "status":
		orderClause = fmt.Sprintf("%s %s", sortField, orderDirection)
	}

	offset := (filter.Page - 1) * filter.Limit
	var items []model.JobCandidate
	err := query.
		Order(orderClause).
		Offset(offset).
		Limit(filter.Limit).
		Preload("Candidate").
		Preload("Candidate.Skills").
		Preload("Reviewer").
		Find(&items).Error
	if err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))
	if totalPages == 0 {
		totalPages = 1
	}

	return &JobCandidateListResult{
		Items:      items,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *gormJobCandidateRepository) UpdateStatus(ctx context.Context, jobID, candidateID string, status model.JobCandidateStatus, reviewedBy string, reviewedAt time.Time) (*model.JobCandidate, error) {
	var jc model.JobCandidate
	err := r.db.WithContext(ctx).
		Where("job_id = ? AND candidate_id = ?", jobID, candidateID).
		First(&jc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJobCandidateNotFound
		}
		return nil, err
	}

	jc.Status = status
	jc.ReviewedBy = &reviewedBy
	jc.ReviewedAt = &reviewedAt
	jc.UpdatedAt = time.Now().UTC()

	if err := r.db.WithContext(ctx).Save(&jc).Error; err != nil {
		return nil, err
	}

	// Reload with reviewer and candidate preloaded
	return r.GetJobCandidate(ctx, jobID, candidateID)
}

func (r *gormJobCandidateRepository) AddJobCandidate(ctx context.Context, jobID, candidateID string) error {
	assoc := &model.JobCandidate{
		JobID:       jobID,
		CandidateID: candidateID,
		Status:      model.JobCandidateStatusReview,
	}
	err := r.db.WithContext(ctx).Create(assoc).Error
	if err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "idx_job_candidate") {
			return ErrJobCandidateAlreadyExists
		}
		return err
	}
	return nil
}

func (r *gormJobCandidateRepository) RemoveJobCandidate(ctx context.Context, jobID, candidateID string) error {
	res := r.db.WithContext(ctx).
		Where("job_id = ? AND candidate_id = ?", jobID, candidateID).
		Delete(&model.JobCandidate{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrJobCandidateNotFound
	}
	return nil
}

func (r *gormJobCandidateRepository) IsJobCandidateAssociated(ctx context.Context, jobID, candidateID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.JobCandidate{}).
		Where("job_id = ? AND candidate_id = ?", jobID, candidateID).
		Count(&count).Error
	return count > 0, err
}
