package model

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RequirementCategory classifies the qualification criteria.
type RequirementCategory string

const (
	CategorySkill         RequirementCategory = "SKILL"
	CategoryExperience    RequirementCategory = "EXPERIENCE"
	CategoryEducation     RequirementCategory = "EDUCATION"
	CategoryCertification RequirementCategory = "CERTIFICATION"
	CategoryLanguage      RequirementCategory = "LANGUAGE"
	CategoryOther         RequirementCategory = "OTHER"
)

// IsValid checks if the requirement category is recognized.
func (c RequirementCategory) IsValid() bool {
	switch c {
	case CategorySkill, CategoryExperience, CategoryEducation, CategoryCertification, CategoryLanguage, CategoryOther:
		return true
	default:
		return false
	}
}

// RequirementImportance specifies whether the criteria is mandatory or optional.
type RequirementImportance string

const (
	ImportanceRequired  RequirementImportance = "REQUIRED"
	ImportancePreferred RequirementImportance = "PREFERRED"
)

// IsValid checks if the requirement importance is recognized.
func (i RequirementImportance) IsValid() bool {
	return i == ImportanceRequired || i == ImportancePreferred
}

// JobRequirement represents a specific recruitment requirement for a Job.
type JobRequirement struct {
	ID          string                `gorm:"type:uuid;primaryKey" json:"id"`
	JobID       string                `gorm:"type:uuid;not null;index" json:"job_id"`
	Category    RequirementCategory   `gorm:"type:varchar(50);not null;index" json:"category"`
	Requirement string                `gorm:"type:varchar(500);not null" json:"requirement"`
	Importance  RequirementImportance `gorm:"type:varchar(50);not null;default:'REQUIRED'" json:"importance"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
}

// BeforeCreate assigns a UUID and validates category and importance.
func (r *JobRequirement) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	r.Requirement = strings.TrimSpace(r.Requirement)
	if r.Requirement == "" {
		return errors.New("requirement text cannot be blank")
	}
	if !r.Category.IsValid() {
		return errors.New("invalid requirement category")
	}
	if r.Importance == "" {
		r.Importance = ImportanceRequired
	}
	if !r.Importance.IsValid() {
		return errors.New("invalid requirement importance")
	}
	return nil
}
