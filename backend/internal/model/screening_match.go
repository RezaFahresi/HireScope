package model

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MatchStatus represents the evaluation status for an individual job requirement.
type MatchStatus string

const (
	MatchStatusMatch         MatchStatus = "MATCH"
	MatchStatusPartial       MatchStatus = "PARTIAL"
	MatchStatusMismatch      MatchStatus = "MISMATCH"
	MatchStatusUnknown       MatchStatus = "UNKNOWN"
	MatchStatusNotApplicable MatchStatus = "NOT_APPLICABLE"
)

// IsValid checks if the match status is recognized.
func (s MatchStatus) IsValid() bool {
	switch s {
	case MatchStatusMatch, MatchStatusPartial, MatchStatusMismatch, MatchStatusUnknown, MatchStatusNotApplicable:
		return true
	default:
		return false
	}
}

// MatchConfidence represents the degree of certainty in the match result.
type MatchConfidence string

const (
	ConfidenceHigh   MatchConfidence = "HIGH"
	ConfidenceMedium MatchConfidence = "MEDIUM"
	ConfidenceLow    MatchConfidence = "LOW"
)

// IsValid checks if the match confidence is recognized.
func (c MatchConfidence) IsValid() bool {
	switch c {
	case ConfidenceHigh, ConfidenceMedium, ConfidenceLow:
		return true
	default:
		return false
	}
}

// MatchSource represents the candidate entity that supplied the evidence.
type MatchSource string

const (
	SourceSkill         MatchSource = "SKILL"
	SourceExperience    MatchSource = "EXPERIENCE"
	SourceEducation     MatchSource = "EDUCATION"
	SourceCertification MatchSource = "CERTIFICATION"
	SourceLanguage      MatchSource = "LANGUAGE"
	SourceProfile       MatchSource = "PROFILE"
	SourceNone          MatchSource = "NONE"
)

// IsValid checks if the match source is recognized.
func (s MatchSource) IsValid() bool {
	switch s {
	case SourceSkill, SourceExperience, SourceEducation, SourceCertification, SourceLanguage, SourceProfile, SourceNone:
		return true
	default:
		return false
	}
}

// ScreeningMatch represents the detailed evaluation of a single requirement in a screening result.
type ScreeningMatch struct {
	ID                string          `gorm:"type:uuid;primaryKey" json:"id"`
	ScreeningResultID string          `gorm:"type:uuid;not null;uniqueIndex:idx_result_requirement;index" json:"screening_result_id"`
	RequirementID     string          `gorm:"type:uuid;not null;uniqueIndex:idx_result_requirement;index" json:"requirement_id"`
	Status            MatchStatus     `gorm:"type:varchar(50);not null" json:"status"`
	Evidence          string          `gorm:"type:text;not null" json:"evidence"`
	Reason            string          `gorm:"type:text;not null" json:"reason"`
	Confidence        MatchConfidence `gorm:"type:varchar(50);not null;default:'HIGH'" json:"confidence"`
	Source            MatchSource     `gorm:"type:varchar(50);not null;default:'NONE'" json:"source"`
	MatchedValue      string          `gorm:"type:varchar(500)" json:"matched_value,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`

	ScreeningResult *ScreeningResult `gorm:"foreignKey:ScreeningResultID;references:ID" json:"-"`
	Requirement     *JobRequirement  `gorm:"foreignKey:RequirementID;references:ID" json:"requirement_details,omitempty"`
}

// BeforeCreate assigns a UUID and validates fields.
func (m *ScreeningMatch) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	if !m.Status.IsValid() {
		return errors.New("invalid match status")
	}
	if !m.Confidence.IsValid() {
		return errors.New("invalid match confidence")
	}
	if !m.Source.IsValid() {
		return errors.New("invalid match source")
	}
	return nil
}

// BeforeUpdate validates fields before saving.
func (m *ScreeningMatch) BeforeUpdate(tx *gorm.DB) error {
	if !m.Status.IsValid() {
		return errors.New("invalid match status")
	}
	if !m.Confidence.IsValid() {
		return errors.New("invalid match confidence")
	}
	if !m.Source.IsValid() {
		return errors.New("invalid match source")
	}
	return nil
}
