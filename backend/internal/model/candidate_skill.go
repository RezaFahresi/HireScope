package model

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CandidateSkill represents a verified or claimed competency of a candidate.
type CandidateSkill struct {
	ID              string    `gorm:"type:uuid;primaryKey" json:"id"`
	CandidateID     string    `gorm:"type:uuid;not null;uniqueIndex:idx_cand_skill" json:"candidate_id"`
	Skill           string    `gorm:"type:varchar(100);not null" json:"skill"`
	NormalizedSkill string    `gorm:"type:varchar(100);not null;uniqueIndex:idx_cand_skill" json:"-"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// BeforeCreate assigns a UUID and normalizes the skill name for case-insensitive uniqueness.
func (s *CandidateSkill) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	s.Skill = strings.TrimSpace(s.Skill)
	if s.Skill == "" {
		return errors.New("skill name cannot be blank")
	}
	s.NormalizedSkill = strings.ToLower(s.Skill)
	return nil
}

// BeforeUpdate normalizes skill name before update.
func (s *CandidateSkill) BeforeUpdate(tx *gorm.DB) error {
	s.Skill = strings.TrimSpace(s.Skill)
	if s.Skill == "" {
		return errors.New("skill name cannot be blank")
	}
	s.NormalizedSkill = strings.ToLower(s.Skill)
	return nil
}
