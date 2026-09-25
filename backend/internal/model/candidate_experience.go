package model

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CandidateExperience represents professional work history for a candidate.
type CandidateExperience struct {
	ID             string          `gorm:"type:uuid;primaryKey" json:"id"`
	CandidateID    string          `gorm:"type:uuid;not null;index" json:"candidate_id"`
	Company        string          `gorm:"type:varchar(255);not null" json:"company"`
	Position       string          `gorm:"type:varchar(255);not null" json:"position"`
	Location       string          `gorm:"type:varchar(100)" json:"location,omitempty"`
	EmploymentType *EmploymentType `gorm:"type:varchar(50)" json:"employment_type,omitempty"`
	StartDate      *time.Time      `gorm:"type:date" json:"start_date,omitempty"`
	EndDate        *time.Time      `gorm:"type:date" json:"end_date,omitempty"`
	IsCurrent      bool            `gorm:"default:false" json:"is_current"`
	Description    string          `gorm:"type:text" json:"description,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

// BeforeCreate assigns a UUID and validates company, position, and date integrity.
func (exp *CandidateExperience) BeforeCreate(tx *gorm.DB) error {
	if exp.ID == "" {
		exp.ID = uuid.New().String()
	}
	exp.Company = strings.TrimSpace(exp.Company)
	if exp.Company == "" {
		return errors.New("company name cannot be blank")
	}
	exp.Position = strings.TrimSpace(exp.Position)
	if exp.Position == "" {
		return errors.New("position cannot be blank")
	}
	if exp.EmploymentType != nil && !exp.EmploymentType.IsValid() {
		return errors.New("invalid employment type")
	}
	if exp.StartDate != nil && exp.EndDate != nil && !exp.IsCurrent && exp.EndDate.Before(*exp.StartDate) {
		return errors.New("end date cannot be before start date")
	}
	return nil
}

// BeforeUpdate validates experience dates and required fields before updating.
func (exp *CandidateExperience) BeforeUpdate(tx *gorm.DB) error {
	exp.Company = strings.TrimSpace(exp.Company)
	if exp.Company == "" {
		return errors.New("company name cannot be blank")
	}
	exp.Position = strings.TrimSpace(exp.Position)
	if exp.Position == "" {
		return errors.New("position cannot be blank")
	}
	if exp.EmploymentType != nil && !exp.EmploymentType.IsValid() {
		return errors.New("invalid employment type")
	}
	if exp.StartDate != nil && exp.EndDate != nil && !exp.IsCurrent && exp.EndDate.Before(*exp.StartDate) {
		return errors.New("end date cannot be before start date")
	}
	return nil
}
