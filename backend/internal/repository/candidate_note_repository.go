package repository

import (
	"context"
	"errors"
	"time"

	"hirescope/backend/internal/model"

	"gorm.io/gorm"
)

var (
	ErrNoteNotFound = errors.New("candidate note not found")
)

// CandidateNoteListResult represents paginated candidate notes with total counts.
type CandidateNoteListResult struct {
	Items      []model.CandidateNote `json:"items"`
	Total      int64                 `json:"total"`
	Page       int                   `json:"page"`
	Limit      int                   `json:"limit"`
	TotalPages int                   `json:"total_pages"`
}

// CandidateNoteRepository defines database access for job-contextual candidate notes.
type CandidateNoteRepository interface {
	Create(ctx context.Context, note *model.CandidateNote) error
	GetByID(ctx context.Context, noteID string) (*model.CandidateNote, error)
	ListByJobAndCandidate(ctx context.Context, jobID, candidateID string, page, limit int) (*CandidateNoteListResult, error)
	Update(ctx context.Context, note *model.CandidateNote) error
	Delete(ctx context.Context, noteID string) error
}

type gormCandidateNoteRepository struct {
	db *gorm.DB
}

// NewCandidateNoteRepository creates a new GORM-backed CandidateNoteRepository.
func NewCandidateNoteRepository(db *gorm.DB) CandidateNoteRepository {
	return &gormCandidateNoteRepository{db: db}
}

func (r *gormCandidateNoteRepository) Create(ctx context.Context, note *model.CandidateNote) error {
	return r.db.WithContext(ctx).Create(note).Error
}

func (r *gormCandidateNoteRepository) GetByID(ctx context.Context, noteID string) (*model.CandidateNote, error) {
	var note model.CandidateNote
	err := r.db.WithContext(ctx).
		Preload("Author").
		Where("id = ?", noteID).
		First(&note).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNoteNotFound
		}
		return nil, err
	}
	return &note, nil
}

func (r *gormCandidateNoteRepository) ListByJobAndCandidate(ctx context.Context, jobID, candidateID string, page, limit int) (*CandidateNoteListResult, error) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}

	query := r.db.WithContext(ctx).Model(&model.CandidateNote{}).
		Where("job_id = ? AND candidate_id = ?", jobID, candidateID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	offset := (page - 1) * limit
	var notes []model.CandidateNote
	err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Preload("Author").
		Find(&notes).Error
	if err != nil {
		return nil, err
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))
	if totalPages == 0 {
		totalPages = 1
	}

	return &CandidateNoteListResult{
		Items:      notes,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (r *gormCandidateNoteRepository) Update(ctx context.Context, note *model.CandidateNote) error {
	note.UpdatedAt = time.Now().UTC()
	res := r.db.WithContext(ctx).
		Model(note).
		Where("id = ?", note.ID).
		Updates(map[string]interface{}{
			"content":    note.Content,
			"updated_at": note.UpdatedAt,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNoteNotFound
	}
	return nil
}

func (r *gormCandidateNoteRepository) Delete(ctx context.Context, noteID string) error {
	res := r.db.WithContext(ctx).Where("id = ?", noteID).Delete(&model.CandidateNote{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNoteNotFound
	}
	return nil
}
