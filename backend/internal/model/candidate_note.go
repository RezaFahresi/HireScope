package model

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CandidateNote represents a job-contextual recruiter observation, evaluation note, or interview comment.
type CandidateNote struct {
	ID          string    `gorm:"type:uuid;primaryKey" json:"id"`
	JobID       string    `gorm:"type:uuid;not null;index" json:"job_id"`
	CandidateID string    `gorm:"type:uuid;not null;index" json:"candidate_id"`
	AuthorID    string    `gorm:"type:uuid;not null;index" json:"author_id"`
	Content     string    `gorm:"type:text;not null" json:"content"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Job       *Job       `gorm:"foreignKey:JobID;references:ID" json:"job,omitempty"`
	Candidate *Candidate `gorm:"foreignKey:CandidateID;references:ID" json:"candidate,omitempty"`
	Author    *User      `gorm:"foreignKey:AuthorID;references:ID" json:"author,omitempty"`
}

// BeforeCreate assigns a UUID and validates note content.
func (n *CandidateNote) BeforeCreate(tx *gorm.DB) error {
	if n.ID == "" {
		n.ID = uuid.New().String()
	}
	n.Content = strings.TrimSpace(n.Content)
	if n.Content == "" {
		return errors.New("note content cannot be empty")
	}
	if len(n.Content) > 5000 {
		return errors.New("note content cannot exceed 5000 characters")
	}
	return nil
}

// BeforeUpdate validates note content before updating.
func (n *CandidateNote) BeforeUpdate(tx *gorm.DB) error {
	n.Content = strings.TrimSpace(n.Content)
	if n.Content == "" {
		return errors.New("note content cannot be empty")
	}
	if len(n.Content) > 5000 {
		return errors.New("note content cannot exceed 5000 characters")
	}
	return nil
}
