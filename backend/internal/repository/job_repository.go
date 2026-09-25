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

var (
	ErrJobNotFound         = errors.New("job not found")
	ErrRequirementNotFound = errors.New("job requirement not found")
)

// JobFilter specifies criteria for querying vacancies.
type JobFilter struct {
	Search         string
	Status         string
	Department     string
	EmploymentType string
	Page           int
	Limit          int
	Sort           string
	Order          string
}

// JobListResult contains paginated job results and pagination metadata.
type JobListResult struct {
	Items      []model.Job `json:"items"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	TotalPages int         `json:"total_pages"`
}

// JobRepository defines database operations for Jobs and JobRequirements.
type JobRepository interface {
	Create(ctx context.Context, job *model.Job) error
	GetByID(ctx context.Context, id string, includeRelations bool) (*model.Job, error)
	List(ctx context.Context, filter JobFilter) (*JobListResult, error)
	Update(ctx context.Context, job *model.Job) error
	NextJobCode(ctx context.Context) (string, error)

	CreateRequirement(ctx context.Context, req *model.JobRequirement) error
	GetRequirementByID(ctx context.Context, reqID string) (*model.JobRequirement, error)
	ListRequirements(ctx context.Context, jobID string) ([]model.JobRequirement, error)
	UpdateRequirement(ctx context.Context, req *model.JobRequirement) error
	DeleteRequirement(ctx context.Context, jobID, reqID string) error
}

type gormJobRepository struct {
	db *gorm.DB
}

// NewJobRepository creates a new JobRepository backed by GORM.
func NewJobRepository(db *gorm.DB) JobRepository {
	return &gormJobRepository{db: db}
}

func (r *gormJobRepository) Create(ctx context.Context, job *model.Job) error {
	return r.db.WithContext(ctx).Create(job).Error
}

func (r *gormJobRepository) GetByID(ctx context.Context, id string, includeRelations bool) (*model.Job, error) {
	var job model.Job
	query := r.db.WithContext(ctx).Where("id = ?", id)
	if includeRelations {
		query = query.Preload("Creator").Preload("Requirements")
	}

	err := query.First(&job).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}
	return &job, nil
}

func (r *gormJobRepository) List(ctx context.Context, filter JobFilter) (*JobListResult, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 10
	} else if filter.Limit > 100 {
		filter.Limit = 100
	}

	query := r.db.WithContext(ctx).Model(&model.Job{})

	// Search filter
	if strings.TrimSpace(filter.Search) != "" {
		s := "%" + strings.ToLower(strings.TrimSpace(filter.Search)) + "%"
		query = query.Where("LOWER(title) LIKE ? OR LOWER(description) LIKE ? OR LOWER(department) LIKE ? OR LOWER(code) LIKE ?", s, s, s, s)
	}

	// Status filter
	if strings.TrimSpace(filter.Status) != "" {
		query = query.Where("status = ?", strings.ToUpper(strings.TrimSpace(filter.Status)))
	}

	// Department filter
	if strings.TrimSpace(filter.Department) != "" {
		query = query.Where("LOWER(department) = ?", strings.ToLower(strings.TrimSpace(filter.Department)))
	}

	// Employment type filter
	if strings.TrimSpace(filter.EmploymentType) != "" {
		query = query.Where("employment_type = ?", strings.ToUpper(strings.TrimSpace(filter.EmploymentType)))
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// Sorting
	allowedSorts := map[string]string{
		"created_at":   "created_at",
		"title":        "title",
		"status":       "status",
		"department":   "department",
		"closing_date": "closing_date",
	}
	sortField, ok := allowedSorts[strings.ToLower(filter.Sort)]
	if !ok {
		sortField = "created_at"
	}

	orderDir := "DESC"
	if strings.EqualFold(filter.Order, "asc") {
		orderDir = "ASC"
	}
	orderClause := fmt.Sprintf("%s %s", sortField, orderDir)

	offset := (filter.Page - 1) * filter.Limit
	var items []model.Job
	err := query.Order(orderClause).
		Offset(offset).
		Limit(filter.Limit).
		Preload("Creator").
		Preload("Requirements").
		Find(&items).Error
	if err != nil {
		return nil, err
	}

	totalPages := int((total + int64(filter.Limit) - 1) / int64(filter.Limit))
	if totalPages == 0 {
		totalPages = 1
	}

	return &JobListResult{
		Items:      items,
		Total:      total,
		Page:       filter.Page,
		Limit:      filter.Limit,
		TotalPages: totalPages,
	}, nil
}

func (r *gormJobRepository) Update(ctx context.Context, job *model.Job) error {
	return r.db.WithContext(ctx).Save(job).Error
}

func (r *gormJobRepository) NextJobCode(ctx context.Context) (string, error) {
	year := time.Now().UTC().Year()
	prefix := fmt.Sprintf("JOB-%d-", year)

	var count int64
	err := r.db.WithContext(ctx).Model(&model.Job{}).
		Where("code LIKE ?", prefix+"%").
		Count(&count).Error
	if err != nil {
		return "", err
	}

	// Generate sequential code: JOB-YYYY-0001
	seq := count + 1
	candidateCode := fmt.Sprintf("JOB-%d-%04d", year, seq)

	// Ensure uniqueness in case of race condition or gaps
	for {
		var existingCount int64
		err := r.db.WithContext(ctx).Model(&model.Job{}).
			Where("code = ?", candidateCode).
			Count(&existingCount).Error
		if err != nil {
			return "", err
		}
		if existingCount == 0 {
			break
		}
		seq++
		candidateCode = fmt.Sprintf("JOB-%d-%04d", year, seq)
	}

	return candidateCode, nil
}

func (r *gormJobRepository) CreateRequirement(ctx context.Context, req *model.JobRequirement) error {
	return r.db.WithContext(ctx).Create(req).Error
}

func (r *gormJobRepository) GetRequirementByID(ctx context.Context, reqID string) (*model.JobRequirement, error) {
	var req model.JobRequirement
	err := r.db.WithContext(ctx).Where("id = ?", reqID).First(&req).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRequirementNotFound
		}
		return nil, err
	}
	return &req, nil
}

func (r *gormJobRepository) ListRequirements(ctx context.Context, jobID string) ([]model.JobRequirement, error) {
	var requirements []model.JobRequirement
	err := r.db.WithContext(ctx).
		Where("job_id = ?", jobID).
		Order("created_at ASC").
		Find(&requirements).Error
	return requirements, err
}

func (r *gormJobRepository) UpdateRequirement(ctx context.Context, req *model.JobRequirement) error {
	return r.db.WithContext(ctx).Save(req).Error
}

func (r *gormJobRepository) DeleteRequirement(ctx context.Context, jobID, reqID string) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND job_id = ?", reqID, jobID).
		Delete(&model.JobRequirement{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrRequirementNotFound
	}
	return nil
}
