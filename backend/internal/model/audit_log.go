package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditAction represents an action in the recruiter workflow audit trail.
type AuditAction string

const (
	AuditActionCandidateStatusChanged AuditAction = "CANDIDATE_STATUS_CHANGED"
	AuditActionCandidateNoteCreated   AuditAction = "CANDIDATE_NOTE_CREATED"
	AuditActionCandidateNoteUpdated   AuditAction = "CANDIDATE_NOTE_UPDATED"
	AuditActionCandidateNoteDeleted   AuditAction = "CANDIDATE_NOTE_DELETED"
	AuditActionInterviewCreated         AuditAction = "INTERVIEW_CREATED"
	AuditActionInterviewUpdated         AuditAction = "INTERVIEW_UPDATED"
	AuditActionInterviewRescheduled     AuditAction = "INTERVIEW_RESCHEDULED"
	AuditActionInterviewCancelled       AuditAction = "INTERVIEW_CANCELLED"
	AuditActionInterviewCompleted       AuditAction = "INTERVIEW_COMPLETED"
	AuditActionInterviewFeedbackUpdated AuditAction = "INTERVIEW_FEEDBACK_UPDATED"
	AuditActionInterviewInvitationSent  AuditAction = "INTERVIEW_INVITATION_SENT"
	AuditActionInterviewReminderSent    AuditAction = "INTERVIEW_REMINDER_SENT"
	AuditActionInterviewEmailRetried    AuditAction = "INTERVIEW_EMAIL_RETRIED"
	AuditActionCalendarConnected        AuditAction = "CALENDAR_CONNECTED"
	AuditActionCalendarDisconnected     AuditAction = "CALENDAR_DISCONNECTED"
	AuditActionCalendarEventCreated     AuditAction = "CALENDAR_EVENT_CREATED"
	AuditActionCalendarEventUpdated     AuditAction = "CALENDAR_EVENT_UPDATED"
	AuditActionCalendarEventDeleted     AuditAction = "CALENDAR_EVENT_DELETED"
	AuditActionCalendarSyncFailed       AuditAction = "CALENDAR_SYNC_FAILED"
	AuditActionAnalyticsExported        AuditAction = "ANALYTICS_EXPORTED"
)

// AuditLog records safe, structured audit events for recruiter workflow decisions.
type AuditLog struct {
	ID          string      `gorm:"type:uuid;primaryKey" json:"id"`
	Action      AuditAction `gorm:"type:varchar(100);not null;index" json:"action"`
	ActorID     string      `gorm:"type:uuid;not null;index" json:"actor_id"`
	JobID       *string     `gorm:"type:uuid;index" json:"job_id,omitempty"`
	CandidateID *string     `gorm:"type:uuid;index" json:"candidate_id,omitempty"`
	Metadata    string      `gorm:"type:text" json:"metadata,omitempty"`
	CreatedAt   time.Time   `gorm:"not null;index" json:"created_at"`

	Actor *User `gorm:"foreignKey:ActorID;references:ID" json:"actor,omitempty"`
}

// BeforeCreate assigns a UUID and creation timestamp.
func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	if a.CreatedAt.IsZero() {
		a.CreatedAt = time.Now().UTC()
	}
	return nil
}
