package repository

import (
	"context"

	"hirescope/backend/internal/model"

	"gorm.io/gorm"
)

// AuditLogRepository defines database persistence for recruiter workflow audit events.
type AuditLogRepository interface {
	Create(ctx context.Context, log *model.AuditLog) error
}

type gormAuditLogRepository struct {
	db *gorm.DB
}

// NewAuditLogRepository creates a new GORM-backed AuditLogRepository.
func NewAuditLogRepository(db *gorm.DB) AuditLogRepository {
	return &gormAuditLogRepository{db: db}
}

func (r *gormAuditLogRepository) Create(ctx context.Context, log *model.AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}
