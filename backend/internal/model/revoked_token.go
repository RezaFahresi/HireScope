package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RevokedToken records cryptographically hashed invalidated JWTs.
type RevokedToken struct {
	ID        string    `gorm:"type:uuid;primaryKey" json:"id"`
	TokenHash string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"token_hash"`
	ExpiresAt time.Time `gorm:"not null;index" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// BeforeCreate generates a UUID for the RevokedToken record.
func (rt *RevokedToken) BeforeCreate(tx *gorm.DB) error {
	if rt.ID == "" {
		rt.ID = uuid.New().String()
	}
	return nil
}
