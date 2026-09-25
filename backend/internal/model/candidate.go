package model

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Candidate represents a job applicant profile in HireScope.
// Note: Candidates are distinct from application Users (recruiters/admins).
type Candidate struct {
	ID          string                `gorm:"type:uuid;primaryKey" json:"id"`
	FullName    string                `gorm:"type:varchar(255);not null;index" json:"full_name"`
	Email       string                `gorm:"type:varchar(255);index" json:"email,omitempty"`
	Phone       string                `gorm:"type:varchar(50)" json:"phone,omitempty"`
	Location    string                `gorm:"type:varchar(100);index" json:"location,omitempty"`
	Headline    string                `gorm:"type:varchar(255)" json:"headline,omitempty"`
	Summary     string                `gorm:"type:text" json:"summary,omitempty"`
	CreatedBy   string                `gorm:"type:uuid;not null;index" json:"created_by"`
	Creator     *User                 `gorm:"foreignKey:CreatedBy;references:ID" json:"creator,omitempty"`
	Documents   []CandidateDocument   `gorm:"foreignKey:CandidateID;references:ID" json:"documents,omitempty"`
	Educations  []CandidateEducation  `gorm:"foreignKey:CandidateID;references:ID" json:"educations,omitempty"`
	Experiences []CandidateExperience `gorm:"foreignKey:CandidateID;references:ID" json:"experiences,omitempty"`
	Skills      []CandidateSkill      `gorm:"foreignKey:CandidateID;references:ID" json:"skills,omitempty"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
}

// BeforeCreate assigns a UUID and normalizes candidate fields.
func (c *Candidate) BeforeCreate(tx *gorm.DB) error {
	if c.ID == "" {
		c.ID = uuid.New().String()
	}
	c.FullName = strings.TrimSpace(c.FullName)
	if c.FullName == "" {
		return errors.New("full name cannot be blank")
	}
	if len(c.FullName) > 255 {
		return errors.New("full name cannot exceed 255 characters")
	}
	if c.Email != "" {
		c.Email = NormalizeEmail(c.Email)
	}
	c.Phone = strings.TrimSpace(c.Phone)
	c.Location = strings.TrimSpace(c.Location)
	c.Headline = strings.TrimSpace(c.Headline)
	c.Summary = strings.TrimSpace(c.Summary)
	return nil
}

// BeforeUpdate trims and validates fields before updating.
func (c *Candidate) BeforeUpdate(tx *gorm.DB) error {
	c.FullName = strings.TrimSpace(c.FullName)
	if c.FullName == "" {
		return errors.New("full name cannot be blank")
	}
	if len(c.FullName) > 255 {
		return errors.New("full name cannot exceed 255 characters")
	}
	if c.Email != "" {
		c.Email = NormalizeEmail(c.Email)
	}
	c.Phone = strings.TrimSpace(c.Phone)
	c.Location = strings.TrimSpace(c.Location)
	c.Headline = strings.TrimSpace(c.Headline)
	c.Summary = strings.TrimSpace(c.Summary)
	return nil
}
