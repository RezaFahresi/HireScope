package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"hirescope/backend/internal/model"
	"hirescope/backend/internal/repository"
	"hirescope/backend/internal/service"

	"github.com/gin-gonic/gin"
)

// CandidateReviewHandler handles HTTP requests for candidate recruiter workflow and review context.
type CandidateReviewHandler struct {
	reviewService service.CandidateReviewService
}

// NewCandidateReviewHandler creates a new CandidateReviewHandler instance.
func NewCandidateReviewHandler(reviewService service.CandidateReviewService) *CandidateReviewHandler {
	return &CandidateReviewHandler{
		reviewService: reviewService,
	}
}

// UpdateStatusRequest payload for updating candidate workflow status.
type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// NoteRequest payload for creating or updating a candidate review note.
type NoteRequest struct {
	Content string `json:"content" binding:"required"`
}

// ListCandidatesByJob handles GET /api/v1/jobs/:id/candidates
func (h *CandidateReviewHandler) ListCandidatesByJob(c *gin.Context) {
	jobID := c.Param("id")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	filter := service.JobCandidateFilterInput{
		JobID:  jobID,
		Status: c.Query("status"),
		Search: c.Query("search"),
		Page:   page,
		Limit:  limit,
		Sort:   c.Query("sort"),
		Order:  c.Query("order"),
	}

	result, err := h.reviewService.GetJobCandidates(c.Request.Context(), filter)
	if err != nil {
		if errors.Is(err, service.ErrInvalidUUID) {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid job ID format")
			return
		}
		if errors.Is(err, repository.ErrJobNotFound) {
			RespondError(c, http.StatusNotFound, "JOB_NOT_FOUND", "job not found")
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list candidates for job")
		return
	}

	RespondSuccess(c, http.StatusOK, gin.H{
		"items": result.Items,
		"pagination": gin.H{
			"page":        result.Page,
			"limit":       result.Limit,
			"total":       result.Total,
			"total_pages": result.TotalPages,
		},
	})
}

// UpdateStatus handles PATCH /api/v1/jobs/:id/candidates/:candidateId/status
func (h *CandidateReviewHandler) UpdateStatus(c *gin.Context) {
	jobID := c.Param("id")
	candidateID := c.Param("candidateId")

	userIDVal, exists := c.Get("userID")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthenticated request")
		return
	}
	reviewerID := userIDVal.(string)

	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "status is required")
		return
	}

	cleanStatus := model.JobCandidateStatus(strings.ToUpper(strings.TrimSpace(req.Status)))
	resp, err := h.reviewService.UpdateStatus(c.Request.Context(), jobID, candidateID, cleanStatus, reviewerID)
	if err != nil {
		if errors.Is(err, service.ErrInvalidUUID) {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid job ID or candidate ID format")
			return
		}
		if errors.Is(err, service.ErrInvalidStatus) {
			RespondError(c, http.StatusBadRequest, "INVALID_STATUS", "invalid candidate workflow status; must be REVIEW, SHORTLISTED, or REJECTED")
			return
		}
		if errors.Is(err, repository.ErrJobNotFound) {
			RespondError(c, http.StatusNotFound, "JOB_NOT_FOUND", "job not found")
			return
		}
		if errors.Is(err, repository.ErrCandidateNotFound) {
			RespondError(c, http.StatusNotFound, "CANDIDATE_NOT_FOUND", "candidate not found")
			return
		}
		if errors.Is(err, repository.ErrJobCandidateNotFound) {
			RespondError(c, http.StatusNotFound, "JOB_CANDIDATE_NOT_FOUND", "candidate is not associated with this job")
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred while updating status")
		return
	}

	RespondSuccess(c, http.StatusOK, resp)
}

// GetReviewDetail handles GET /api/v1/jobs/:id/candidates/:candidateId/review
func (h *CandidateReviewHandler) GetReviewDetail(c *gin.Context) {
	jobID := c.Param("id")
	candidateID := c.Param("candidateId")

	resp, err := h.reviewService.GetReviewDetail(c.Request.Context(), jobID, candidateID)
	if err != nil {
		if errors.Is(err, service.ErrInvalidUUID) {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid job ID or candidate ID format")
			return
		}
		if errors.Is(err, repository.ErrJobNotFound) {
			RespondError(c, http.StatusNotFound, "JOB_NOT_FOUND", "job not found")
			return
		}
		if errors.Is(err, repository.ErrCandidateNotFound) {
			RespondError(c, http.StatusNotFound, "CANDIDATE_NOT_FOUND", "candidate not found")
			return
		}
		if errors.Is(err, repository.ErrJobCandidateNotFound) {
			RespondError(c, http.StatusNotFound, "JOB_CANDIDATE_NOT_FOUND", "candidate is not associated with this job")
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred while retrieving review details")
		return
	}

	RespondSuccess(c, http.StatusOK, resp)
}

// CreateNote handles POST /api/v1/jobs/:id/candidates/:candidateId/notes
func (h *CandidateReviewHandler) CreateNote(c *gin.Context) {
	jobID := c.Param("id")
	candidateID := c.Param("candidateId")

	userIDVal, exists := c.Get("userID")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthenticated request")
		return
	}
	authorID := userIDVal.(string)

	var req NoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "content is required")
		return
	}

	resp, err := h.reviewService.CreateNote(c.Request.Context(), jobID, candidateID, authorID, req.Content)
	if err != nil {
		if errors.Is(err, service.ErrInvalidUUID) {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid job ID or candidate ID format")
			return
		}
		if errors.Is(err, service.ErrValidation) || strings.Contains(err.Error(), "content") {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		if errors.Is(err, repository.ErrJobNotFound) {
			RespondError(c, http.StatusNotFound, "JOB_NOT_FOUND", "job not found")
			return
		}
		if errors.Is(err, repository.ErrCandidateNotFound) {
			RespondError(c, http.StatusNotFound, "CANDIDATE_NOT_FOUND", "candidate not found")
			return
		}
		if errors.Is(err, repository.ErrJobCandidateNotFound) {
			RespondError(c, http.StatusNotFound, "JOB_CANDIDATE_NOT_FOUND", "candidate is not associated with this job")
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred while creating note")
		return
	}

	RespondSuccess(c, http.StatusCreated, resp)
}

// ListNotes handles GET /api/v1/jobs/:id/candidates/:candidateId/notes
func (h *CandidateReviewHandler) ListNotes(c *gin.Context) {
	jobID := c.Param("id")
	candidateID := c.Param("candidateId")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	resp, err := h.reviewService.ListNotes(c.Request.Context(), jobID, candidateID, page, limit)
	if err != nil {
		if errors.Is(err, service.ErrInvalidUUID) {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid job ID or candidate ID format")
			return
		}
		if errors.Is(err, repository.ErrJobCandidateNotFound) {
			RespondError(c, http.StatusNotFound, "JOB_CANDIDATE_NOT_FOUND", "candidate is not associated with this job")
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred while listing notes")
		return
	}

	RespondSuccess(c, http.StatusOK, resp)
}

// UpdateNote handles PUT /api/v1/jobs/:id/candidates/:candidateId/notes/:noteId
func (h *CandidateReviewHandler) UpdateNote(c *gin.Context) {
	jobID := c.Param("id")
	candidateID := c.Param("candidateId")
	noteID := c.Param("noteId")

	userIDVal, exists := c.Get("userID")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthenticated request")
		return
	}
	userID := userIDVal.(string)
	userRole := model.Role(c.GetString("userRole"))

	var req NoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "content is required")
		return
	}

	resp, err := h.reviewService.UpdateNote(c.Request.Context(), jobID, candidateID, noteID, userID, userRole, req.Content)
	if err != nil {
		if errors.Is(err, service.ErrInvalidUUID) {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid UUID format")
			return
		}
		if errors.Is(err, service.ErrValidation) || strings.Contains(err.Error(), "content") {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		if errors.Is(err, service.ErrNoteForbidden) {
			RespondError(c, http.StatusForbidden, "NOTE_FORBIDDEN", "you do not have permission to modify this note")
			return
		}
		if errors.Is(err, repository.ErrNoteNotFound) {
			RespondError(c, http.StatusNotFound, "NOTE_NOT_FOUND", "candidate note not found")
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred while updating note")
		return
	}

	RespondSuccess(c, http.StatusOK, resp)
}

// DeleteNote handles DELETE /api/v1/jobs/:id/candidates/:candidateId/notes/:noteId
func (h *CandidateReviewHandler) DeleteNote(c *gin.Context) {
	jobID := c.Param("id")
	candidateID := c.Param("candidateId")
	noteID := c.Param("noteId")

	userIDVal, exists := c.Get("userID")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthenticated request")
		return
	}
	userID := userIDVal.(string)
	userRole := model.Role(c.GetString("userRole"))

	err := h.reviewService.DeleteNote(c.Request.Context(), jobID, candidateID, noteID, userID, userRole)
	if err != nil {
		if errors.Is(err, service.ErrInvalidUUID) {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid UUID format")
			return
		}
		if errors.Is(err, service.ErrNoteForbidden) {
			RespondError(c, http.StatusForbidden, "NOTE_FORBIDDEN", "you do not have permission to modify this note")
			return
		}
		if errors.Is(err, repository.ErrNoteNotFound) {
			RespondError(c, http.StatusNotFound, "NOTE_NOT_FOUND", "candidate note not found")
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred while deleting note")
		return
	}

	RespondSuccess(c, http.StatusOK, gin.H{"message": "note deleted successfully"})
}
