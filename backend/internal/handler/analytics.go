package handler

import (
	"errors"
	"fmt"
	"net/http"

	"hirescope/backend/internal/service"

	"github.com/gin-gonic/gin"
)

// AnalyticsHandler handles HTTP endpoints for deterministic recruitment analytics and reports.
type AnalyticsHandler struct {
	analyticsService service.AnalyticsService
}

// NewAnalyticsHandler creates a new AnalyticsHandler instance.
func NewAnalyticsHandler(analyticsService service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{analyticsService: analyticsService}
}

// extractQuery extracts query parameters into an AnalyticsQuery struct.
func extractQuery(c *gin.Context) service.AnalyticsQuery {
	return service.AnalyticsQuery{
		From:        c.Query("from"),
		To:          c.Query("to"),
		JobID:       c.Query("job_id"),
		Department:  c.Query("department"),
		JobStatus:   c.Query("job_status"),
		RecruiterID: c.Query("recruiter_id"),
		Interval:    c.DefaultQuery("interval", "day"),
	}
}

// GetOverview handles GET /api/v1/analytics/overview
func (h *AnalyticsHandler) GetOverview(c *gin.Context) {
	query := extractQuery(c)
	resp, err := h.analyticsService.GetOverview(c.Request.Context(), query)
	if err != nil {
		if errors.Is(err, service.ErrInvalidDateRange) || errors.Is(err, service.ErrDateRangeTooLong) || errors.Is(err, service.ErrInvalidInterval) {
			RespondError(c, http.StatusBadRequest, "INVALID_QUERY_PARAMS", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	RespondSuccess(c, http.StatusOK, resp)
}

// GetJobPerformance handles GET /api/v1/analytics/jobs
func (h *AnalyticsHandler) GetJobPerformance(c *gin.Context) {
	query := extractQuery(c)
	resp, err := h.analyticsService.GetJobPerformance(c.Request.Context(), query)
	if err != nil {
		if errors.Is(err, service.ErrInvalidDateRange) || errors.Is(err, service.ErrDateRangeTooLong) {
			RespondError(c, http.StatusBadRequest, "INVALID_QUERY_PARAMS", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	RespondSuccess(c, http.StatusOK, resp)
}

// GetRecruiterActivity handles GET /api/v1/analytics/recruiters
func (h *AnalyticsHandler) GetRecruiterActivity(c *gin.Context) {
	query := extractQuery(c)
	resp, err := h.analyticsService.GetRecruiterActivity(c.Request.Context(), query)
	if err != nil {
		if errors.Is(err, service.ErrInvalidDateRange) || errors.Is(err, service.ErrDateRangeTooLong) {
			RespondError(c, http.StatusBadRequest, "INVALID_QUERY_PARAMS", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	RespondSuccess(c, http.StatusOK, resp)
}

// GetDepartments handles GET /api/v1/analytics/departments
func (h *AnalyticsHandler) GetDepartments(c *gin.Context) {
	departments, err := h.analyticsService.GetDepartments(c.Request.Context())
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	RespondSuccess(c, http.StatusOK, gin.H{"departments": departments})
}

// ExportCSV handles GET /api/v1/analytics/export
func (h *AnalyticsHandler) ExportCSV(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	actorID := ""
	if exists {
		if uid, ok := userIDVal.(string); ok {
			actorID = uid
		}
	}

	query := extractQuery(c)
	csvBytes, filename, err := h.analyticsService.ExportCSV(c.Request.Context(), query, actorID)
	if err != nil {
		if errors.Is(err, service.ErrInvalidDateRange) || errors.Is(err, service.ErrDateRangeTooLong) {
			RespondError(c, http.StatusBadRequest, "INVALID_QUERY_PARAMS", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	c.Data(http.StatusOK, "text/csv; charset=utf-8", csvBytes)
}
