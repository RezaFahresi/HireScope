package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"hirescope/backend/internal/repository"
	"hirescope/backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// InterviewHandler handles HTTP requests for interview management.
type InterviewHandler struct {
	interviewService    service.InterviewService
	notificationService service.InterviewNotificationService
}

// NewInterviewHandler constructs a new InterviewHandler.
func NewInterviewHandler(interviewService service.InterviewService, notificationServices ...service.InterviewNotificationService) *InterviewHandler {
	h := &InterviewHandler{
		interviewService: interviewService,
	}
	if len(notificationServices) > 0 {
		h.notificationService = notificationServices[0]
	}
	return h
}

// SetNotificationService assigns the notification service.
func (h *InterviewHandler) SetNotificationService(ns service.InterviewNotificationService) {
	h.notificationService = ns
}

// CreateInterview handles POST /api/v1/interviews
func (h *InterviewHandler) CreateInterview(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthenticated request")
		return
	}
	createdBy := userIDVal.(string)

	var req service.CreateInterviewInput
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	result, err := h.interviewService.CreateInterview(c.Request.Context(), req, createdBy)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	RespondSuccess(c, http.StatusCreated, result)
}

// GetInterview handles GET /api/v1/interviews/:id
func (h *InterviewHandler) GetInterview(c *gin.Context) {
	id := c.Param("id")
	result, err := h.interviewService.GetInterview(c.Request.Context(), id)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	RespondSuccess(c, http.StatusOK, result)
}

// UpdateInterview handles PUT /api/v1/interviews/:id
func (h *InterviewHandler) UpdateInterview(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthenticated request")
		return
	}
	actorID := userIDVal.(string)
	id := c.Param("id")

	var req service.UpdateInterviewInput
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	result, err := h.interviewService.UpdateInterview(c.Request.Context(), id, req, actorID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	RespondSuccess(c, http.StatusOK, result)
}

// RescheduleInterview handles PATCH /api/v1/interviews/:id/reschedule
func (h *InterviewHandler) RescheduleInterview(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthenticated request")
		return
	}
	actorID := userIDVal.(string)
	id := c.Param("id")

	var req service.RescheduleInterviewInput
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	result, err := h.interviewService.RescheduleInterview(c.Request.Context(), id, req, actorID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	RespondSuccess(c, http.StatusOK, result)
}

// CancelInterview handles PATCH /api/v1/interviews/:id/cancel
func (h *InterviewHandler) CancelInterview(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthenticated request")
		return
	}
	actorID := userIDVal.(string)
	id := c.Param("id")

	var req service.CancelInterviewInput
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	result, err := h.interviewService.CancelInterview(c.Request.Context(), id, req, actorID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	RespondSuccess(c, http.StatusOK, result)
}

// CompleteInterview handles POST /api/v1/interviews/:id/complete
func (h *InterviewHandler) CompleteInterview(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthenticated request")
		return
	}
	actorID := userIDVal.(string)
	id := c.Param("id")

	var req service.CompleteInterviewInput
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	result, err := h.interviewService.CompleteInterview(c.Request.Context(), id, req, actorID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	RespondSuccess(c, http.StatusOK, result)
}

// ListInterviews handles GET /api/v1/interviews
func (h *InterviewHandler) ListInterviews(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	var fromTime *time.Time
	if fromStr := c.Query("from"); fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			fromTime = &t
		}
	}

	var toTime *time.Time
	if toStr := c.Query("to"); toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			toTime = &t
		}
	}

	filter := service.InterviewFilterInput{
		JobID:         c.Query("job_id"),
		CandidateID:   c.Query("candidate_id"),
		Status:        c.Query("status"),
		Stage:         c.Query("stage"),
		InterviewType: c.Query("interview_type"),
		From:          fromTime,
		To:            toTime,
		Page:          page,
		Limit:         limit,
		Sort:          c.Query("sort"),
		Order:         c.Query("order"),
	}

	result, err := h.interviewService.ListInterviews(c.Request.Context(), filter)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	RespondSuccess(c, http.StatusOK, result)
}

// GetCandidateInterviews handles GET /api/v1/candidates/:id/interviews
func (h *InterviewHandler) GetCandidateInterviews(c *gin.Context) {
	candidateID := c.Param("id")
	if candidateID == "" {
		candidateID = c.Param("candidateId")
	}
	jobID := c.Query("job_id")

	result, err := h.interviewService.GetCandidateInterviews(c.Request.Context(), candidateID, jobID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	RespondSuccess(c, http.StatusOK, result)
}

// GetCandidateNextInterview handles GET /api/v1/candidates/:id/next-interview
func (h *InterviewHandler) GetCandidateNextInterview(c *gin.Context) {
	candidateID := c.Param("id")
	if candidateID == "" {
		candidateID = c.Param("candidateId")
	}
	jobID := c.Query("job_id")

	result, err := h.interviewService.GetCandidateNextInterview(c.Request.Context(), candidateID, jobID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	RespondSuccess(c, http.StatusOK, result)
}

// ListUsers handles GET /api/v1/users (for selecting interviewers)
func (h *InterviewHandler) ListUsers(c *gin.Context) {
	users, err := h.interviewService.ListUsers(c.Request.Context())
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list users")
		return
	}
	RespondSuccess(c, http.StatusOK, users)
}

// ExportICS handles GET /api/v1/interviews/:id/ics
func (h *InterviewHandler) ExportICS(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid interview ID format")
		return
	}

	filename, icsBytes, err := h.interviewService.GenerateICS(c.Request.Context(), id)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.Header("Content-Type", "text/calendar; charset=utf-8")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Data(http.StatusOK, "text/calendar; charset=utf-8", icsBytes)
}

func (h *InterviewHandler) handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidUUID):
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid UUID format")
	case errors.Is(err, service.ErrInvalidDateRange):
		RespondError(c, http.StatusBadRequest, "INVALID_DATE_RANGE", err.Error())
	case errors.Is(err, service.ErrDateRangeTooLarge):
		RespondError(c, http.StatusBadRequest, "DATE_RANGE_EXCEEDED", err.Error())
	case errors.Is(err, service.ErrCandidateNotAssociatedWithJob):
		RespondError(c, http.StatusBadRequest, "CANDIDATE_NOT_ASSOCIATED", err.Error())
	case errors.Is(err, service.ErrInvalidInterviewStage):
		RespondError(c, http.StatusBadRequest, "INVALID_STAGE", err.Error())
	case errors.Is(err, service.ErrInvalidInterviewType):
		RespondError(c, http.StatusBadRequest, "INVALID_TYPE", err.Error())
	case errors.Is(err, service.ErrInvalidInterviewTime):
		RespondError(c, http.StatusBadRequest, "INVALID_TIME", err.Error())
	case errors.Is(err, service.ErrInterviewDurationTooLong):
		RespondError(c, http.StatusBadRequest, "DURATION_TOO_LONG", err.Error())
	case errors.Is(err, service.ErrMissingMeetingURL):
		RespondError(c, http.StatusBadRequest, "MISSING_MEETING_URL", err.Error())
	case errors.Is(err, service.ErrMissingLocation):
		RespondError(c, http.StatusBadRequest, "MISSING_LOCATION", err.Error())
	case errors.Is(err, service.ErrNoInterviewers):
		RespondError(c, http.StatusBadRequest, "NO_INTERVIEWERS", err.Error())
	case errors.Is(err, service.ErrInterviewerConflict):
		RespondError(c, http.StatusConflict, "INTERVIEWER_CONFLICT", err.Error())
	case errors.Is(err, service.ErrInvalidStatusTransition):
		RespondError(c, http.StatusBadRequest, "INVALID_STATUS_TRANSITION", err.Error())
	case errors.Is(err, service.ErrCancellationReasonRequired):
		RespondError(c, http.StatusBadRequest, "CANCELLATION_REASON_REQUIRED", err.Error())
	case errors.Is(err, service.ErrCancellationReasonTooLong):
		RespondError(c, http.StatusBadRequest, "CANCELLATION_REASON_TOO_LONG", err.Error())
	case errors.Is(err, service.ErrFeedbackTooLong):
		RespondError(c, http.StatusBadRequest, "FEEDBACK_TOO_LONG", err.Error())
	case errors.Is(err, service.ErrInvalidInterviewResult):
		RespondError(c, http.StatusBadRequest, "INVALID_RESULT", err.Error())
	case errors.Is(err, service.ErrOnlyFailedDeliveriesCanBeRetried):
		RespondError(c, http.StatusBadRequest, "INVALID_RETRY_STATE", err.Error())
	case errors.Is(err, service.ErrMaxRetryAttemptsReached):
		RespondError(c, http.StatusBadRequest, "MAX_RETRY_REACHED", err.Error())
	case errors.Is(err, repository.ErrInterviewNotFound):
		RespondError(c, http.StatusNotFound, "INTERVIEW_NOT_FOUND", "interview not found")
	case errors.Is(err, repository.ErrDeliveryNotFound):
		RespondError(c, http.StatusNotFound, "DELIVERY_NOT_FOUND", "email delivery record not found")
	default:
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
}

// ListEmailDeliveries handles GET /api/v1/interviews/:id/email-deliveries
func (h *InterviewHandler) ListEmailDeliveries(c *gin.Context) {
	id := c.Param("id")
	if h.notificationService == nil {
		RespondSuccess(c, http.StatusOK, []interface{}{})
		return
	}
	deliveries, err := h.notificationService.ListDeliveries(c.Request.Context(), id)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	RespondSuccess(c, http.StatusOK, deliveries)
}

// SendInvitationRequest defines payload for triggering invitations manually.
type SendInvitationRequest struct {
	SendCandidate    *bool `json:"send_candidate"`
	SendInterviewers *bool `json:"send_interviewers"`
}

// SendInvitation handles POST /api/v1/interviews/:id/send-invitation
func (h *InterviewHandler) SendInvitation(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthenticated request")
		return
	}
	actorID := userIDVal.(string)
	id := c.Param("id")

	var req SendInvitationRequest
	_ = c.ShouldBindJSON(&req)

	sendCandidate := true
	if req.SendCandidate != nil {
		sendCandidate = *req.SendCandidate
	}
	sendInterviewers := true
	if req.SendInterviewers != nil {
		sendInterviewers = *req.SendInterviewers
	}

	if !sendCandidate && !sendInterviewers {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "at least one recipient type must be selected")
		return
	}

	if h.notificationService == nil {
		RespondError(c, http.StatusServiceUnavailable, "EMAIL_SERVICE_UNAVAILABLE", "email notification service is not configured")
		return
	}

	deliveries, err := h.notificationService.TriggerInvitations(c.Request.Context(), id, sendCandidate, sendInterviewers, actorID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	RespondSuccess(c, http.StatusOK, deliveries)
}

// RetryEmailDelivery handles POST /api/v1/interviews/:id/email-deliveries/:deliveryId/retry
func (h *InterviewHandler) RetryEmailDelivery(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthenticated request")
		return
	}
	actorID := userIDVal.(string)
	deliveryID := c.Param("deliveryId")

	if h.notificationService == nil {
		RespondError(c, http.StatusServiceUnavailable, "EMAIL_SERVICE_UNAVAILABLE", "email notification service is not configured")
		return
	}

	delivery, err := h.notificationService.RetryDelivery(c.Request.Context(), deliveryID, actorID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}
	RespondSuccess(c, http.StatusOK, delivery)
}
