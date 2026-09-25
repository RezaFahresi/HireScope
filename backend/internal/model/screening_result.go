package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ScreeningResultStatus represents the overall qualification status derived deterministically.
type ScreeningResultStatus string

const (
	StatusQualified    ScreeningResultStatus = "QUALIFIED"
	StatusReview       ScreeningResultStatus = "REVIEW"
	StatusNotQualified ScreeningResultStatus = "NOT_QUALIFIED"
)

// IsValid checks if the screening result status is recognized.
func (s ScreeningResultStatus) IsValid() bool {
	switch s {
	case StatusQualified, StatusReview, StatusNotQualified:
		return true
	default:
		return false
	}
}

// ScreeningResult represents the outcome of an automated, deterministic screening run for a Candidate against a Job.
type ScreeningResult struct {
	ID                     string                `gorm:"type:uuid;primaryKey" json:"id"`
	CandidateID            string                `gorm:"type:uuid;not null;uniqueIndex:idx_cand_job_screening;index" json:"candidate_id"`
	JobID                  string                `gorm:"type:uuid;not null;uniqueIndex:idx_cand_job_screening;index" json:"job_id"`
	Status                 ScreeningResultStatus `gorm:"type:varchar(50);not null" json:"status"`
	RequiredMatchCount     int                   `gorm:"not null;default:0" json:"required_match_count"`
	RequiredPartialCount   int                   `gorm:"not null;default:0" json:"required_partial_count"`
	RequiredMismatchCount  int                   `gorm:"not null;default:0" json:"required_mismatch_count"`
	RequiredUnknownCount   int                   `gorm:"not null;default:0" json:"required_unknown_count"`
	PreferredMatchCount    int                   `gorm:"not null;default:0" json:"preferred_match_count"`
	PreferredPartialCount  int                   `gorm:"not null;default:0" json:"preferred_partial_count"`
	PreferredMismatchCount int                   `gorm:"not null;default:0" json:"preferred_mismatch_count"`
	PreferredUnknownCount  int                   `gorm:"not null;default:0" json:"preferred_unknown_count"`
	EvaluatedAt            time.Time             `gorm:"not null" json:"evaluated_at"`
	CreatedAt              time.Time             `json:"created_at"`
	UpdatedAt              time.Time             `json:"updated_at"`

	Candidate *Candidate       `gorm:"foreignKey:CandidateID;references:ID" json:"candidate,omitempty"`
	Job       *Job             `gorm:"foreignKey:JobID;references:ID" json:"job,omitempty"`
	Matches   []ScreeningMatch `gorm:"foreignKey:ScreeningResultID;references:ID;constraint:OnDelete:CASCADE" json:"matches,omitempty"`
}

// BeforeCreate assigns a UUID and sets default timestamp.
func (r *ScreeningResult) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	if !r.Status.IsValid() {
		return errors.New("invalid screening result status")
	}
	if r.EvaluatedAt.IsZero() {
		r.EvaluatedAt = time.Now().UTC()
	}
	return nil
}

// BeforeUpdate validates status.
func (r *ScreeningResult) BeforeUpdate(tx *gorm.DB) error {
	if !r.Status.IsValid() {
		return errors.New("invalid screening result status")
	}
	return nil
}
