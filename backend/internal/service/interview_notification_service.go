package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"hirescope/backend/internal/config"
	"hirescope/backend/internal/email"
	"hirescope/backend/internal/model"
	"hirescope/backend/internal/repository"
)

var (
	ErrOnlyFailedDeliveriesCanBeRetried = errors.New("only failed email deliveries can be retried")
	ErrMaxRetryAttemptsReached          = errors.New("maximum retry limit (3 attempts) reached for this email delivery")
	ErrCandidateHasNoEmail              = errors.New("candidate has no email address configured")
	ErrInterviewerHasNoEmail            = errors.New("interviewer has no email address configured")
)

// InterviewNotificationService manages interview transactional emails and background reminders.
type InterviewNotificationService interface {
	TriggerInvitations(ctx context.Context, interviewID string, sendCandidate, sendInterviewers bool, actorID string) ([]*model.InterviewEmailDelivery, error)
	RetryDelivery(ctx context.Context, deliveryID string, actorID string) (*model.InterviewEmailDelivery, error)
	ListDeliveries(ctx context.Context, interviewID string) ([]*model.InterviewEmailDelivery, error)
	ScheduleReminders(ctx context.Context, interview *model.Interview, interviewerIDs []string) error
	InvalidateReminders(ctx context.Context, interviewID string, reason string) error
	ProcessDueReminders(ctx context.Context) (int, error)
}

type interviewNotificationService struct {
	deliveryRepo  repository.InterviewEmailDeliveryRepository
	interviewRepo repository.InterviewRepository
	userRepo      repository.UserRepository
	auditRepo     repository.AuditLogRepository
	emailProvider email.EmailProvider
	cfg           *config.Config
}

// NewInterviewNotificationService creates a new InterviewNotificationService instance.
func NewInterviewNotificationService(
	deliveryRepo repository.InterviewEmailDeliveryRepository,
	interviewRepo repository.InterviewRepository,
	userRepo repository.UserRepository,
	auditRepo repository.AuditLogRepository,
	emailProvider email.EmailProvider,
	cfg *config.Config,
) InterviewNotificationService {
	return &interviewNotificationService{
		deliveryRepo:  deliveryRepo,
		interviewRepo: interviewRepo,
		userRepo:      userRepo,
		auditRepo:     auditRepo,
		emailProvider: emailProvider,
		cfg:           cfg,
	}
}

func (s *interviewNotificationService) ListDeliveries(ctx context.Context, interviewID string) ([]*model.InterviewEmailDelivery, error) {
	return s.deliveryRepo.ListByInterviewID(ctx, interviewID)
}

func (s *interviewNotificationService) InvalidateReminders(ctx context.Context, interviewID string, reason string) error {
	return s.deliveryRepo.InvalidatePendingReminders(ctx, interviewID, reason)
}

func (s *interviewNotificationService) ScheduleReminders(ctx context.Context, interview *model.Interview, interviewerIDs []string) error {
	if interview == nil || interview.Status != model.InterviewStatusScheduled {
		return nil
	}

	reminderHours := s.cfg.InterviewReminderHours
	if reminderHours <= 0 {
		reminderHours = 24
	}

	scheduledFor := interview.ScheduledStart.Add(-time.Duration(reminderHours) * time.Hour)

	// Ensure full interview with candidate is loaded if Candidate is nil
	if interview.Candidate == nil && interview.ID != "" {
		if full, err := s.interviewRepo.GetByID(ctx, interview.ID); err == nil && full != nil {
			interview = full
		}
	}

	// 1. Candidate reminder record
	candidateID := interview.CandidateID
	candidateEmail := ""
	candidateName := "Candidate"
	if interview.Candidate != nil {
		candidateEmail = interview.Candidate.Email
		candidateName = interview.Candidate.FullName
	}

	if candidateEmail != "" {
		candKey := fmt.Sprintf("interview:%s:reminder:candidate:%d", interview.ID, scheduledFor.Unix())
		existing, err := s.deliveryRepo.GetByIdempotencyKey(ctx, candKey)
		if err != nil && !errors.Is(err, repository.ErrDeliveryNotFound) {
			return err
		}
		if existing == nil {
			candDelivery := &model.InterviewEmailDelivery{
				InterviewID:          interview.ID,
				RecipientType:        model.RecipientTypeCandidate,
				RecipientCandidateID: &candidateID,
				RecipientEmail:       candidateEmail,
				RecipientName:        candidateName,
				EmailType:            model.EmailTypeInterviewReminder,
				Status:               model.DeliveryStatusPending,
				Provider:             s.cfg.EmailProvider,
				IdempotencyKey:       candKey,
				ScheduledFor:         &scheduledFor,
				CreatedAt:            time.Now().UTC(),
				UpdatedAt:            time.Now().UTC(),
			}
			_ = s.deliveryRepo.Create(ctx, candDelivery)
		}
	}

	// 2. Interviewer reminder records
	for _, uid := range interviewerIDs {
		u, err := s.userRepo.FindByID(ctx, uid)
		if err != nil || u == nil || u.Email == "" {
			continue
		}
		userID := u.ID
		intKey := fmt.Sprintf("interview:%s:reminder:interviewer:%s:%d", interview.ID, u.ID, scheduledFor.Unix())
		existing, err := s.deliveryRepo.GetByIdempotencyKey(ctx, intKey)
		if err != nil && !errors.Is(err, repository.ErrDeliveryNotFound) {
			return err
		}
		if existing == nil {
			intDelivery := &model.InterviewEmailDelivery{
				InterviewID:     interview.ID,
				RecipientType:   model.RecipientTypeInterviewer,
				RecipientUserID: &userID,
				RecipientEmail:  u.Email,
				RecipientName:   u.Name,
				EmailType:       model.EmailTypeInterviewReminder,
				Status:          model.DeliveryStatusPending,
				Provider:        s.cfg.EmailProvider,
				IdempotencyKey:  intKey,
				ScheduledFor:    &scheduledFor,
				CreatedAt:       time.Now().UTC(),
				UpdatedAt:       time.Now().UTC(),
			}
			_ = s.deliveryRepo.Create(ctx, intDelivery)
		}
	}

	return nil
}

func (s *interviewNotificationService) TriggerInvitations(ctx context.Context, interviewID string, sendCandidate, sendInterviewers bool, actorID string) ([]*model.InterviewEmailDelivery, error) {
	interview, err := s.interviewRepo.GetByID(ctx, interviewID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch interview for invitation: %w", err)
	}

	var results []*model.InterviewEmailDelivery

	var interviewerNames []string
	for _, ii := range interview.Interviewers {
		if ii.User != nil {
			interviewerNames = append(interviewerNames, ii.User.Name)
		}
	}

	jobTitle := "Job Position"
	jobCode := ""
	if interview.Job != nil {
		jobTitle = interview.Job.Title
		jobCode = interview.Job.Code
	}

	candidateName := "Candidate"
	candidateEmail := ""
	if interview.Candidate != nil {
		candidateName = interview.Candidate.FullName
		candidateEmail = interview.Candidate.Email
	}

	meetingURL := ""
	if interview.MeetingURL != nil {
		meetingURL = *interview.MeetingURL
	}
	location := ""
	if interview.Location != nil {
		location = *interview.Location
	}

	templateData := email.InvitationTemplateData{
		CandidateName:    candidateName,
		JobTitle:         jobTitle,
		JobCode:          jobCode,
		StageLabel:       interview.Stage.DisplayLabel(),
		InterviewType:    string(interview.InterviewType),
		ScheduledStart:   interview.ScheduledStart,
		ScheduledEnd:     interview.ScheduledEnd,
		MeetingURL:       meetingURL,
		Location:         location,
		InterviewerNames: interviewerNames,
		InterviewID:      interview.ID,
		CandidateID:      interview.CandidateID,
		BaseURL:          s.cfg.EmailBaseURL,
	}

	fromAddress := s.cfg.EmailFromAddress
	if fromAddress == "" {
		fromAddress = "onboarding@resend.dev"
	}
	fromHeader := fmt.Sprintf("%s <%s>", s.cfg.EmailFromName, fromAddress)

	// 1. Send Candidate Invitation
	if sendCandidate {
		candKey := fmt.Sprintf("interview:%s:invitation:candidate", interview.ID)
		delivery, err := s.deliveryRepo.GetByIdempotencyKey(ctx, candKey)
		if err != nil && !errors.Is(err, repository.ErrDeliveryNotFound) {
			return nil, err
		}

		if delivery == nil {
			candID := interview.CandidateID
			delivery = &model.InterviewEmailDelivery{
				InterviewID:          interview.ID,
				RecipientType:        model.RecipientTypeCandidate,
				RecipientCandidateID: &candID,
				RecipientEmail:       candidateEmail,
				RecipientName:        candidateName,
				EmailType:            model.EmailTypeInterviewInvitation,
				Status:               model.DeliveryStatusPending,
				Provider:             s.cfg.EmailProvider,
				IdempotencyKey:       candKey,
				AttemptCount:         0,
				CreatedAt:            time.Now().UTC(),
				UpdatedAt:            time.Now().UTC(),
			}
			if err := s.deliveryRepo.Create(ctx, delivery); err != nil {
				return nil, err
			}
		}

		// Only send if not already SENT
		if delivery.Status != model.DeliveryStatusSent {
			if candidateEmail == "" {
				delivery.Status = model.DeliveryStatusFailed
				errMsg := ErrCandidateHasNoEmail.Error()
				delivery.LastError = &errMsg
				_ = s.deliveryRepo.Update(ctx, delivery)
			} else {
				subj, htmlBody, textBody := email.BuildCandidateInvitation(templateData)
				msg := email.EmailMessage{
					From:           fromHeader,
					To:             candidateEmail,
					ReplyTo:        s.cfg.EmailReplyTo,
					Subject:        subj,
					HTML:           htmlBody,
					Text:           textBody,
					IdempotencyKey: candKey,
				}

				delivery.Status = model.DeliveryStatusSending
				delivery.AttemptCount++
				_ = s.deliveryRepo.Update(ctx, delivery)

				res, sendErr := s.emailProvider.Send(ctx, msg)
				now := time.Now().UTC()
				if sendErr != nil {
					if errors.Is(sendErr, email.ErrEmailDisabled) {
						delivery.Status = model.DeliveryStatusSkipped
						errMsg := sendErr.Error()
						delivery.LastError = &errMsg
					} else {
						delivery.Status = model.DeliveryStatusFailed
						errMsg := sendErr.Error()
						delivery.LastError = &errMsg
					}
				} else {
					delivery.Status = model.DeliveryStatusSent
					delivery.SentAt = &now
					delivery.LastError = nil
					if res != nil {
						delivery.ProviderMessageID = &res.ProviderMessageID
					}
				}
				_ = s.deliveryRepo.Update(ctx, delivery)

				// Record Audit Log
				meta, _ := json.Marshal(map[string]interface{}{
					"interview_id":    interview.ID,
					"recipient_type":  model.RecipientTypeCandidate,
					"recipient_email": candidateEmail,
					"delivery_status": delivery.Status,
					"delivery_id":     delivery.ID,
				})
				_ = s.auditRepo.Create(ctx, &model.AuditLog{
					Action:      model.AuditActionInterviewInvitationSent,
					ActorID:     actorID,
					JobID:       &interview.JobID,
					CandidateID: &interview.CandidateID,
					Metadata:    string(meta),
					CreatedAt:   time.Now().UTC(),
				})
			}
		}
		results = append(results, delivery)
	}

	// 2. Send Interviewer Invitations
	if sendInterviewers {
		for _, ii := range interview.Interviewers {
			if ii.User == nil || ii.User.Email == "" {
				continue
			}
			u := ii.User
			userID := u.ID
			intKey := fmt.Sprintf("interview:%s:invitation:interviewer:%s", interview.ID, u.ID)

			delivery, err := s.deliveryRepo.GetByIdempotencyKey(ctx, intKey)
			if err != nil && !errors.Is(err, repository.ErrDeliveryNotFound) {
				return nil, err
			}

			if delivery == nil {
				delivery = &model.InterviewEmailDelivery{
					InterviewID:     interview.ID,
					RecipientType:   model.RecipientTypeInterviewer,
					RecipientUserID: &userID,
					RecipientEmail:  u.Email,
					RecipientName:   u.Name,
					EmailType:       model.EmailTypeInterviewInvitation,
					Status:          model.DeliveryStatusPending,
					Provider:        s.cfg.EmailProvider,
					IdempotencyKey:  intKey,
					AttemptCount:    0,
					CreatedAt:       time.Now().UTC(),
					UpdatedAt:       time.Now().UTC(),
				}
				if err := s.deliveryRepo.Create(ctx, delivery); err != nil {
					return nil, err
				}
			}

			if delivery.Status != model.DeliveryStatusSent {
				subj, htmlBody, textBody := email.BuildInterviewerInvitation(u.Name, templateData)
				msg := email.EmailMessage{
					From:           fromHeader,
					To:             u.Email,
					ReplyTo:        s.cfg.EmailReplyTo,
					Subject:        subj,
					HTML:           htmlBody,
					Text:           textBody,
					IdempotencyKey: intKey,
				}

				delivery.Status = model.DeliveryStatusSending
				delivery.AttemptCount++
				_ = s.deliveryRepo.Update(ctx, delivery)

				res, sendErr := s.emailProvider.Send(ctx, msg)
				now := time.Now().UTC()
				if sendErr != nil {
					if errors.Is(sendErr, email.ErrEmailDisabled) {
						delivery.Status = model.DeliveryStatusSkipped
						errMsg := sendErr.Error()
						delivery.LastError = &errMsg
					} else {
						delivery.Status = model.DeliveryStatusFailed
						errMsg := sendErr.Error()
						delivery.LastError = &errMsg
					}
				} else {
					delivery.Status = model.DeliveryStatusSent
					delivery.SentAt = &now
					delivery.LastError = nil
					if res != nil {
						delivery.ProviderMessageID = &res.ProviderMessageID
					}
				}
				_ = s.deliveryRepo.Update(ctx, delivery)

				meta, _ := json.Marshal(map[string]interface{}{
					"interview_id":    interview.ID,
					"recipient_type":  model.RecipientTypeInterviewer,
					"recipient_user":  u.ID,
					"recipient_email": u.Email,
					"delivery_status": delivery.Status,
					"delivery_id":     delivery.ID,
				})
				_ = s.auditRepo.Create(ctx, &model.AuditLog{
					Action:      model.AuditActionInterviewInvitationSent,
					ActorID:     actorID,
					JobID:       &interview.JobID,
					CandidateID: &interview.CandidateID,
					Metadata:    string(meta),
					CreatedAt:   time.Now().UTC(),
				})
			}
			results = append(results, delivery)
		}
	}

	return results, nil
}

func (s *interviewNotificationService) RetryDelivery(ctx context.Context, deliveryID string, actorID string) (*model.InterviewEmailDelivery, error) {
	delivery, err := s.deliveryRepo.GetByID(ctx, deliveryID)
	if err != nil {
		return nil, err
	}

	if delivery.Status != model.DeliveryStatusFailed {
		return nil, ErrOnlyFailedDeliveriesCanBeRetried
	}

	if delivery.AttemptCount >= 3 {
		return nil, ErrMaxRetryAttemptsReached
	}

	interview, err := s.interviewRepo.GetByID(ctx, delivery.InterviewID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch interview for retry: %w", err)
	}

	var interviewerNames []string
	for _, ii := range interview.Interviewers {
		if ii.User != nil {
			interviewerNames = append(interviewerNames, ii.User.Name)
		}
	}

	jobTitle := "Job Position"
	jobCode := ""
	if interview.Job != nil {
		jobTitle = interview.Job.Title
		jobCode = interview.Job.Code
	}

	candidateName := "Candidate"
	if interview.Candidate != nil {
		candidateName = interview.Candidate.FullName
	}

	meetingURL := ""
	if interview.MeetingURL != nil {
		meetingURL = *interview.MeetingURL
	}
	location := ""
	if interview.Location != nil {
		location = *interview.Location
	}

	fromAddress := s.cfg.EmailFromAddress
	if fromAddress == "" {
		fromAddress = "onboarding@resend.dev"
	}
	fromHeader := fmt.Sprintf("%s <%s>", s.cfg.EmailFromName, fromAddress)

	var subj, htmlBody, textBody string
	if delivery.EmailType == model.EmailTypeInterviewInvitation {
		templateData := email.InvitationTemplateData{
			CandidateName:    candidateName,
			JobTitle:         jobTitle,
			JobCode:          jobCode,
			StageLabel:       interview.Stage.DisplayLabel(),
			InterviewType:    string(interview.InterviewType),
			ScheduledStart:   interview.ScheduledStart,
			ScheduledEnd:     interview.ScheduledEnd,
			MeetingURL:       meetingURL,
			Location:         location,
			InterviewerNames: interviewerNames,
			InterviewID:      interview.ID,
			CandidateID:      interview.CandidateID,
			BaseURL:          s.cfg.EmailBaseURL,
		}
		if delivery.RecipientType == model.RecipientTypeCandidate {
			subj, htmlBody, textBody = email.BuildCandidateInvitation(templateData)
		} else {
			subj, htmlBody, textBody = email.BuildInterviewerInvitation(delivery.RecipientName, templateData)
		}
	} else {
		// Reminder
		templateData := email.ReminderTemplateData{
			RecipientName:  delivery.RecipientName,
			CandidateName:  candidateName,
			JobTitle:       jobTitle,
			JobCode:        jobCode,
			StageLabel:     interview.Stage.DisplayLabel(),
			InterviewType:  string(interview.InterviewType),
			ScheduledStart: interview.ScheduledStart,
			ScheduledEnd:   interview.ScheduledEnd,
			MeetingURL:     meetingURL,
			Location:       location,
			InterviewID:    interview.ID,
			BaseURL:        s.cfg.EmailBaseURL,
		}
		subj, htmlBody, textBody = email.BuildReminder(templateData)
	}

	msg := email.EmailMessage{
		From:           fromHeader,
		To:             delivery.RecipientEmail,
		ReplyTo:        s.cfg.EmailReplyTo,
		Subject:        subj,
		HTML:           htmlBody,
		Text:           textBody,
		IdempotencyKey: fmt.Sprintf("%s:retry:%d", delivery.IdempotencyKey, delivery.AttemptCount+1),
	}

	delivery.Status = model.DeliveryStatusSending
	delivery.AttemptCount++
	_ = s.deliveryRepo.Update(ctx, delivery)

	res, sendErr := s.emailProvider.Send(ctx, msg)
	now := time.Now().UTC()
	if sendErr != nil {
		if errors.Is(sendErr, email.ErrEmailDisabled) {
			delivery.Status = model.DeliveryStatusSkipped
			errMsg := sendErr.Error()
			delivery.LastError = &errMsg
		} else {
			delivery.Status = model.DeliveryStatusFailed
			errMsg := sendErr.Error()
			delivery.LastError = &errMsg
		}
	} else {
		delivery.Status = model.DeliveryStatusSent
		delivery.SentAt = &now
		delivery.LastError = nil
		if res != nil {
			delivery.ProviderMessageID = &res.ProviderMessageID
		}
	}
	if err := s.deliveryRepo.Update(ctx, delivery); err != nil {
		return nil, err
	}

	// Audit Log
	meta, _ := json.Marshal(map[string]interface{}{
		"interview_id":    delivery.InterviewID,
		"delivery_id":     delivery.ID,
		"recipient_email": delivery.RecipientEmail,
		"delivery_status": delivery.Status,
		"attempt_count":   delivery.AttemptCount,
	})
	_ = s.auditRepo.Create(ctx, &model.AuditLog{
		Action:      model.AuditActionInterviewEmailRetried,
		ActorID:     actorID,
		JobID:       &interview.JobID,
		CandidateID: &interview.CandidateID,
		Metadata:    string(meta),
		CreatedAt:   time.Now().UTC(),
	})

	return delivery, nil
}

func (s *interviewNotificationService) ProcessDueReminders(ctx context.Context) (int, error) {
	now := time.Now().UTC()
	dueDeliveries, err := s.deliveryRepo.GetDuePendingReminders(ctx, now, 50)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch due reminders: %w", err)
	}

	processedCount := 0
	fromAddress := s.cfg.EmailFromAddress
	if fromAddress == "" {
		fromAddress = "onboarding@resend.dev"
	}
	fromHeader := fmt.Sprintf("%s <%s>", s.cfg.EmailFromName, fromAddress)

	for _, delivery := range dueDeliveries {
		claimed, err := s.deliveryRepo.ClaimReminder(ctx, delivery.ID)
		if err != nil || !claimed {
			continue
		}

		interview, err := s.interviewRepo.GetByID(ctx, delivery.InterviewID)
		if err != nil || interview == nil {
			delivery.Status = model.DeliveryStatusFailed
			errMsg := "associated interview not found"
			delivery.LastError = &errMsg
			_ = s.deliveryRepo.Update(ctx, delivery)
			continue
		}

		// Reminders are strictly excluded for cancelled or completed interviews
		if interview.Status != model.InterviewStatusScheduled {
			delivery.Status = model.DeliveryStatusSkipped
			skipMsg := fmt.Sprintf("interview is not scheduled (status: %s)", interview.Status)
			delivery.LastError = &skipMsg
			_ = s.deliveryRepo.Update(ctx, delivery)
			continue
		}

		jobTitle := "Job Position"
		jobCode := ""
		if interview.Job != nil {
			jobTitle = interview.Job.Title
			jobCode = interview.Job.Code
		}

		candidateName := "Candidate"
		if interview.Candidate != nil {
			candidateName = interview.Candidate.FullName
		}

		meetingURL := ""
		if interview.MeetingURL != nil {
			meetingURL = *interview.MeetingURL
		}
		location := ""
		if interview.Location != nil {
			location = *interview.Location
		}

		templateData := email.ReminderTemplateData{
			RecipientName:  delivery.RecipientName,
			CandidateName:  candidateName,
			JobTitle:       jobTitle,
			JobCode:        jobCode,
			StageLabel:     interview.Stage.DisplayLabel(),
			InterviewType:  string(interview.InterviewType),
			ScheduledStart: interview.ScheduledStart,
			ScheduledEnd:   interview.ScheduledEnd,
			MeetingURL:     meetingURL,
			Location:       location,
			InterviewID:    interview.ID,
			BaseURL:        s.cfg.EmailBaseURL,
		}

		subj, htmlBody, textBody := email.BuildReminder(templateData)
		msg := email.EmailMessage{
			From:           fromHeader,
			To:             delivery.RecipientEmail,
			ReplyTo:        s.cfg.EmailReplyTo,
			Subject:        subj,
			HTML:           htmlBody,
			Text:           textBody,
			IdempotencyKey: delivery.IdempotencyKey,
		}

		res, sendErr := s.emailProvider.Send(ctx, msg)
		sentTime := time.Now().UTC()
		if sendErr != nil {
			if errors.Is(sendErr, email.ErrEmailDisabled) {
				delivery.Status = model.DeliveryStatusSkipped
				errMsg := sendErr.Error()
				delivery.LastError = &errMsg
			} else {
				delivery.Status = model.DeliveryStatusFailed
				errMsg := sendErr.Error()
				delivery.LastError = &errMsg
			}
		} else {
			delivery.Status = model.DeliveryStatusSent
			delivery.SentAt = &sentTime
			delivery.LastError = nil
			if res != nil {
				delivery.ProviderMessageID = &res.ProviderMessageID
			}
		}
		_ = s.deliveryRepo.Update(ctx, delivery)

		meta, _ := json.Marshal(map[string]interface{}{
			"interview_id":    interview.ID,
			"delivery_id":     delivery.ID,
			"recipient_email": delivery.RecipientEmail,
			"delivery_status": delivery.Status,
		})
		_ = s.auditRepo.Create(ctx, &model.AuditLog{
			Action:      model.AuditActionInterviewReminderSent,
			ActorID:     interview.CreatedBy,
			JobID:       &interview.JobID,
			CandidateID: &interview.CandidateID,
			Metadata:    string(meta),
			CreatedAt:   time.Now().UTC(),
		})

		processedCount++
	}

	return processedCount, nil
}
