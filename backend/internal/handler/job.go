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
	"github.com/google/uuid"
)

// JobHandler handles job vacancy and recruitment requirement endpoints.
type JobHandler struct {
	jobService service.JobService
}

// NewJobHandler creates a new JobHandler instance.
func NewJobHandler(jobService service.JobService) *JobHandler {
	return &JobHandler{jobService: jobService}
}

// Create handles POST /api/v1/jobs
func (h *JobHandler) Create(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthenticated request")
		return
	}
	creatorID := userIDVal.(string)

	var input service.CreateJobInput
	if err := c.ShouldBindJSON(&input); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	job, err := h.jobService.CreateJob(c.Request.Context(), creatorID, input)
	if err != nil {
		if errors.Is(err, service.ErrTitleRequired) ||
			errors.Is(err, service.ErrDescriptionRequired) ||
			errors.Is(err, service.ErrInvalidEmploymentType) ||
			errors.Is(err, service.ErrInvalidJobStatus) {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		if strings.Contains(err.Error(), "closing_date") || strings.Contains(err.Error(), "length") {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create job vacancy")
		return
	}

	RespondSuccess(c, http.StatusCreated, job)
}

// List handles GET /api/v1/jobs
func (h *JobHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	filter := repository.JobFilter{
		Search:         c.Query("search"),
		Status:         c.Query("status"),
		Department:     c.Query("department"),
		EmploymentType: c.Query("employment_type"),
		Page:           page,
		Limit:          limit,
		Sort:           c.DefaultQuery("sort", "created_at"),
		Order:          c.DefaultQuery("order", "desc"),
	}

	result, err := h.jobService.ListJobs(c.Request.Context(), filter)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve jobs")
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

// GetByID handles GET /api/v1/jobs/:id
func (h *JobHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid job ID format")
		return
	}

	job, err := h.jobService.GetJobByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "job vacancy not found")
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve job")
		return
	}

	RespondSuccess(c, http.StatusOK, job)
}

// Update handles PUT /api/v1/jobs/:id
func (h *JobHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid job ID format")
		return
	}

	var input service.UpdateJobInput
	if err := c.ShouldBindJSON(&input); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	job, err := h.jobService.UpdateJob(c.Request.Context(), id, input)
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "job vacancy not found")
			return
		}
		if errors.Is(err, service.ErrInvalidStatusTransition) {
			RespondError(c, http.StatusBadRequest, "INVALID_STATUS_TRANSITION", err.Error())
			return
		}
		if errors.Is(err, service.ErrTitleRequired) ||
			errors.Is(err, service.ErrDescriptionRequired) ||
			errors.Is(err, service.ErrInvalidEmploymentType) ||
			errors.Is(err, service.ErrInvalidJobStatus) {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		if strings.Contains(err.Error(), "closing_date") || strings.Contains(err.Error(), "length") {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update job")
		return
	}

	RespondSuccess(c, http.StatusOK, job)
}

// UpdateStatus handles PATCH /api/v1/jobs/:id/status
func (h *JobHandler) UpdateStatus(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid job ID format")
		return
	}

	var req struct {
		Status model.JobStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	if !req.Status.IsValid() {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid job status")
		return
	}

	job, err := h.jobService.UpdateJob(c.Request.Context(), id, service.UpdateJobInput{
		Status: &req.Status,
	})
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "job vacancy not found")
			return
		}
		if errors.Is(err, service.ErrInvalidStatusTransition) {
			RespondError(c, http.StatusBadRequest, "INVALID_STATUS_TRANSITION", err.Error())
			return
		}
		if errors.Is(err, service.ErrInvalidJobStatus) {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update job status")
		return
	}

	RespondSuccess(c, http.StatusOK, job)
}

// Archive handles POST /api/v1/jobs/:id/archive
func (h *JobHandler) Archive(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid job ID format")
		return
	}

	job, err := h.jobService.ArchiveJob(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "job vacancy not found")
			return
		}
		if errors.Is(err, service.ErrInvalidStatusTransition) {
			RespondError(c, http.StatusBadRequest, "INVALID_STATUS_TRANSITION", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to archive job")
		return
	}

	RespondSuccess(c, http.StatusOK, job)
}

func getJobID(c *gin.Context) string {
	id := c.Param("jobId")
	if id == "" {
		id = c.Param("id")
	}
	return id
}

// ListRequirements handles GET /api/v1/jobs/:id/requirements
func (h *JobHandler) ListRequirements(c *gin.Context) {
	jobID := getJobID(c)
	if _, err := uuid.Parse(jobID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid job ID format")
		return
	}

	requirements, err := h.jobService.ListRequirements(c.Request.Context(), jobID)
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "job vacancy not found")
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve requirements")
		return
	}

	RespondSuccess(c, http.StatusOK, requirements)
}

// CreateRequirement handles POST /api/v1/jobs/:id/requirements
func (h *JobHandler) CreateRequirement(c *gin.Context) {
	jobID := getJobID(c)
	if _, err := uuid.Parse(jobID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid job ID format")
		return
	}

	var input service.CreateRequirementInput
	if err := c.ShouldBindJSON(&input); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	req, err := h.jobService.CreateRequirement(c.Request.Context(), jobID, input)
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "job vacancy not found")
			return
		}
		if errors.Is(err, service.ErrRequirementTextRequired) ||
			errors.Is(err, service.ErrInvalidCategory) ||
			errors.Is(err, service.ErrInvalidImportance) {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		if strings.Contains(err.Error(), "length") {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create requirement")
		return
	}

	RespondSuccess(c, http.StatusCreated, req)
}

// UpdateRequirement handles PUT /api/v1/jobs/:id/requirements/:requirementId
func (h *JobHandler) UpdateRequirement(c *gin.Context) {
	jobID := getJobID(c)
	if _, err := uuid.Parse(jobID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid job ID format")
		return
	}

	reqID := c.Param("requirementId")
	if _, err := uuid.Parse(reqID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid requirement ID format")
		return
	}

	var input service.UpdateRequirementInput
	if err := c.ShouldBindJSON(&input); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	req, err := h.jobService.UpdateRequirement(c.Request.Context(), jobID, reqID, input)
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "job vacancy not found")
			return
		}
		if errors.Is(err, repository.ErrRequirementNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "requirement not found")
			return
		}
		if errors.Is(err, service.ErrCrossJobRequirement) {
			RespondError(c, http.StatusBadRequest, "INVALID_RELATION", "requirement does not belong to the specified job")
			return
		}
		if errors.Is(err, service.ErrRequirementTextRequired) ||
			errors.Is(err, service.ErrInvalidCategory) ||
			errors.Is(err, service.ErrInvalidImportance) {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update requirement")
		return
	}

	RespondSuccess(c, http.StatusOK, req)
}

// DeleteRequirement handles DELETE /api/v1/jobs/:id/requirements/:requirementId
func (h *JobHandler) DeleteRequirement(c *gin.Context) {
	jobID := getJobID(c)
	if _, err := uuid.Parse(jobID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid job ID format")
		return
	}

	reqID := c.Param("requirementId")
	if _, err := uuid.Parse(reqID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid requirement ID format")
		return
	}

	err := h.jobService.DeleteRequirement(c.Request.Context(), jobID, reqID)
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "job vacancy not found")
			return
		}
		if errors.Is(err, repository.ErrRequirementNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "requirement not found")
			return
		}
		if errors.Is(err, service.ErrCrossJobRequirement) {
			RespondError(c, http.StatusBadRequest, "INVALID_RELATION", "requirement does not belong to the specified job")
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete requirement")
		return
	}

	RespondSuccess(c, http.StatusOK, gin.H{
		"message": "requirement successfully deleted",
	})
}
