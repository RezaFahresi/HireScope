package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"hirescope/backend/internal/model"
	"hirescope/backend/internal/repository"

	"github.com/google/uuid"
)

var (
	ErrCandidateNotAssociatedWithJob = errors.New("candidate is not associated with this job")
	ErrInvalidInterviewStage         = errors.New("invalid interview stage")
	ErrInvalidInterviewType          = errors.New("invalid interview type")
	ErrInvalidInterviewTime          = errors.New("scheduled end time must be after scheduled start time")
	ErrInterviewDurationTooLong      = errors.New("interview duration cannot exceed 8 hours")
	ErrMissingMeetingURL             = errors.New("meeting URL is required for online interviews")
	ErrMissingLocation               = errors.New("location is required for onsite interviews")
	ErrNoInterviewers                = errors.New("at least one interviewer must be selected")
	ErrInterviewerConflict           = errors.New("interviewer scheduling conflict")
	ErrCancellationReasonRequired    = errors.New("cancellation reason is required")
	ErrFeedbackTooLong               = errors.New("feedback cannot exceed 5000 characters")
	ErrCancellationReasonTooLong     = errors.New("cancellation reason cannot exceed 1000 characters")
	ErrInvalidInterviewResult        = errors.New("invalid interview result")
	ErrInvalidDateRange              = errors.New("'to' date must be after 'from' date")
	ErrDateRangeTooLarge             = errors.New("date range cannot exceed 366 days")
)

// DTOs

type InterviewJobDTO struct {
	ID             string               `json:"id"`
	Code           string               `json:"code"`
	Title          string               `json:"title"`
	Department     string               `json:"department"`
	Location       string               `json:"location"`
	EmploymentType model.EmploymentType `json:"employment_type"`
	Status         model.JobStatus      `json:"status"`
}

type InterviewCandidateDTO struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Location string `json:"location"`
	Headline string `json:"headline"`
}

type InterviewerDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

type InterviewDTO struct {
	ID                 string                 `json:"id"`
	JobID              string                 `json:"job_id"`
	CandidateID        string                 `json:"candidate_id"`
	Title              string                 `json:"title"`
	Stage              model.InterviewStage   `json:"stage"`
	StageLabel         string                 `json:"stage_label"`
	InterviewType      model.InterviewType    `json:"interview_type"`
	ScheduledStart     time.Time              `json:"scheduled_start"`
	ScheduledEnd       time.Time              `json:"scheduled_end"`
	Timezone           string                 `json:"timezone"`
	MeetingURL         *string                `json:"meeting_url,omitempty"`
	Location           *string                `json:"location,omitempty"`
	Notes              *string                `json:"notes,omitempty"`
	Status             model.InterviewStatus  `json:"status"`
	CreatedBy          string                 `json:"created_by"`
	CompletedAt        *time.Time             `json:"completed_at,omitempty"`
	CompletedBy        *string                `json:"completed_by,omitempty"`
	Result             *model.InterviewResult `json:"result,omitempty"`
	Feedback           *string                `json:"feedback,omitempty"`
	CancelledAt        *time.Time             `json:"cancelled_at,omitempty"`
	CancelledBy        *string                `json:"cancelled_by,omitempty"`
	CancellationReason *string                `json:"cancellation_reason,omitempty"`
	CreatedAt          time.Time              `json:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at"`

	Job           *InterviewJobDTO       `json:"job,omitempty"`
	Candidate     *InterviewCandidateDTO `json:"candidate,omitempty"`
	CreatorName   string                 `json:"creator_name,omitempty"`
	CompletedName string                 `json:"completed_name,omitempty"`
	CancelledName string                 `json:"cancelled_name,omitempty"`
	Interviewers  []InterviewerDTO       `json:"interviewers"`
}

type InterviewListDTO struct {
	Items      []InterviewDTO `json:"items"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	Limit      int            `json:"limit"`
	TotalPages int            `json:"total_pages"`
}

type CreateInterviewInput struct {
	JobID                     string               `json:"job_id" binding:"required"`
	CandidateID               string               `json:"candidate_id" binding:"required"`
	Title                     string               `json:"title" binding:"required"`
	Stage                     model.InterviewStage `json:"stage" binding:"required"`
	InterviewType             model.InterviewType  `json:"interview_type" binding:"required"`
	ScheduledStart            time.Time            `json:"scheduled_start" binding:"required"`
	ScheduledEnd              time.Time            `json:"scheduled_end" binding:"required"`
	Timezone                  string               `json:"timezone"`
	MeetingURL                *string              `json:"meeting_url"`
	Location                  *string              `json:"location"`
	Notes                     *string              `json:"notes"`
	InterviewerIDs            []string             `json:"interviewer_ids" binding:"required"`
	SendCandidateInvitation   bool                 `json:"send_candidate_invitation"`
	SendInterviewerInvitation bool                 `json:"send_interviewer_invitation"`
}

type UpdateInterviewInput struct {
	Title          string               `json:"title" binding:"required"`
	Stage          model.InterviewStage `json:"stage" binding:"required"`
	InterviewType  model.InterviewType  `json:"interview_type" binding:"required"`
	ScheduledStart time.Time            `json:"scheduled_start" binding:"required"`
	ScheduledEnd   time.Time            `json:"scheduled_end" binding:"required"`
	Timezone       string               `json:"timezone"`
	MeetingURL     *string              `json:"meeting_url"`
	Location       *string              `json:"location"`
	Notes          *string              `json:"notes"`
	InterviewerIDs []string             `json:"interviewer_ids" binding:"required"`
}

type RescheduleInterviewInput struct {
	ScheduledStart time.Time `json:"scheduled_start" binding:"required"`
	ScheduledEnd   time.Time `json:"scheduled_end" binding:"required"`
	Reason         *string   `json:"reason"`
}

type CancelInterviewInput struct {
	Reason string `json:"reason" binding:"required"`
}

type CompleteInterviewInput struct {
	Result   model.InterviewResult `json:"result" binding:"required"`
	Feedback *string               `json:"feedback"`
}

type InterviewFilterInput struct {
	JobID         string
	CandidateID   string
	Status        string
	Stage         string
	InterviewType string
	From          *time.Time
	To            *time.Time
	Page          int
	Limit         int
	Sort          string
	Order         string
}

// InterviewService defines business operations for interview management.
type InterviewService interface {
	CreateInterview(ctx context.Context, input CreateInterviewInput, createdBy string) (*InterviewDTO, error)
	GetInterview(ctx context.Context, id string) (*InterviewDTO, error)
	UpdateInterview(ctx context.Context, id string, input UpdateInterviewInput, actorID string) (*InterviewDTO, error)
	RescheduleInterview(ctx context.Context, id string, input RescheduleInterviewInput, actorID string) (*InterviewDTO, error)
	CancelInterview(ctx context.Context, id string, input CancelInterviewInput, actorID string) (*InterviewDTO, error)
	CompleteInterview(ctx context.Context, id string, input CompleteInterviewInput, actorID string) (*InterviewDTO, error)
	ListInterviews(ctx context.Context, filter InterviewFilterInput) (*InterviewListDTO, error)
	GetCandidateInterviews(ctx context.Context, candidateID, jobID string) ([]InterviewDTO, error)
	GetCandidateNextInterview(ctx context.Context, candidateID, jobID string) (*InterviewDTO, error)
	ListUsers(ctx context.Context) ([]model.UserResponse, error)
	GenerateICS(ctx context.Context, id string) (filename string, content []byte, err error)
	SetNotificationService(notificationService InterviewNotificationService)
	SetCalendarIntegrationService(calendarService CalendarIntegrationService)
}

type interviewService struct {
	interviewRepo       repository.InterviewRepository
	jobCandidateRepo    repository.JobCandidateRepository
	userRepo            repository.UserRepository
	notificationService InterviewNotificationService
	calendarService     CalendarIntegrationService
}

// NewInterviewService constructs a new InterviewService.
func NewInterviewService(
	interviewRepo repository.InterviewRepository,
	jobCandidateRepo repository.JobCandidateRepository,
	userRepo repository.UserRepository,
	notificationServices ...InterviewNotificationService,
) InterviewService {
	s := &interviewService{
		interviewRepo:    interviewRepo,
		jobCandidateRepo: jobCandidateRepo,
		userRepo:         userRepo,
	}
	if len(notificationServices) > 0 {
		s.notificationService = notificationServices[0]
	}
	return s
}

func (s *interviewService) SetNotificationService(ns InterviewNotificationService) {
	s.notificationService = ns
}

func (s *interviewService) SetCalendarIntegrationService(cs CalendarIntegrationService) {
	s.calendarService = cs
}

func (s *interviewService) CreateInterview(ctx context.Context, input CreateInterviewInput, createdBy string) (*InterviewDTO, error) {
	if _, err := uuid.Parse(input.JobID); err != nil {
		return nil, ErrInvalidUUID
	}
	if _, err := uuid.Parse(input.CandidateID); err != nil {
		return nil, ErrInvalidUUID
	}

	// 1. Candidate Association Guard
	associated, err := s.jobCandidateRepo.IsJobCandidateAssociated(ctx, input.JobID, input.CandidateID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify candidate job association: %w", err)
	}
	if !associated {
		return nil, ErrCandidateNotAssociatedWithJob
	}

	// 2. Validate Stage & Type
	if !input.Stage.IsValid() {
		return nil, ErrInvalidInterviewStage
	}
	if !input.InterviewType.IsValid() {
		return nil, ErrInvalidInterviewType
	}

	// 3. Validate Date & Time
	if input.ScheduledEnd.Before(input.ScheduledStart) || input.ScheduledEnd.Equal(input.ScheduledStart) {
		return nil, ErrInvalidInterviewTime
	}
	if input.ScheduledEnd.Sub(input.ScheduledStart) > 8*time.Hour {
		return nil, ErrInterviewDurationTooLong
	}

	// 4. Validate Modality Fields
	var meetingURL *string
	if input.MeetingURL != nil {
		trimmed := strings.TrimSpace(*input.MeetingURL)
		if trimmed != "" {
			meetingURL = &trimmed
		}
	}
	var location *string
	if input.Location != nil {
		trimmed := strings.TrimSpace(*input.Location)
		if trimmed != "" {
			location = &trimmed
		}
	}

	if input.InterviewType == model.InterviewTypeOnline && (meetingURL == nil || *meetingURL == "") {
		return nil, ErrMissingMeetingURL
	}
	if input.InterviewType == model.InterviewTypeOnsite && (location == nil || *location == "") {
		return nil, ErrMissingLocation
	}

	// 5. Validate Interviewers
	interviewerIDs := deduplicateStrings(input.InterviewerIDs)
	if len(interviewerIDs) == 0 {
		return nil, ErrNoInterviewers
	}

	// Verify all interviewers exist
	for _, uid := range interviewerIDs {
		if _, err := uuid.Parse(uid); err != nil {
			return nil, fmt.Errorf("invalid interviewer user ID format: %s", uid)
		}
		if _, err := s.userRepo.FindByID(ctx, uid); err != nil {
			return nil, fmt.Errorf("interviewer user not found: %s", uid)
		}
	}

	// 6. Conflict Detection
	conflictName, err := s.interviewRepo.CheckInterviewerConflict(ctx, interviewerIDs, input.ScheduledStart, input.ScheduledEnd, "")
	if err != nil {
		return nil, fmt.Errorf("failed checking interviewer conflicts: %w", err)
	}
	if conflictName != "" {
		return nil, fmt.Errorf("%w: %s already has an interview scheduled during this time", ErrInterviewerConflict, conflictName)
	}

	// 7. Timezone fallback
	tz := strings.TrimSpace(input.Timezone)
	if tz == "" {
		tz = "Asia/Jakarta"
	}

	interview := &model.Interview{
		ID:             uuid.New().String(),
		JobID:          input.JobID,
		CandidateID:    input.CandidateID,
		Title:          strings.TrimSpace(input.Title),
		Stage:          input.Stage,
		InterviewType:  input.InterviewType,
		ScheduledStart: input.ScheduledStart.UTC(),
		ScheduledEnd:   input.ScheduledEnd.UTC(),
		Timezone:       tz,
		MeetingURL:     meetingURL,
		Location:       location,
		Notes:          input.Notes,
		Status:         model.InterviewStatusScheduled,
		CreatedBy:      createdBy,
		CreatedAt:      time.Now().UTC(),
		UpdatedAt:      time.Now().UTC(),
	}

	// 8. Structured Audit Log
	meta, _ := json.Marshal(map[string]interface{}{
		"interview_id":   interview.ID,
		"title":          interview.Title,
		"stage":          interview.Stage,
		"interview_type": interview.InterviewType,
		"start":          interview.ScheduledStart.Format(time.RFC3339),
		"end":            interview.ScheduledEnd.Format(time.RFC3339),
	})

	audit := &model.AuditLog{
		Action:      model.AuditActionInterviewCreated,
		ActorID:     createdBy,
		JobID:       &input.JobID,
		CandidateID: &input.CandidateID,
		Metadata:    string(meta),
		CreatedAt:   time.Now().UTC(),
	}

	if err := s.interviewRepo.Create(ctx, interview, interviewerIDs, audit); err != nil {
		return nil, fmt.Errorf("failed to save interview: %w", err)
	}

	// 9. Process email notifications and reminders if notification service is wired
	if s.notificationService != nil {
		_ = s.notificationService.ScheduleReminders(ctx, interview, interviewerIDs)
		if input.SendCandidateInvitation || input.SendInterviewerInvitation {
			_, _ = s.notificationService.TriggerInvitations(
				ctx,
				interview.ID,
				input.SendCandidateInvitation,
				input.SendInterviewerInvitation,
				createdBy,
			)
		}
	}

	return s.GetInterview(ctx, interview.ID)
}

func (s *interviewService) GetInterview(ctx context.Context, id string) (*InterviewDTO, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrInvalidUUID
	}
	interview, err := s.interviewRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toInterviewDTO(interview), nil
}

func (s *interviewService) UpdateInterview(ctx context.Context, id string, input UpdateInterviewInput, actorID string) (*InterviewDTO, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrInvalidUUID
	}

	interview, err := s.interviewRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if interview.Status != model.InterviewStatusScheduled {
		return nil, fmt.Errorf("%w: cannot edit interview in %s status", ErrInvalidStatusTransition, interview.Status)
	}

	if !input.Stage.IsValid() {
		return nil, ErrInvalidInterviewStage
	}
	if !input.InterviewType.IsValid() {
		return nil, ErrInvalidInterviewType
	}

	if input.ScheduledEnd.Before(input.ScheduledStart) || input.ScheduledEnd.Equal(input.ScheduledStart) {
		return nil, ErrInvalidInterviewTime
	}
	if input.ScheduledEnd.Sub(input.ScheduledStart) > 8*time.Hour {
		return nil, ErrInterviewDurationTooLong
	}

	var meetingURL *string
	if input.MeetingURL != nil {
		trimmed := strings.TrimSpace(*input.MeetingURL)
		if trimmed != "" {
			meetingURL = &trimmed
		}
	}
	var location *string
	if input.Location != nil {
		trimmed := strings.TrimSpace(*input.Location)
		if trimmed != "" {
			location = &trimmed
		}
	}

	if input.InterviewType == model.InterviewTypeOnline && (meetingURL == nil || *meetingURL == "") {
		return nil, ErrMissingMeetingURL
	}
	if input.InterviewType == model.InterviewTypeOnsite && (location == nil || *location == "") {
		return nil, ErrMissingLocation
	}

	interviewerIDs := deduplicateStrings(input.InterviewerIDs)
	if len(interviewerIDs) == 0 {
		return nil, ErrNoInterviewers
	}

	// Verify interviewers
	for _, uid := range interviewerIDs {
		if _, err := uuid.Parse(uid); err != nil {
			return nil, fmt.Errorf("invalid interviewer user ID format: %s", uid)
		}
		if _, err := s.userRepo.FindByID(ctx, uid); err != nil {
			return nil, fmt.Errorf("interviewer user not found: %s", uid)
		}
	}

	// Conflict detection (exclude current interview)
	conflictName, err := s.interviewRepo.CheckInterviewerConflict(ctx, interviewerIDs, input.ScheduledStart, input.ScheduledEnd, id)
	if err != nil {
		return nil, fmt.Errorf("failed checking interviewer conflicts: %w", err)
	}
	if conflictName != "" {
		return nil, fmt.Errorf("%w: %s already has an interview scheduled during this time", ErrInterviewerConflict, conflictName)
	}

	tz := strings.TrimSpace(input.Timezone)
	if tz == "" {
		tz = interview.Timezone
	}

	interview.Title = strings.TrimSpace(input.Title)
	interview.Stage = input.Stage
	interview.InterviewType = input.InterviewType
	interview.ScheduledStart = input.ScheduledStart.UTC()
	interview.ScheduledEnd = input.ScheduledEnd.UTC()
	interview.Timezone = tz
	interview.MeetingURL = meetingURL
	interview.Location = location
	interview.Notes = input.Notes
	interview.UpdatedAt = time.Now().UTC()

	meta, _ := json.Marshal(map[string]interface{}{
		"interview_id": id,
		"action":       "details_updated",
		"title":        interview.Title,
	})

	audit := &model.AuditLog{
		Action:      model.AuditActionInterviewUpdated,
		ActorID:     actorID,
		JobID:       &interview.JobID,
		CandidateID: &interview.CandidateID,
		Metadata:    string(meta),
		CreatedAt:   time.Now().UTC(),
	}

	if err := s.interviewRepo.Update(ctx, interview, interviewerIDs, audit); err != nil {
		return nil, fmt.Errorf("failed to update interview: %w", err)
	}

	return s.GetInterview(ctx, id)
}

func (s *interviewService) RescheduleInterview(ctx context.Context, id string, input RescheduleInterviewInput, actorID string) (*InterviewDTO, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrInvalidUUID
	}

	interview, err := s.interviewRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if interview.Status != model.InterviewStatusScheduled {
		return nil, fmt.Errorf("%w: cannot reschedule interview in %s status", ErrInvalidStatusTransition, interview.Status)
	}

	if input.ScheduledEnd.Before(input.ScheduledStart) || input.ScheduledEnd.Equal(input.ScheduledStart) {
		return nil, ErrInvalidInterviewTime
	}
	if input.ScheduledEnd.Sub(input.ScheduledStart) > 8*time.Hour {
		return nil, ErrInterviewDurationTooLong
	}

	// Extract existing interviewer IDs
	var interviewerIDs []string
	for _, ii := range interview.Interviewers {
		interviewerIDs = append(interviewerIDs, ii.UserID)
	}

	// Check conflicts for existing interviewers
	conflictName, err := s.interviewRepo.CheckInterviewerConflict(ctx, interviewerIDs, input.ScheduledStart, input.ScheduledEnd, id)
	if err != nil {
		return nil, fmt.Errorf("failed checking interviewer conflicts: %w", err)
	}
	if conflictName != "" {
		return nil, fmt.Errorf("%w: %s already has an interview scheduled during this time", ErrInterviewerConflict, conflictName)
	}

	oldStart := interview.ScheduledStart
	oldEnd := interview.ScheduledEnd

	interview.ScheduledStart = input.ScheduledStart.UTC()
	interview.ScheduledEnd = input.ScheduledEnd.UTC()
	interview.UpdatedAt = time.Now().UTC()

	metaMap := map[string]interface{}{
		"interview_id": id,
		"old_start":    oldStart.Format(time.RFC3339),
		"old_end":      oldEnd.Format(time.RFC3339),
		"new_start":    interview.ScheduledStart.Format(time.RFC3339),
		"new_end":      interview.ScheduledEnd.Format(time.RFC3339),
	}
	if input.Reason != nil && strings.TrimSpace(*input.Reason) != "" {
		metaMap["reason"] = strings.TrimSpace(*input.Reason)
	}
	meta, _ := json.Marshal(metaMap)

	audit := &model.AuditLog{
		Action:      model.AuditActionInterviewRescheduled,
		ActorID:     actorID,
		JobID:       &interview.JobID,
		CandidateID: &interview.CandidateID,
		Metadata:    string(meta),
		CreatedAt:   time.Now().UTC(),
	}

	if err := s.interviewRepo.UpdateStatus(ctx, interview, audit); err != nil {
		return nil, fmt.Errorf("failed to save rescheduled interview: %w", err)
	}

	if s.notificationService != nil {
		_ = s.notificationService.InvalidateReminders(ctx, id, "Interview was rescheduled")
		var interviewerIDs []string
		for _, ii := range interview.Interviewers {
			interviewerIDs = append(interviewerIDs, ii.UserID)
		}
		_ = s.notificationService.ScheduleReminders(ctx, interview, interviewerIDs)
	}

	if s.calendarService != nil {
		go func(itw *model.Interview, actID string) {
			bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = s.calendarService.HandleInterviewRescheduled(bgCtx, itw, actID)
		}(interview, actorID)
	}

	return s.GetInterview(ctx, id)
}

func (s *interviewService) CancelInterview(ctx context.Context, id string, input CancelInterviewInput, actorID string) (*InterviewDTO, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrInvalidUUID
	}

	interview, err := s.interviewRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if interview.Status != model.InterviewStatusScheduled {
		return nil, fmt.Errorf("%w: cannot cancel interview in %s status", ErrInvalidStatusTransition, interview.Status)
	}

	reason := strings.TrimSpace(input.Reason)
	if reason == "" {
		return nil, ErrCancellationReasonRequired
	}
	if len(reason) > 1000 {
		return nil, ErrCancellationReasonTooLong
	}

	now := time.Now().UTC()
	interview.Status = model.InterviewStatusCancelled
	interview.CancelledAt = &now
	interview.CancelledBy = &actorID
	interview.CancellationReason = &reason
	interview.UpdatedAt = now

	meta, _ := json.Marshal(map[string]interface{}{
		"interview_id":        id,
		"cancellation_reason": reason,
	})

	audit := &model.AuditLog{
		Action:      model.AuditActionInterviewCancelled,
		ActorID:     actorID,
		JobID:       &interview.JobID,
		CandidateID: &interview.CandidateID,
		Metadata:    string(meta),
		CreatedAt:   now,
	}

	if err := s.interviewRepo.UpdateStatus(ctx, interview, audit); err != nil {
		return nil, fmt.Errorf("failed to cancel interview: %w", err)
	}

	if s.notificationService != nil {
		_ = s.notificationService.InvalidateReminders(ctx, id, "Interview was cancelled")
	}

	if s.calendarService != nil {
		go func(itw *model.Interview, actID string) {
			bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			_ = s.calendarService.HandleInterviewCancelled(bgCtx, itw, actID)
		}(interview, actorID)
	}

	return s.GetInterview(ctx, id)
}

func (s *interviewService) CompleteInterview(ctx context.Context, id string, input CompleteInterviewInput, actorID string) (*InterviewDTO, error) {
	if _, err := uuid.Parse(id); err != nil {
		return nil, ErrInvalidUUID
	}

	interview, err := s.interviewRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if interview.Status != model.InterviewStatusScheduled {
		return nil, fmt.Errorf("%w: cannot complete interview in %s status", ErrInvalidStatusTransition, interview.Status)
	}

	if !input.Result.IsValid() {
		return nil, ErrInvalidInterviewResult
	}

	var feedback *string
	if input.Feedback != nil {
		trimmed := strings.TrimSpace(*input.Feedback)
		if len(trimmed) > 5000 {
			return nil, ErrFeedbackTooLong
		}
		if trimmed != "" {
			feedback = &trimmed
		}
	}

	now := time.Now().UTC()
	interview.Status = model.InterviewStatusCompleted
	interview.CompletedAt = &now
	interview.CompletedBy = &actorID
	interview.Result = &input.Result
	interview.Feedback = feedback
	interview.UpdatedAt = now

	meta, _ := json.Marshal(map[string]interface{}{
		"interview_id": id,
		"result":       input.Result,
	})

	audit := &model.AuditLog{
		Action:      model.AuditActionInterviewCompleted,
		ActorID:     actorID,
		JobID:       &interview.JobID,
		CandidateID: &interview.CandidateID,
		Metadata:    string(meta),
		CreatedAt:   now,
	}

	if err := s.interviewRepo.UpdateStatus(ctx, interview, audit); err != nil {
		return nil, fmt.Errorf("failed to complete interview: %w", err)
	}

	if s.notificationService != nil {
		_ = s.notificationService.InvalidateReminders(ctx, id, "Interview was completed")
	}

	return s.GetInterview(ctx, id)
}

func (s *interviewService) ListInterviews(ctx context.Context, filter InterviewFilterInput) (*InterviewListDTO, error) {
	if filter.From != nil && filter.To != nil {
		if filter.To.Before(*filter.From) {
			return nil, ErrInvalidDateRange
		}
		if filter.To.Sub(*filter.From) > 366*24*time.Hour {
			return nil, ErrDateRangeTooLarge
		}
	}

	repoFilter := repository.InterviewFilter{
		JobID:         filter.JobID,
		CandidateID:   filter.CandidateID,
		Status:        filter.Status,
		Stage:         filter.Stage,
		InterviewType: filter.InterviewType,
		From:          filter.From,
		To:            filter.To,
		Page:          filter.Page,
		Limit:         filter.Limit,
		Sort:          filter.Sort,
		Order:         filter.Order,
	}

	result, err := s.interviewRepo.List(ctx, repoFilter)
	if err != nil {
		return nil, err
	}

	var dtos []InterviewDTO
	for i := range result.Items {
		dtos = append(dtos, *toInterviewDTO(&result.Items[i]))
	}

	return &InterviewListDTO{
		Items:      dtos,
		Total:      result.Total,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	}, nil
}

func (s *interviewService) GetCandidateInterviews(ctx context.Context, candidateID, jobID string) ([]InterviewDTO, error) {
	if _, err := uuid.Parse(candidateID); err != nil {
		return nil, ErrInvalidUUID
	}
	if jobID != "" {
		if _, err := uuid.Parse(jobID); err != nil {
			return nil, ErrInvalidUUID
		}
	}

	items, err := s.interviewRepo.ListByCandidateAndJob(ctx, candidateID, jobID)
	if err != nil {
		return nil, err
	}

	var dtos []InterviewDTO
	for i := range items {
		dtos = append(dtos, *toInterviewDTO(&items[i]))
	}
	return dtos, nil
}

func (s *interviewService) GetCandidateNextInterview(ctx context.Context, candidateID, jobID string) (*InterviewDTO, error) {
	if _, err := uuid.Parse(candidateID); err != nil {
		return nil, ErrInvalidUUID
	}
	if jobID != "" {
		if _, err := uuid.Parse(jobID); err != nil {
			return nil, ErrInvalidUUID
		}
	}

	item, err := s.interviewRepo.GetCandidateNextInterview(ctx, candidateID, jobID)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}
	return toInterviewDTO(item), nil
}

func (s *interviewService) ListUsers(ctx context.Context) ([]model.UserResponse, error) {
	users, err := s.userRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	var res []model.UserResponse
	for _, u := range users {
		res = append(res, u.ToResponse())
	}
	return res, nil
}

func toInterviewDTO(item *model.Interview) *InterviewDTO {
	if item == nil {
		return nil
	}

	dto := &InterviewDTO{
		ID:                 item.ID,
		JobID:              item.JobID,
		CandidateID:        item.CandidateID,
		Title:              item.Title,
		Stage:              item.Stage,
		StageLabel:         item.Stage.DisplayLabel(),
		InterviewType:      item.InterviewType,
		ScheduledStart:     item.ScheduledStart,
		ScheduledEnd:       item.ScheduledEnd,
		Timezone:           item.Timezone,
		MeetingURL:         item.MeetingURL,
		Location:           item.Location,
		Notes:              item.Notes,
		Status:             item.Status,
		CreatedBy:          item.CreatedBy,
		CompletedAt:        item.CompletedAt,
		CompletedBy:        item.CompletedBy,
		Result:             item.Result,
		Feedback:           item.Feedback,
		CancelledAt:        item.CancelledAt,
		CancelledBy:        item.CancelledBy,
		CancellationReason: item.CancellationReason,
		CreatedAt:          item.CreatedAt,
		UpdatedAt:          item.UpdatedAt,
		Interviewers:       make([]InterviewerDTO, 0),
	}

	if item.Creator != nil {
		dto.CreatorName = item.Creator.Name
	}
	if item.CompletedUser != nil {
		dto.CompletedName = item.CompletedUser.Name
	}
	if item.CancelledUser != nil {
		dto.CancelledName = item.CancelledUser.Name
	}

	if item.Job != nil {
		dto.Job = &InterviewJobDTO{
			ID:             item.Job.ID,
			Code:           item.Job.Code,
			Title:          item.Job.Title,
			Department:     item.Job.Department,
			Location:       item.Job.Location,
			EmploymentType: item.Job.EmploymentType,
			Status:         item.Job.Status,
		}
	}

	if item.Candidate != nil {
		dto.Candidate = &InterviewCandidateDTO{
			ID:       item.Candidate.ID,
			FullName: item.Candidate.FullName,
			Email:    item.Candidate.Email,
			Phone:    item.Candidate.Phone,
			Location: item.Candidate.Location,
			Headline: item.Candidate.Headline,
		}
	}

	for _, ii := range item.Interviewers {
		if ii.User != nil {
			dto.Interviewers = append(dto.Interviewers, InterviewerDTO{
				ID:    ii.User.ID,
				Name:  ii.User.Name,
				Email: ii.User.Email,
				Role:  string(ii.User.Role),
			})
		}
	}

	return dto
}

func deduplicateStrings(input []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range input {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" && !seen[trimmed] {
			seen[trimmed] = true
			result = append(result, trimmed)
		}
	}
	return result
}

func (s *interviewService) GenerateICS(ctx context.Context, id string) (string, []byte, error) {
	if _, err := uuid.Parse(id); err != nil {
		return "", nil, ErrInvalidUUID
	}

	interview, err := s.interviewRepo.GetByID(ctx, id)
	if err != nil {
		return "", nil, err
	}

	candidateName := "Candidate"
	if interview.Candidate != nil && strings.TrimSpace(interview.Candidate.FullName) != "" {
		candidateName = strings.TrimSpace(interview.Candidate.FullName)
	}

	jobTitle := ""
	jobCode := ""
	if interview.Job != nil {
		jobTitle = strings.TrimSpace(interview.Job.Title)
		jobCode = strings.TrimSpace(interview.Job.Code)
	}

	// 1. Summary: <StageLabel> — <CandidateName> — <JobTitle>
	summary := fmt.Sprintf("%s — %s", interview.Stage.DisplayLabel(), candidateName)
	if jobTitle != "" {
		summary = fmt.Sprintf("%s — %s", summary, jobTitle)
	}

	// 2. Location & Meeting URL
	location := "Online"
	var meetingURL string
	switch interview.InterviewType {
	case model.InterviewTypeOnline:
		location = "Online"
		if interview.MeetingURL != nil {
			meetingURL = strings.TrimSpace(*interview.MeetingURL)
		}
	case model.InterviewTypeOnsite:
		if interview.Location != nil && strings.TrimSpace(*interview.Location) != "" {
			location = strings.TrimSpace(*interview.Location)
		} else {
			location = "Onsite"
		}
	case model.InterviewTypePhone:
		location = "Phone Interview"
	}

	// 3. Description
	var descLines []string
	descLines = append(descLines, fmt.Sprintf("Candidate: %s", candidateName))
	if jobTitle != "" {
		if jobCode != "" {
			descLines = append(descLines, fmt.Sprintf("Job: %s (%s)", jobTitle, jobCode))
		} else {
			descLines = append(descLines, fmt.Sprintf("Job: %s", jobTitle))
		}
	}
	descLines = append(descLines, fmt.Sprintf("Stage: %s", interview.Stage.DisplayLabel()))
	descLines = append(descLines, fmt.Sprintf("Type: %s", string(interview.InterviewType)))
	descLines = append(descLines, fmt.Sprintf("Status: %s", string(interview.Status)))

	var interviewerNames []string
	for _, iv := range interview.Interviewers {
		if iv.User != nil && strings.TrimSpace(iv.User.Name) != "" {
			interviewerNames = append(interviewerNames, iv.User.Name)
		}
	}
	if len(interviewerNames) > 0 {
		descLines = append(descLines, fmt.Sprintf("Interviewers: %s", strings.Join(interviewerNames, ", ")))
	}

	if interview.Notes != nil && strings.TrimSpace(*interview.Notes) != "" {
		descLines = append(descLines, fmt.Sprintf("Notes: %s", strings.TrimSpace(*interview.Notes)))
	}

	description := strings.Join(descLines, "\n")

	// 4. Status mapping
	statusStr := "CONFIRMED"
	if interview.Status == model.InterviewStatusCancelled {
		statusStr = "CANCELLED"
	} else if interview.Status == model.InterviewStatusCompleted {
		statusStr = "COMPLETED"
	}

	// 5. Timestamps in UTC
	dtStamp := time.Now().UTC().Format("20060102T150405Z")
	dtStart := interview.ScheduledStart.UTC().Format("20060102T150405Z")
	dtEnd := interview.ScheduledEnd.UTC().Format("20060102T150405Z")

	// 6. Build ICS document with RFC 5545 compliance
	var b strings.Builder
	writeICSLine(&b, "BEGIN:VCALENDAR")
	writeICSLine(&b, "VERSION:2.0")
	writeICSLine(&b, "PRODID:-//HireScope//Interview Calendar//EN")
	writeICSLine(&b, "CALSCALE:GREGORIAN")
	writeICSLine(&b, "METHOD:PUBLISH")
	writeICSLine(&b, "BEGIN:VEVENT")
	writeICSLine(&b, fmt.Sprintf("UID:%s@hirescope.local", interview.ID))
	writeICSLine(&b, fmt.Sprintf("DTSTAMP:%s", dtStamp))
	writeICSLine(&b, fmt.Sprintf("DTSTART:%s", dtStart))
	writeICSLine(&b, fmt.Sprintf("DTEND:%s", dtEnd))
	writeICSLine(&b, fmt.Sprintf("SUMMARY:%s", escapeICSText(summary)))
	writeICSLine(&b, fmt.Sprintf("DESCRIPTION:%s", escapeICSText(description)))
	writeICSLine(&b, fmt.Sprintf("LOCATION:%s", escapeICSText(location)))
	if meetingURL != "" {
		cleanURL := strings.ReplaceAll(strings.ReplaceAll(meetingURL, "\r", ""), "\n", "")
		writeICSLine(&b, fmt.Sprintf("URL:%s", cleanURL))
	}
	writeICSLine(&b, fmt.Sprintf("STATUS:%s", statusStr))
	writeICSLine(&b, "END:VEVENT")
	writeICSLine(&b, "END:VCALENDAR")

	candidateSlug := slugify(candidateName)
	if candidateSlug == "" {
		candidateSlug = "candidate"
	}
	dateStr := interview.ScheduledStart.UTC().Format("20060102")
	filename := fmt.Sprintf("hirescope-interview-%s-%s.ics", candidateSlug, dateStr)

	return filename, []byte(b.String()), nil
}

func escapeICSText(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `;`, `\;`)
	s = strings.ReplaceAll(s, `,`, `\,`)
	s = strings.ReplaceAll(s, "\r\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\n`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return s
}

func slugify(s string) string {
	s = strings.ToLower(s)
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	s = reg.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func writeICSLine(b *strings.Builder, line string) {
	const maxOctets = 75
	if len(line) <= maxOctets {
		b.WriteString(line)
		b.WriteString("\r\n")
		return
	}

	remaining := line
	first := true
	for len(remaining) > 0 {
		limit := maxOctets
		if !first {
			limit = maxOctets - 1
		}
		if len(remaining) <= limit {
			if !first {
				b.WriteString(" ")
			}
			b.WriteString(remaining)
			b.WriteString("\r\n")
			break
		}

		cut := limit
		for cut > 0 && !isUTF8Start(remaining[cut]) {
			cut--
		}
		if cut == 0 {
			cut = limit
		}

		chunk := remaining[:cut]
		remaining = remaining[cut:]

		if !first {
			b.WriteString(" ")
		}
		b.WriteString(chunk)
		b.WriteString("\r\n")
		first = false
	}
}

func isUTF8Start(b byte) bool {
	return (b & 0xC0) != 0x80
}
