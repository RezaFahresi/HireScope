package repository

import (
	"context"
	"errors"
	"time"

	"hirescope/backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrScreeningResultNotFound = errors.New("screening result not found")
)

// ScreeningRepository defines database operations for candidate screening results and matches.
type ScreeningRepository interface {
	GetByCandidateAndJob(ctx context.Context, candidateID, jobID string) (*model.ScreeningResult, error)
	SaveScreeningResult(ctx context.Context, result *model.ScreeningResult, matches []model.ScreeningMatch) (*model.ScreeningResult, error)
	GetByID(ctx context.Context, id string) (*model.ScreeningResult, error)
}

type gormScreeningRepository struct {
	db *gorm.DB
}

// NewScreeningRepository creates a new GORM-backed ScreeningRepository.
func NewScreeningRepository(db *gorm.DB) ScreeningRepository {
	return &gormScreeningRepository{db: db}
}

func (r *gormScreeningRepository) GetByCandidateAndJob(ctx context.Context, candidateID, jobID string) (*model.ScreeningResult, error) {
	var result model.ScreeningResult
	err := r.db.WithContext(ctx).
		Preload("Matches").
		Preload("Matches.Requirement").
		Where("candidate_id = ? AND job_id = ?", candidateID, jobID).
		First(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrScreeningResultNotFound
		}
		return nil, err
	}
	return &result, nil
}

func (r *gormScreeningRepository) GetByID(ctx context.Context, id string) (*model.ScreeningResult, error) {
	var result model.ScreeningResult
	err := r.db.WithContext(ctx).
		Preload("Matches").
		Preload("Matches.Requirement").
		Where("id = ?", id).
		First(&result).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrScreeningResultNotFound
		}
		return nil, err
	}
	return &result, nil
}

func (r *gormScreeningRepository) SaveScreeningResult(ctx context.Context, result *model.ScreeningResult, matches []model.ScreeningMatch) (*model.ScreeningResult, error) {
	var savedResult model.ScreeningResult
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing model.ScreeningResult
		err := tx.Where("candidate_id = ? AND job_id = ?", result.CandidateID, result.JobID).First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		now := time.Now().UTC()
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Create new
			if result.ID == "" {
				result.ID = uuid.New().String()
			}
			result.EvaluatedAt = now
			if err := tx.Create(result).Error; err != nil {
				return err
			}
			savedResult = *result
		} else {
			// Update existing
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

			if err := tx.Save(&existing).Error; err != nil {
				return err
			}
			savedResult = existing

			// Atomically clear prior matches to prevent duplicate or orphaned matches
			if err := tx.Where("screening_result_id = ?", existing.ID).Delete(&model.ScreeningMatch{}).Error; err != nil {
				return err
			}
		}

		// Insert new matches
		for i := range matches {
			matches[i].ID = uuid.New().String()
			matches[i].ScreeningResultID = savedResult.ID
			if err := tx.Create(&matches[i]).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Reload freshly saved result with full preloads
	return r.GetByCandidateAndJob(ctx, result.CandidateID, result.JobID)
}
