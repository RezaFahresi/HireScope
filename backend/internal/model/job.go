package model

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// JobStatus represents the lifecycle state of a recruitment vacancy.
type JobStatus string

const (
	JobStatusDraft    JobStatus = "DRAFT"
	JobStatusOpen     JobStatus = "OPEN"
	JobStatusClosed   JobStatus = "CLOSED"
	JobStatusArchived JobStatus = "ARCHIVED"
)

// IsValid checks if the job status is recognized.
func (s JobStatus) IsValid() bool {
	switch s {
	case JobStatusDraft, JobStatusOpen, JobStatusClosed, JobStatusArchived:
		return true
	default:
		return false
	}
}

// CanTransitionTo determines if transitioning from s to target is permitted by business rules.
func (s JobStatus) CanTransitionTo(target JobStatus) bool {
	if s == target {
		return true
	}
	switch s {
	case JobStatusDraft:
		return target == JobStatusOpen || target == JobStatusArchived
	case JobStatusOpen:
		return target == JobStatusClosed || target == JobStatusArchived
	case JobStatusClosed:
		return target == JobStatusOpen || target == JobStatusArchived
	case JobStatusArchived:
		return false
	default:
		return false
	}
}

// EmploymentType represents the contract classification of a job.
type EmploymentType string

const (
	EmpFullTime   EmploymentType = "FULL_TIME"
	EmpPartTime   EmploymentType = "PART_TIME"
	EmpContract   EmploymentType = "CONTRACT"
	EmpInternship EmploymentType = "INTERNSHIP"
	EmpFreelance  EmploymentType = "FREELANCE"
)

// IsValid checks if the employment type is recognized.
func (e EmploymentType) IsValid() bool {
	switch e {
	case EmpFullTime, EmpPartTime, EmpContract, EmpInternship, EmpFreelance:
		return true
	default:
		return false
	}
}

// Job represents a recruitment vacancy in HireScope.
type Job struct {
	ID             string           `gorm:"type:uuid;primaryKey" json:"id"`
	Code           string           `gorm:"type:varchar(50);uniqueIndex;not null" json:"code"`
	Title          string           `gorm:"type:varchar(255);not null;index" json:"title"`
	Description    string           `gorm:"type:text;not null" json:"description"`
	Department     string           `gorm:"type:varchar(100);index" json:"department"`
	Location       string           `gorm:"type:varchar(100);index" json:"location"`
	EmploymentType EmploymentType   `gorm:"type:varchar(50);not null;index" json:"employment_type"`
	Status         JobStatus        `gorm:"type:varchar(50);not null;default:'DRAFT';index" json:"status"`
	ClosingDate    *time.Time       `gorm:"type:date" json:"closing_date,omitempty"`
	CreatedBy      string           `gorm:"type:uuid;not null;index" json:"created_by"`
	Creator        *User            `gorm:"foreignKey:CreatedBy;references:ID" json:"creator,omitempty"`
	Requirements   []JobRequirement `gorm:"foreignKey:JobID;references:ID" json:"requirements,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	UpdatedAt      time.Time        `json:"updated_at"`
}

// BeforeCreate assigns a UUID if not already present.
func (j *Job) BeforeCreate(tx *gorm.DB) error {
	if j.ID == "" {
		j.ID = uuid.New().String()
	}
	j.Title = strings.TrimSpace(j.Title)
	j.Department = strings.TrimSpace(j.Department)
	j.Location = strings.TrimSpace(j.Location)
	if j.Status == "" {
		j.Status = JobStatusDraft
	}
	if !j.Status.IsValid() {
		return errors.New("invalid job status")
	}
	if !j.EmploymentType.IsValid() {
		return errors.New("invalid employment type")
	}
	return nil
}
