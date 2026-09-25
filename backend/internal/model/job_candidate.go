package model

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// JobCandidateStatus represents the recruiter's workflow status for a candidate in the context of a specific job.
type JobCandidateStatus string

const (
	JobCandidateStatusReview      JobCandidateStatus = "REVIEW"
	JobCandidateStatusShortlisted JobCandidateStatus = "SHORTLISTED"
	JobCandidateStatusRejected    JobCandidateStatus = "REJECTED"
)

// IsValid checks whether the candidate workflow status is recognized.
func (s JobCandidateStatus) IsValid() bool {
	switch s {
	case JobCandidateStatusReview, JobCandidateStatusShortlisted, JobCandidateStatusRejected:
		return true
	default:
		return false
	}
}

// JobCandidate associates a candidate with a specific recruitment vacancy and tracks the recruiter workflow status.
type JobCandidate struct {
	ID          string             `gorm:"type:uuid;primaryKey" json:"id"`
	JobID       string             `gorm:"type:uuid;not null;uniqueIndex:idx_job_candidate" json:"job_id"`
	CandidateID string             `gorm:"type:uuid;not null;uniqueIndex:idx_job_candidate" json:"candidate_id"`
	Status      JobCandidateStatus `gorm:"type:varchar(50);not null;default:'REVIEW';index" json:"status"`
	ReviewedAt  *time.Time         `gorm:"type:timestamp;index" json:"reviewed_at,omitempty"`
	ReviewedBy  *string            `gorm:"type:uuid;index" json:"reviewed_by,omitempty"`
	CreatedAt   time.Time          `gorm:"index" json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`

	Job       *Job       `gorm:"foreignKey:JobID;references:ID" json:"job,omitempty"`
	Candidate *Candidate `gorm:"foreignKey:CandidateID;references:ID" json:"candidate,omitempty"`
	Reviewer  *User      `gorm:"foreignKey:ReviewedBy;references:ID" json:"reviewer,omitempty"`
}

// BeforeCreate assigns a UUID and sets the default workflow status to REVIEW.
func (jc *JobCandidate) BeforeCreate(tx *gorm.DB) error {
	if jc.ID == "" {
		jc.ID = uuid.New().String()
	}
	jc.Status = JobCandidateStatus(strings.ToUpper(strings.TrimSpace(string(jc.Status))))
	if jc.Status == "" {
		jc.Status = JobCandidateStatusReview
	}
	if !jc.Status.IsValid() {
		return errors.New("invalid candidate workflow status")
	}
	return nil
}

// BeforeUpdate validates the workflow status before persistence.
func (jc *JobCandidate) BeforeUpdate(tx *gorm.DB) error {
	jc.Status = JobCandidateStatus(strings.ToUpper(strings.TrimSpace(string(jc.Status))))
	if jc.Status == "" {
		jc.Status = JobCandidateStatusReview
	}
	if !jc.Status.IsValid() {
		return errors.New("invalid candidate workflow status")
	}
	return nil
}
