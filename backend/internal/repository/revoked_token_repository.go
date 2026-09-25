package repository

import (
	"context"
	"time"

	"hirescope/backend/internal/model"

	"gorm.io/gorm"
)

// RevokedTokenRepository handles operations on invalidated JWT tokens.
type RevokedTokenRepository interface {
	Revoke(ctx context.Context, tokenHash string, expiresAt time.Time) error
	IsRevoked(ctx context.Context, tokenHash string) (bool, error)
	CleanExpired(ctx context.Context) error
}

type gormRevokedTokenRepository struct {
	db *gorm.DB
}

// NewRevokedTokenRepository creates a new RevokedTokenRepository.
func NewRevokedTokenRepository(db *gorm.DB) RevokedTokenRepository {
	return &gormRevokedTokenRepository{db: db}
}

func (r *gormRevokedTokenRepository) Revoke(ctx context.Context, tokenHash string, expiresAt time.Time) error {
	revoked := &model.RevokedToken{
		TokenHash: tokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now().UTC(),
	}
	return r.db.WithContext(ctx).Create(revoked).Error
}

func (r *gormRevokedTokenRepository) IsRevoked(ctx context.Context, tokenHash string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.RevokedToken{}).
		Where("token_hash = ? AND expires_at > ?", tokenHash, time.Now().UTC()).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *gormRevokedTokenRepository) CleanExpired(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Where("expires_at <= ?", time.Now().UTC()).
		Delete(&model.RevokedToken{}).Error
}
