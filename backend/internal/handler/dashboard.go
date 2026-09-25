package handler

import (
	"net/http"
	"strconv"

	"hirescope/backend/internal/service"

	"github.com/gin-gonic/gin"
)

// DashboardHandler handles HTTP requests for dashboard KPI metrics and pipeline intelligence.
type DashboardHandler struct {
	dashboardService service.DashboardService
}

// NewDashboardHandler creates a new DashboardHandler instance.
func NewDashboardHandler(dashboardService service.DashboardService) *DashboardHandler {
	return &DashboardHandler{dashboardService: dashboardService}
}

// GetSummary handles GET /api/v1/dashboard/summary
func (h *DashboardHandler) GetSummary(c *gin.Context) {
	summary, err := h.dashboardService.GetSummary(c.Request.Context())
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve dashboard summary")
		return
	}
	RespondSuccess(c, http.StatusOK, summary)
}

// GetRecentCandidates handles GET /api/v1/dashboard/recent-candidates
func (h *DashboardHandler) GetRecentCandidates(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5"))
	candidates, err := h.dashboardService.GetRecentCandidates(c.Request.Context(), limit)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve recent candidates")
		return
	}
	RespondSuccess(c, http.StatusOK, candidates)
}

// GetOpenJobs handles GET /api/v1/dashboard/open-jobs
func (h *DashboardHandler) GetOpenJobs(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5"))
	jobs, err := h.dashboardService.GetOpenJobs(c.Request.Context(), limit)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve open jobs")
		return
	}
	RespondSuccess(c, http.StatusOK, jobs)
}

// GetActivity handles GET /api/v1/dashboard/activity
func (h *DashboardHandler) GetActivity(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	activity, err := h.dashboardService.GetRecentActivity(c.Request.Context(), limit)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve recent activity")
		return
	}
	RespondSuccess(c, http.StatusOK, activity)
}
