package handler

import (
	"errors"
	"net/http"

	"hirescope/backend/internal/repository"
	"hirescope/backend/internal/service"

	"github.com/gin-gonic/gin"
)

// ScreeningHandler handles HTTP requests for candidate screening and matching evaluations.
type ScreeningHandler struct {
	screeningService service.ScreeningService
}

// NewScreeningHandler creates an instance of ScreeningHandler.
func NewScreeningHandler(screeningService service.ScreeningService) *ScreeningHandler {
	return &ScreeningHandler{
		screeningService: screeningService,
	}
}

// ScreenCandidate handles POST /api/v1/jobs/:id/candidates/:candidateId/screen
func (h *ScreeningHandler) ScreenCandidate(c *gin.Context) {
	jobID := c.Param("id")
	candidateID := c.Param("candidateId")

	resp, err := h.screeningService.ScreenCandidate(c.Request.Context(), jobID, candidateID)
	if err != nil {
		if errors.Is(err, service.ErrInvalidUUID) {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid job ID or candidate ID format")
			return
		}
		if errors.Is(err, repository.ErrJobNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "job not found")
			return
		}
		if errors.Is(err, repository.ErrCandidateNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "candidate not found")
			return
		}
		if errors.Is(err, repository.ErrJobCandidateNotFound) {
			RespondError(c, http.StatusBadRequest, "INVALID_ASSOCIATION", "candidate is not associated with this job")
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred while executing screening")
		return
	}

	RespondSuccess(c, http.StatusOK, resp)
}

// GetLatestScreening handles GET /api/v1/jobs/:id/candidates/:candidateId/screening
func (h *ScreeningHandler) GetLatestScreening(c *gin.Context) {
	jobID := c.Param("id")
	candidateID := c.Param("candidateId")

	resp, err := h.screeningService.GetLatestScreeningResult(c.Request.Context(), jobID, candidateID)
	if err != nil {
		if errors.Is(err, service.ErrInvalidUUID) {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid job ID or candidate ID format")
			return
		}
		if errors.Is(err, repository.ErrJobNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "job not found")
			return
		}
		if errors.Is(err, repository.ErrCandidateNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "candidate not found")
			return
		}
		if errors.Is(err, repository.ErrJobCandidateNotFound) {
			RespondError(c, http.StatusBadRequest, "INVALID_ASSOCIATION", "candidate is not associated with this job")
			return
		}
		if errors.Is(err, repository.ErrScreeningResultNotFound) {
			RespondError(c, http.StatusNotFound, "SCREENING_NOT_FOUND", "No screening result exists for this candidate and job.")
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred while retrieving screening result")
		return
	}

	RespondSuccess(c, http.StatusOK, resp)
}
