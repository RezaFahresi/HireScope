package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"hirescope/backend/internal/model"

	"gorm.io/gorm"
)

var (
	ErrDeliveryNotFound = errors.New("email delivery record not found")
)

// InterviewEmailDeliveryRepository defines database operations for interview email deliveries.
type InterviewEmailDeliveryRepository interface {
	Create(ctx context.Context, delivery *model.InterviewEmailDelivery) error
	GetByID(ctx context.Context, id string) (*model.InterviewEmailDelivery, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*model.InterviewEmailDelivery, error)
	ListByInterviewID(ctx context.Context, interviewID string) ([]*model.InterviewEmailDelivery, error)
	Update(ctx context.Context, delivery *model.InterviewEmailDelivery) error
	GetDuePendingReminders(ctx context.Context, dueBefore time.Time, limit int) ([]*model.InterviewEmailDelivery, error)
	ClaimReminder(ctx context.Context, id string) (bool, error)
	InvalidatePendingReminders(ctx context.Context, interviewID string, reason string) error
}

type interviewEmailDeliveryRepository struct {
	db *gorm.DB
}

// NewInterviewEmailDeliveryRepository creates a new repository instance.
func NewInterviewEmailDeliveryRepository(db *gorm.DB) InterviewEmailDeliveryRepository {
	return &interviewEmailDeliveryRepository{db: db}
}

func (r *interviewEmailDeliveryRepository) Create(ctx context.Context, delivery *model.InterviewEmailDelivery) error {
	if err := r.db.WithContext(ctx).Create(delivery).Error; err != nil {
		return fmt.Errorf("failed to create email delivery: %w", err)
	}
	return nil
}

func (r *interviewEmailDeliveryRepository) GetByID(ctx context.Context, id string) (*model.InterviewEmailDelivery, error) {
	var delivery model.InterviewEmailDelivery
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&delivery).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDeliveryNotFound
		}
		return nil, fmt.Errorf("failed to get email delivery by id: %w", err)
	}
	return &delivery, nil
}

func (r *interviewEmailDeliveryRepository) GetByIdempotencyKey(ctx context.Context, key string) (*model.InterviewEmailDelivery, error) {
	var delivery model.InterviewEmailDelivery
	if err := r.db.WithContext(ctx).Where("idempotency_key = ?", key).First(&delivery).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDeliveryNotFound
		}
		return nil, fmt.Errorf("failed to get email delivery by idempotency key: %w", err)
	}
	return &delivery, nil
}

func (r *interviewEmailDeliveryRepository) ListByInterviewID(ctx context.Context, interviewID string) ([]*model.InterviewEmailDelivery, error) {
	var deliveries []*model.InterviewEmailDelivery
	if err := r.db.WithContext(ctx).
		Where("interview_id = ?", interviewID).
		Order("created_at ASC").
		Find(&deliveries).Error; err != nil {
		return nil, fmt.Errorf("failed to list email deliveries: %w", err)
	}
	return deliveries, nil
}

func (r *interviewEmailDeliveryRepository) Update(ctx context.Context, delivery *model.InterviewEmailDelivery) error {
	delivery.UpdatedAt = time.Now().UTC()
	if err := r.db.WithContext(ctx).Save(delivery).Error; err != nil {
		return fmt.Errorf("failed to update email delivery: %w", err)
	}
	return nil
}

func (r *interviewEmailDeliveryRepository) GetDuePendingReminders(ctx context.Context, dueBefore time.Time, limit int) ([]*model.InterviewEmailDelivery, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var deliveries []*model.InterviewEmailDelivery
	if err := r.db.WithContext(ctx).
		Where("email_type = ? AND status = ? AND scheduled_for <= ?",
			model.EmailTypeInterviewReminder, model.DeliveryStatusPending, dueBefore).
		Order("scheduled_for ASC").
		Limit(limit).
		Find(&deliveries).Error; err != nil {
		return nil, fmt.Errorf("failed to get due reminders: %w", err)
	}
	return deliveries, nil
}

func (r *interviewEmailDeliveryRepository) ClaimReminder(ctx context.Context, id string) (bool, error) {
	now := time.Now().UTC()
	res := r.db.WithContext(ctx).
		Model(&model.InterviewEmailDelivery{}).
		Where("id = ? AND status = ?", id, model.DeliveryStatusPending).
		Updates(map[string]interface{}{
			"status":        model.DeliveryStatusSending,
			"attempt_count": gorm.Expr("attempt_count + 1"),
			"updated_at":    now,
		})
	if res.Error != nil {
		return false, fmt.Errorf("failed to claim reminder: %w", res.Error)
	}
	return res.RowsAffected > 0, nil
}

func (r *interviewEmailDeliveryRepository) InvalidatePendingReminders(ctx context.Context, interviewID string, reason string) error {
	now := time.Now().UTC()
	err := r.db.WithContext(ctx).
		Model(&model.InterviewEmailDelivery{}).
		Where("interview_id = ? AND email_type = ? AND status = ?",
			interviewID, model.EmailTypeInterviewReminder, model.DeliveryStatusPending).
		Updates(map[string]interface{}{
			"status":     model.DeliveryStatusSkipped,
			"last_error": reason,
			"updated_at": now,
		}).Error
	if err != nil {
		return fmt.Errorf("failed to invalidate pending reminders: %w", err)
	}
	return nil
}
