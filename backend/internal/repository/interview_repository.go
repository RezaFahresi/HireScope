package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"hirescope/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrInterviewNotFound = errors.New("interview not found")
)

// InterviewFilter specifies query criteria for listing interviews.
type InterviewFilter struct {
	JobID         string
	CandidateID   string
	Status        string
	Stage         string
	InterviewType string
	From          *time.Time
	To            *time.Time
	Page          int
	Limit         int
	Sort          string
	Order         string
}

// InterviewListResult wraps paginated interview query results.
type InterviewListResult struct {
	Items      []model.Interview `json:"items"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	Limit      int               `json:"limit"`
	TotalPages int               `json:"total_pages"`
}

// InterviewRepository defines database access for interview entities.
type InterviewRepository interface {
	Create(ctx context.Context, interview *model.Interview, interviewerIDs []string, audit *model.AuditLog) error
	GetByID(ctx context.Context, id string) (*model.Interview, error)
	Update(ctx context.Context, interview *model.Interview, interviewerIDs []string, audit *model.AuditLog) error
	UpdateStatus(ctx context.Context, interview *model.Interview, audit *model.AuditLog) error
	List(ctx context.Context, filter InterviewFilter) (*InterviewListResult, error)
	CheckInterviewerConflict(ctx context.Context, interviewerIDs []string, start, end time.Time, excludeInterviewID string) (string, error)
	GetCandidateNextInterview(ctx context.Context, candidateID, jobID string) (*model.Interview, error)
	ListByCandidateAndJob(ctx context.Context, candidateID, jobID string) ([]model.Interview, error)
}

type gormInterviewRepository struct {
	db *gorm.DB
}

// NewInterviewRepository constructs a GORM-backed InterviewRepository.
func NewInterviewRepository(db *gorm.DB) InterviewRepository {
	return &gormInterviewRepository{db: db}
}

func (r *gormInterviewRepository) Create(ctx context.Context, interview *model.Interview, interviewerIDs []string, audit *model.AuditLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(interview).Error; err != nil {
			return fmt.Errorf("failed to create interview: %w", err)
		}

		for _, uid := range interviewerIDs {
			interviewer := model.InterviewInterviewer{
				ID:          uuid.New().String(),
				InterviewID: interview.ID,
				UserID:      uid,
				CreatedAt:   time.Now().UTC(),
			}
			if err := tx.Create(&interviewer).Error; err != nil {
				return fmt.Errorf("failed to add interviewer %s: %w", uid, err)
			}
		}

		if audit != nil {
			if audit.ID == "" {
				audit.ID = uuid.New().String()
			}
			if err := tx.Create(audit).Error; err != nil {
				return fmt.Errorf("failed to create audit log: %w", err)
			}
		}

		return nil
	})
}

func (r *gormInterviewRepository) GetByID(ctx context.Context, id string) (*model.Interview, error) {
	var item model.Interview
	err := r.db.WithContext(ctx).
		Preload("Job").
		Preload("Candidate").
		Preload("Creator").
		Preload("CompletedUser").
		Preload("CancelledUser").
		Preload("Interviewers.User").
		Where("id = ?", id).
		First(&item).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInterviewNotFound
		}
		return nil, err
	}
	return &item, nil
}

func (r *gormInterviewRepository) Update(ctx context.Context, interview *model.Interview, interviewerIDs []string, audit *model.AuditLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(interview).Error; err != nil {
			return fmt.Errorf("failed to update interview: %w", err)
		}

		if interviewerIDs != nil {
			// Remove existing interviewers and re-insert
			if err := tx.Where("interview_id = ?", interview.ID).Delete(&model.InterviewInterviewer{}).Error; err != nil {
				return fmt.Errorf("failed to clear previous interviewers: %w", err)
			}

			for _, uid := range interviewerIDs {
				interviewer := model.InterviewInterviewer{
					ID:          uuid.New().String(),
					InterviewID: interview.ID,
					UserID:      uid,
					CreatedAt:   time.Now().UTC(),
				}
				if err := tx.Create(&interviewer).Error; err != nil {
					return fmt.Errorf("failed to add interviewer %s: %w", uid, err)
				}
			}
		}

		if audit != nil {
			if audit.ID == "" {
				audit.ID = uuid.New().String()
			}
			if err := tx.Create(audit).Error; err != nil {
				return fmt.Errorf("failed to create audit log: %w", err)
			}
		}

		return nil
	})
}

func (r *gormInterviewRepository) UpdateStatus(ctx context.Context, interview *model.Interview, audit *model.AuditLog) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(interview).Error; err != nil {
			return fmt.Errorf("failed to save interview status: %w", err)
		}

		if audit != nil {
			if audit.ID == "" {
				audit.ID = uuid.New().String()
			}
			if err := tx.Create(audit).Error; err != nil {
				return fmt.Errorf("failed to create audit log: %w", err)
			}
		}

		return nil
	})
}

func (r *gormInterviewRepository) List(ctx context.Context, filter InterviewFilter) (*InterviewListResult, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	limit := filter.Limit
	if limit < 1 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}

	query := r.db.WithContext(ctx).Model(&model.Interview{})

	if filter.JobID != "" {
		query = query.Where("job_id = ?", filter.JobID)
	}
	if filter.CandidateID != "" {
		query = query.Where("candidate_id = ?", filter.CandidateID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Stage != "" {
		query = query.Where("stage = ?", filter.Stage)
	}
	if filter.InterviewType != "" {
		query = query.Where("interview_type = ?", filter.InterviewType)
	}
	if filter.From != nil {
		query = query.Where("scheduled_start >= ?", *filter.From)
	}
	if filter.To != nil {
		query = query.Where("scheduled_start <= ?", *filter.To)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count interviews: %w", err)
	}

	// Sorting
	sortField := "scheduled_start"
	switch strings.ToLower(filter.Sort) {
	case "title":
		sortField = "title"
	case "created_at":
		sortField = "created_at"
	case "status":
		sortField = "status"
	case "scheduled_start":
		sortField = "scheduled_start"
	}

	orderDir := "ASC"
	if strings.ToUpper(filter.Order) == "DESC" {
		orderDir = "DESC"
	}

	offset := (page - 1) * limit
	var items []model.Interview
	err := query.
		Preload("Job").
		Preload("Candidate").
		Preload("Creator").
		Preload("CompletedUser").
		Preload("CancelledUser").
		Preload("Interviewers.User").
		Order(fmt.Sprintf("%s %s", sortField, orderDir)).
		Offset(offset).
		Limit(limit).
		Find(&items).Error

	if err != nil {
		return nil, fmt.Errorf("failed to fetch interviews: %w", err)
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))
	if totalPages == 0 {
		totalPages = 1
	}

	return &InterviewListResult{
		Items:      items,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (r *gormInterviewRepository) CheckInterviewerConflict(ctx context.Context, interviewerIDs []string, start, end time.Time, excludeInterviewID string) (string, error) {
	if len(interviewerIDs) == 0 {
		return "", nil
	}

	type conflictResult struct {
		UserName string `gorm:"column:user_name"`
	}

	var result conflictResult
	query := r.db.WithContext(ctx).
		Table("interview_interviewers ii").
		Joins("JOIN interviews i ON ii.interview_id = i.id").
		Joins("JOIN users u ON ii.user_id = u.id").
		Where("ii.user_id IN (?)", interviewerIDs).
		Where("i.status = ?", model.InterviewStatusScheduled).
		Where("i.scheduled_start < ? AND i.scheduled_end > ?", end, start)

	if excludeInterviewID != "" {
		query = query.Where("i.id != ?", excludeInterviewID)
	}

	err := query.Select("u.name as user_name").Limit(1).Scan(&result).Error
	if err != nil {
		return "", err
	}

	return result.UserName, nil
}

func (r *gormInterviewRepository) GetCandidateNextInterview(ctx context.Context, candidateID, jobID string) (*model.Interview, error) {
	var item model.Interview
	now := time.Now().UTC()

	query := r.db.WithContext(ctx).
		Preload("Interviewers.User").
		Where("candidate_id = ?", candidateID).
		Where("status = ?", model.InterviewStatusScheduled).
		Where("scheduled_end >= ?", now)

	if jobID != "" {
		query = query.Where("job_id = ?", jobID)
	}

	err := query.Order("scheduled_start ASC").First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No upcoming interview
		}
		return nil, err
	}
	return &item, nil
}

func (r *gormInterviewRepository) ListByCandidateAndJob(ctx context.Context, candidateID, jobID string) ([]model.Interview, error) {
	var items []model.Interview
	query := r.db.WithContext(ctx).
		Preload("Job").
		Preload("Candidate").
		Preload("Interviewers.User").
		Where("candidate_id = ?", candidateID)

	if jobID != "" {
		query = query.Where("job_id = ?", jobID)
	}

	err := query.Order("scheduled_start DESC").Find(&items).Error
	if err != nil {
		return nil, err
	}
	return items, nil
}
