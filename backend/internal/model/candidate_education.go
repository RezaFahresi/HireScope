package model

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CandidateEducation represents an academic credential in a candidate's background.
type CandidateEducation struct {
	ID           string     `gorm:"type:uuid;primaryKey" json:"id"`
	CandidateID  string     `gorm:"type:uuid;not null;index" json:"candidate_id"`
	Institution  string     `gorm:"type:varchar(255);not null" json:"institution"`
	Degree       string     `gorm:"type:varchar(100)" json:"degree,omitempty"`
	FieldOfStudy string     `gorm:"type:varchar(100)" json:"field_of_study,omitempty"`
	StartDate    *time.Time `gorm:"type:date" json:"start_date,omitempty"`
	EndDate      *time.Time `gorm:"type:date" json:"end_date,omitempty"`
	Description  string     `gorm:"type:text" json:"description,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// BeforeCreate assigns a UUID and validates education dates.
func (e *CandidateEducation) BeforeCreate(tx *gorm.DB) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	e.Institution = strings.TrimSpace(e.Institution)
	if e.Institution == "" {
		return errors.New("institution cannot be blank")
	}
	if e.StartDate != nil && e.EndDate != nil && e.EndDate.Before(*e.StartDate) {
		return errors.New("end date cannot be before start date")
	}
	return nil
}

// BeforeUpdate validates education dates before updating.
func (e *CandidateEducation) BeforeUpdate(tx *gorm.DB) error {
	e.Institution = strings.TrimSpace(e.Institution)
	if e.Institution == "" {
		return errors.New("institution cannot be blank")
	}
	if e.StartDate != nil && e.EndDate != nil && e.EndDate.Before(*e.StartDate) {
		return errors.New("end date cannot be before start date")
	}
	return nil
}
