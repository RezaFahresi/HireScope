package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RecipientType indicates whether the recipient is a candidate or internal interviewer.
type RecipientType string

const (
	RecipientTypeCandidate   RecipientType = "CANDIDATE"
	RecipientTypeInterviewer RecipientType = "INTERVIEWER"
)

// EmailType indicates the purpose of the transactional email.
type EmailType string

const (
	EmailTypeInterviewInvitation EmailType = "INTERVIEW_INVITATION"
	EmailTypeInterviewReminder   EmailType = "INTERVIEW_REMINDER"
)

// DeliveryStatus indicates the lifecycle of an email transmission attempt.
type DeliveryStatus string

const (
	DeliveryStatusPending DeliveryStatus = "PENDING"
	DeliveryStatusSending DeliveryStatus = "SENDING"
	DeliveryStatusSent    DeliveryStatus = "SENT"
	DeliveryStatusFailed  DeliveryStatus = "FAILED"
	DeliveryStatusSkipped DeliveryStatus = "SKIPPED"
)

// InterviewEmailDelivery records transactional email dispatches for interview sessions.
type InterviewEmailDelivery struct {
	ID                   string         `gorm:"type:uuid;primaryKey" json:"id"`
	InterviewID          string         `gorm:"type:uuid;not null;index:idx_delivery_interview" json:"interview_id"`
	RecipientType        RecipientType  `gorm:"type:varchar(50);not null" json:"recipient_type"`
	RecipientUserID      *string        `gorm:"type:uuid;index" json:"recipient_user_id,omitempty"`
	RecipientCandidateID *string        `gorm:"type:uuid;index" json:"recipient_candidate_id,omitempty"`
	RecipientEmail       string         `gorm:"type:varchar(255);not null" json:"recipient_email"`
	RecipientName        string         `gorm:"type:varchar(255);not null" json:"recipient_name"`
	EmailType            EmailType      `gorm:"type:varchar(50);not null;index" json:"email_type"`
	Status               DeliveryStatus `gorm:"type:varchar(50);not null;default:'PENDING';index:idx_delivery_status_sched,priority:1;index:idx_delivery_status" json:"status"`
	Provider             string         `gorm:"type:varchar(50);not null;default:'resend'" json:"provider"`
	ProviderMessageID    *string        `gorm:"type:varchar(255)" json:"provider_message_id,omitempty"`
	IdempotencyKey       string         `gorm:"type:varchar(255);uniqueIndex;not null" json:"idempotency_key"`
	AttemptCount         int            `gorm:"not null;default:0" json:"attempt_count"`
	LastError            *string        `gorm:"type:text" json:"last_error,omitempty"`
	SentAt               *time.Time     `gorm:"type:timestamp" json:"sent_at,omitempty"`
	ScheduledFor         *time.Time     `gorm:"type:timestamp;index:idx_delivery_status_sched,priority:2;index:idx_delivery_sched" json:"scheduled_for,omitempty"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`

	Interview *Interview `gorm:"foreignKey:InterviewID;references:ID" json:"interview,omitempty"`
}

// TableName returns the explicit table name for GORM.
func (InterviewEmailDelivery) TableName() string {
	return "interview_email_deliveries"
}

// BeforeCreate assigns a UUID if empty.
func (d *InterviewEmailDelivery) BeforeCreate(tx *gorm.DB) error {
	if d.ID == "" {
		d.ID = uuid.New().String()
	}
	return nil
}
