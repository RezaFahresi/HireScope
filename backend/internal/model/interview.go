package model

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InterviewStage represents the recruitment assessment stage.
type InterviewStage string

const (
	InterviewStagePhoneScreen        InterviewStage = "PHONE_SCREEN"
	InterviewStageHRInterview        InterviewStage = "HR_INTERVIEW"
	InterviewStageTechnicalInterview InterviewStage = "TECHNICAL_INTERVIEW"
	InterviewStageManagerInterview   InterviewStage = "MANAGER_INTERVIEW"
	InterviewStageFinalInterview     InterviewStage = "FINAL_INTERVIEW"
	InterviewStageOther              InterviewStage = "OTHER"
)

// IsValid checks if the interview stage is recognized.
func (s InterviewStage) IsValid() bool {
	switch s {
	case InterviewStagePhoneScreen,
		InterviewStageHRInterview,
		InterviewStageTechnicalInterview,
		InterviewStageManagerInterview,
		InterviewStageFinalInterview,
		InterviewStageOther:
		return true
	default:
		return false
	}
}

// DisplayLabel returns human-friendly stage name.
func (s InterviewStage) DisplayLabel() string {
	switch s {
	case InterviewStagePhoneScreen:
		return "Phone Screen"
	case InterviewStageHRInterview:
		return "HR Interview"
	case InterviewStageTechnicalInterview:
		return "Technical Interview"
	case InterviewStageManagerInterview:
		return "Manager Interview"
	case InterviewStageFinalInterview:
		return "Final Interview"
	case InterviewStageOther:
		return "Other"
	default:
		return string(s)
	}
}

// InterviewType represents the modality of the interview.
type InterviewType string

const (
	InterviewTypeOnline InterviewType = "ONLINE"
	InterviewTypeOnsite InterviewType = "ONSITE"
	InterviewTypePhone  InterviewType = "PHONE"
)

// IsValid checks if the interview type is recognized.
func (t InterviewType) IsValid() bool {
	switch t {
	case InterviewTypeOnline, InterviewTypeOnsite, InterviewTypePhone:
		return true
	default:
		return false
	}
}

// InterviewStatus represents the lifecycle state of an interview.
type InterviewStatus string

const (
	InterviewStatusScheduled InterviewStatus = "SCHEDULED"
	InterviewStatusCompleted InterviewStatus = "COMPLETED"
	InterviewStatusCancelled InterviewStatus = "CANCELLED"
)

// IsValid checks if the interview status is recognized.
func (s InterviewStatus) IsValid() bool {
	switch s {
	case InterviewStatusScheduled, InterviewStatusCompleted, InterviewStatusCancelled:
		return true
	default:
		return false
	}
}

// InterviewResult represents the recruiter/interviewer outcome for a completed interview.
type InterviewResult string

const (
	InterviewResultPassed InterviewResult = "PASSED"
	InterviewResultFailed InterviewResult = "FAILED"
	InterviewResultOnHold InterviewResult = "ON_HOLD"
	InterviewResultNoShow InterviewResult = "NO_SHOW"
)

// IsValid checks if the interview result is recognized.
func (r InterviewResult) IsValid() bool {
	switch r {
	case InterviewResultPassed, InterviewResultFailed, InterviewResultOnHold, InterviewResultNoShow:
		return true
	default:
		return false
	}
}

// Interview represents a scheduled candidate evaluation session.
type Interview struct {
	ID                 string           `gorm:"type:uuid;primaryKey" json:"id"`
	JobID              string           `gorm:"type:uuid;not null;index" json:"job_id"`
	CandidateID        string           `gorm:"type:uuid;not null;index" json:"candidate_id"`
	Title              string           `gorm:"type:varchar(255);not null" json:"title"`
	Stage              InterviewStage   `gorm:"type:varchar(50);not null;index" json:"stage"`
	InterviewType      InterviewType    `gorm:"type:varchar(50);not null;index" json:"interview_type"`
	ScheduledStart     time.Time        `gorm:"not null;index" json:"scheduled_start"`
	ScheduledEnd       time.Time        `gorm:"not null;index" json:"scheduled_end"`
	Timezone           string           `gorm:"type:varchar(50);not null;default:'Asia/Jakarta'" json:"timezone"`
	MeetingURL         *string          `gorm:"type:varchar(500)" json:"meeting_url,omitempty"`
	Location           *string          `gorm:"type:varchar(255)" json:"location,omitempty"`
	Notes              *string          `gorm:"type:text" json:"notes,omitempty"`
	Status             InterviewStatus  `gorm:"type:varchar(50);not null;default:'SCHEDULED';index" json:"status"`
	CreatedBy          string           `gorm:"type:uuid;not null;index" json:"created_by"`
	CompletedAt        *time.Time       `gorm:"type:timestamp" json:"completed_at,omitempty"`
	CompletedBy        *string          `gorm:"type:uuid" json:"completed_by,omitempty"`
	Result             *InterviewResult `gorm:"type:varchar(50)" json:"result,omitempty"`
	Feedback           *string          `gorm:"type:text" json:"feedback,omitempty"`
	CancelledAt        *time.Time       `gorm:"type:timestamp" json:"cancelled_at,omitempty"`
	CancelledBy        *string          `gorm:"type:uuid" json:"cancelled_by,omitempty"`
	CancellationReason *string          `gorm:"type:varchar(1000)" json:"cancellation_reason,omitempty"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`

	Job           *Job                   `gorm:"foreignKey:JobID;references:ID" json:"job,omitempty"`
	Candidate     *Candidate             `gorm:"foreignKey:CandidateID;references:ID" json:"candidate,omitempty"`
	Creator       *User                  `gorm:"foreignKey:CreatedBy;references:ID" json:"creator,omitempty"`
	CompletedUser *User                  `gorm:"foreignKey:CompletedBy;references:ID" json:"completed_user,omitempty"`
	CancelledUser *User                  `gorm:"foreignKey:CancelledBy;references:ID" json:"cancelled_user,omitempty"`
	Interviewers  []InterviewInterviewer `gorm:"foreignKey:InterviewID;references:ID;constraint:OnDelete:CASCADE" json:"interviewers,omitempty"`
}

// BeforeCreate assigns a UUID and sets defaults.
func (i *Interview) BeforeCreate(tx *gorm.DB) error {
	if i.ID == "" {
		i.ID = uuid.New().String()
	}
	i.Title = strings.TrimSpace(i.Title)
	if i.Title == "" {
		return errors.New("interview title cannot be empty")
	}
	if i.Status == "" {
		i.Status = InterviewStatusScheduled
	}
	if i.Timezone == "" {
		i.Timezone = "Asia/Jakarta"
	}
	return nil
}

// InterviewInterviewer represents an assigned interviewer (HireScope user) for an interview session.
type InterviewInterviewer struct {
	ID          string    `gorm:"type:uuid;primaryKey" json:"id"`
	InterviewID string    `gorm:"type:uuid;not null;uniqueIndex:idx_interview_user" json:"interview_id"`
	UserID      string    `gorm:"type:uuid;not null;uniqueIndex:idx_interview_user;index" json:"user_id"`
	CreatedAt   time.Time `json:"created_at"`

	Interview *Interview `gorm:"foreignKey:InterviewID;references:ID" json:"interview,omitempty"`
	User      *User      `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
}

// BeforeCreate assigns a UUID.
func (ii *InterviewInterviewer) BeforeCreate(tx *gorm.DB) error {
	if ii.ID == "" {
		ii.ID = uuid.New().String()
	}
	return nil
}
