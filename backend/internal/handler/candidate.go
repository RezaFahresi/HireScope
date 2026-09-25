package handler

import (
	"errors"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"hirescope/backend/internal/extractor"
	"hirescope/backend/internal/repository"
	"hirescope/backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CandidateHandler handles candidate profiles, education, experience, skills,
// documents, and job association endpoints.
type CandidateHandler struct {
	candidateService service.CandidateService
}

// NewCandidateHandler creates a new CandidateHandler instance.
func NewCandidateHandler(candidateService service.CandidateService) *CandidateHandler {
	return &CandidateHandler{candidateService: candidateService}
}

// LinkCandidateRequest payload for associating candidate with a job vacancy.
type LinkCandidateRequest struct {
	CandidateID string `json:"candidate_id" binding:"required"`
}

// Create handles POST /api/v1/candidates
func (h *CandidateHandler) Create(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthenticated request")
		return
	}
	creatorID := userIDVal.(string)

	var input service.CreateCandidateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	candidate, err := h.candidateService.CreateCandidate(c.Request.Context(), creatorID, input)
	if err != nil {
		if errors.Is(err, service.ErrFullNameRequired) ||
			errors.Is(err, service.ErrInvalidEmailFormat) ||
			strings.Contains(err.Error(), "length") ||
			strings.Contains(err.Error(), "blank") {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create candidate profile")
		return
	}

	RespondSuccess(c, http.StatusCreated, candidate)
}

// List handles GET /api/v1/candidates
func (h *CandidateHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	filter := repository.CandidateFilter{
		Search:   c.Query("search"),
		Location: c.Query("location"),
		Page:     page,
		Limit:    limit,
		Sort:     c.DefaultQuery("sort", "created_at"),
		Order:    c.DefaultQuery("order", "desc"),
	}

	result, err := h.candidateService.ListCandidates(c.Request.Context(), filter)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve candidates")
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

// GetByID handles GET /api/v1/candidates/:id
func (h *CandidateHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid candidate ID format")
		return
	}

	candidate, err := h.candidateService.GetCandidateByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "candidate not found")
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve candidate profile")
		return
	}

	RespondSuccess(c, http.StatusOK, candidate)
}

// Update handles PUT /api/v1/candidates/:id
func (h *CandidateHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if _, err := uuid.Parse(id); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid candidate ID format")
		return
	}

	var input service.UpdateCandidateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	updated, err := h.candidateService.UpdateCandidate(c.Request.Context(), id, input)
	if err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "candidate not found")
			return
		}
		if errors.Is(err, service.ErrFullNameRequired) ||
			errors.Is(err, service.ErrInvalidEmailFormat) ||
			strings.Contains(err.Error(), "length") ||
			strings.Contains(err.Error(), "blank") {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update candidate profile")
		return
	}

	RespondSuccess(c, http.StatusOK, updated)
}

// --- Job-Candidate Associations ---

// AddJobCandidate handles POST /api/v1/jobs/:id/candidates
func (h *CandidateHandler) AddJobCandidate(c *gin.Context) {
	jobID := c.Param("id")
	if _, err := uuid.Parse(jobID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid job ID format")
		return
	}

	var req LinkCandidateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "candidate_id is required")
		return
	}

	if _, err := uuid.Parse(req.CandidateID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid candidate_id format")
		return
	}

	err := h.candidateService.AddCandidateToJob(c.Request.Context(), jobID, req.CandidateID)
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "job not found")
			return
		}
		if errors.Is(err, repository.ErrCandidateNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "candidate not found")
			return
		}
		if errors.Is(err, repository.ErrJobCandidateAlreadyExists) {
			RespondError(c, http.StatusConflict, "CONFLICT", "candidate is already associated with this job")
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to link candidate to job")
		return
	}

	RespondSuccess(c, http.StatusCreated, gin.H{
		"message":      "candidate successfully linked to job",
		"job_id":       jobID,
		"candidate_id": req.CandidateID,
	})
}

// RemoveJobCandidate handles DELETE /api/v1/jobs/:id/candidates/:candidateId
func (h *CandidateHandler) RemoveJobCandidate(c *gin.Context) {
	jobID := c.Param("id")
	if _, err := uuid.Parse(jobID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid job ID format")
		return
	}

	candidateID := c.Param("candidateId")
	if _, err := uuid.Parse(candidateID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid candidate ID format")
		return
	}

	err := h.candidateService.RemoveCandidateFromJob(c.Request.Context(), jobID, candidateID)
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "job not found")
			return
		}
		if errors.Is(err, repository.ErrCandidateNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "candidate not found")
			return
		}
		if errors.Is(err, repository.ErrJobCandidateNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "candidate is not associated with this job")
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to unlink candidate from job")
		return
	}

	RespondSuccess(c, http.StatusOK, gin.H{"message": "candidate successfully unlinked from job"})
}

// ListCandidatesByJob handles GET /api/v1/jobs/:id/candidates
func (h *CandidateHandler) ListCandidatesByJob(c *gin.Context) {
	jobID := c.Param("id")
	if _, err := uuid.Parse(jobID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid job ID format")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	result, err := h.candidateService.ListCandidatesByJob(c.Request.Context(), jobID, page, limit)
	if err != nil {
		if errors.Is(err, repository.ErrJobNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "job not found")
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

// ListJobsByCandidate handles GET /api/v1/candidates/:id/jobs
func (h *CandidateHandler) ListJobsByCandidate(c *gin.Context) {
	candidateID := c.Param("id")
	if _, err := uuid.Parse(candidateID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid candidate ID format")
		return
	}

	jobs, err := h.candidateService.ListJobsByCandidate(c.Request.Context(), candidateID)
	if err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "candidate not found")
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to list jobs for candidate")
		return
	}

	RespondSuccess(c, http.StatusOK, jobs)
}

// --- Education Endpoints ---

// CreateEducation handles POST /api/v1/candidates/:id/educations
func (h *CandidateHandler) CreateEducation(c *gin.Context) {
	candidateID := c.Param("id")
	if _, err := uuid.Parse(candidateID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid candidate ID format")
		return
	}

	var input service.CreateEducationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	edu, err := h.candidateService.CreateEducation(c.Request.Context(), candidateID, input)
	if err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "candidate not found")
			return
		}
		if errors.Is(err, service.ErrInstitutionRequired) ||
			errors.Is(err, service.ErrDateOrder) ||
			strings.Contains(err.Error(), "date format") ||
			strings.Contains(err.Error(), "blank") {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create education record")
		return
	}

	RespondSuccess(c, http.StatusCreated, edu)
}

// ListEducations handles GET /api/v1/candidates/:id/educations
func (h *CandidateHandler) ListEducations(c *gin.Context) {
	candidateID := c.Param("id")
	if _, err := uuid.Parse(candidateID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid candidate ID format")
		return
	}

	list, err := h.candidateService.ListEducations(c.Request.Context(), candidateID)
	if err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "candidate not found")
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve educations")
		return
	}

	RespondSuccess(c, http.StatusOK, list)
}

// UpdateEducation handles PUT /api/v1/candidates/:id/educations/:educationId
func (h *CandidateHandler) UpdateEducation(c *gin.Context) {
	candidateID := c.Param("id")
	if _, err := uuid.Parse(candidateID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid candidate ID format")
		return
	}

	eduID := c.Param("educationId")
	if _, err := uuid.Parse(eduID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid education ID format")
		return
	}

	var input service.UpdateEducationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	edu, err := h.candidateService.UpdateEducation(c.Request.Context(), candidateID, eduID, input)
	if err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) || errors.Is(err, repository.ErrEducationNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "education record not found")
			return
		}
		if errors.Is(err, service.ErrInstitutionRequired) ||
			errors.Is(err, service.ErrDateOrder) ||
			strings.Contains(err.Error(), "date format") ||
			strings.Contains(err.Error(), "blank") {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update education record")
		return
	}

	RespondSuccess(c, http.StatusOK, edu)
}

// DeleteEducation handles DELETE /api/v1/candidates/:id/educations/:educationId
func (h *CandidateHandler) DeleteEducation(c *gin.Context) {
	candidateID := c.Param("id")
	if _, err := uuid.Parse(candidateID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid candidate ID format")
		return
	}

	eduID := c.Param("educationId")
	if _, err := uuid.Parse(eduID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid education ID format")
		return
	}

	err := h.candidateService.DeleteEducation(c.Request.Context(), candidateID, eduID)
	if err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) || errors.Is(err, repository.ErrEducationNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "education record not found")
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete education record")
		return
	}

	RespondSuccess(c, http.StatusOK, gin.H{"message": "education record deleted successfully"})
}

// --- Experience Endpoints ---

// CreateExperience handles POST /api/v1/candidates/:id/experiences
func (h *CandidateHandler) CreateExperience(c *gin.Context) {
	candidateID := c.Param("id")
	if _, err := uuid.Parse(candidateID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid candidate ID format")
		return
	}

	var input service.CreateExperienceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	exp, err := h.candidateService.CreateExperience(c.Request.Context(), candidateID, input)
	if err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "candidate not found")
			return
		}
		if errors.Is(err, service.ErrCompanyRequired) ||
			errors.Is(err, service.ErrPositionRequired) ||
			errors.Is(err, service.ErrInvalidEmploymentType) ||
			errors.Is(err, service.ErrDateOrder) ||
			strings.Contains(err.Error(), "date format") ||
			strings.Contains(err.Error(), "blank") {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create experience record")
		return
	}

	RespondSuccess(c, http.StatusCreated, exp)
}

// ListExperiences handles GET /api/v1/candidates/:id/experiences
func (h *CandidateHandler) ListExperiences(c *gin.Context) {
	candidateID := c.Param("id")
	if _, err := uuid.Parse(candidateID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid candidate ID format")
		return
	}

	list, err := h.candidateService.ListExperiences(c.Request.Context(), candidateID)
	if err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "candidate not found")
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve experiences")
		return
	}

	RespondSuccess(c, http.StatusOK, list)
}

// UpdateExperience handles PUT /api/v1/candidates/:id/experiences/:experienceId
func (h *CandidateHandler) UpdateExperience(c *gin.Context) {
	candidateID := c.Param("id")
	if _, err := uuid.Parse(candidateID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid candidate ID format")
		return
	}

	expID := c.Param("experienceId")
	if _, err := uuid.Parse(expID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid experience ID format")
		return
	}

	var input service.UpdateExperienceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	exp, err := h.candidateService.UpdateExperience(c.Request.Context(), candidateID, expID, input)
	if err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) || errors.Is(err, repository.ErrExperienceNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "experience record not found")
			return
		}
		if errors.Is(err, service.ErrCompanyRequired) ||
			errors.Is(err, service.ErrPositionRequired) ||
			errors.Is(err, service.ErrInvalidEmploymentType) ||
			errors.Is(err, service.ErrDateOrder) ||
			strings.Contains(err.Error(), "date format") ||
			strings.Contains(err.Error(), "blank") {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update experience record")
		return
	}

	RespondSuccess(c, http.StatusOK, exp)
}

// DeleteExperience handles DELETE /api/v1/candidates/:id/experiences/:experienceId
func (h *CandidateHandler) DeleteExperience(c *gin.Context) {
	candidateID := c.Param("id")
	if _, err := uuid.Parse(candidateID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid candidate ID format")
		return
	}

	expID := c.Param("experienceId")
	if _, err := uuid.Parse(expID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid experience ID format")
		return
	}

	err := h.candidateService.DeleteExperience(c.Request.Context(), candidateID, expID)
	if err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) || errors.Is(err, repository.ErrExperienceNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "experience record not found")
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete experience record")
		return
	}

	RespondSuccess(c, http.StatusOK, gin.H{"message": "experience record deleted successfully"})
}

// --- Skill Endpoints ---

// CreateSkill handles POST /api/v1/candidates/:id/skills
func (h *CandidateHandler) CreateSkill(c *gin.Context) {
	candidateID := c.Param("id")
	if _, err := uuid.Parse(candidateID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid candidate ID format")
		return
	}

	var input service.CreateSkillInput
	if err := c.ShouldBindJSON(&input); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	skill, err := h.candidateService.CreateSkill(c.Request.Context(), candidateID, input)
	if err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "candidate not found")
			return
		}
		if errors.Is(err, repository.ErrDuplicateSkill) {
			RespondError(c, http.StatusConflict, "CONFLICT", "skill already exists for this candidate")
			return
		}
		if errors.Is(err, service.ErrSkillRequired) ||
			strings.Contains(err.Error(), "length") ||
			strings.Contains(err.Error(), "blank") {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create skill")
		return
	}

	RespondSuccess(c, http.StatusCreated, skill)
}

// ListSkills handles GET /api/v1/candidates/:id/skills
func (h *CandidateHandler) ListSkills(c *gin.Context) {
	candidateID := c.Param("id")
	if _, err := uuid.Parse(candidateID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid candidate ID format")
		return
	}

	skills, err := h.candidateService.ListSkills(c.Request.Context(), candidateID)
	if err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "candidate not found")
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve skills")
		return
	}

	RespondSuccess(c, http.StatusOK, skills)
}

// UpdateSkill handles PUT /api/v1/candidates/:id/skills/:skillId
func (h *CandidateHandler) UpdateSkill(c *gin.Context) {
	candidateID := c.Param("id")
	if _, err := uuid.Parse(candidateID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid candidate ID format")
		return
	}

	skillID := c.Param("skillId")
	if _, err := uuid.Parse(skillID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid skill ID format")
		return
	}

	var input service.UpdateSkillInput
	if err := c.ShouldBindJSON(&input); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	skill, err := h.candidateService.UpdateSkill(c.Request.Context(), candidateID, skillID, input)
	if err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) || errors.Is(err, repository.ErrSkillNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "skill not found")
			return
		}
		if errors.Is(err, repository.ErrDuplicateSkill) {
			RespondError(c, http.StatusConflict, "CONFLICT", "skill already exists for this candidate")
			return
		}
		if errors.Is(err, service.ErrSkillRequired) ||
			strings.Contains(err.Error(), "length") ||
			strings.Contains(err.Error(), "blank") {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to update skill")
		return
	}

	RespondSuccess(c, http.StatusOK, skill)
}

// DeleteSkill handles DELETE /api/v1/candidates/:id/skills/:skillId
func (h *CandidateHandler) DeleteSkill(c *gin.Context) {
	candidateID := c.Param("id")
	if _, err := uuid.Parse(candidateID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid candidate ID format")
		return
	}

	skillID := c.Param("skillId")
	if _, err := uuid.Parse(skillID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid skill ID format")
		return
	}

	err := h.candidateService.DeleteSkill(c.Request.Context(), candidateID, skillID)
	if err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) || errors.Is(err, repository.ErrSkillNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "skill not found")
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete skill")
		return
	}

	RespondSuccess(c, http.StatusOK, gin.H{"message": "skill deleted successfully"})
}

// --- Document Endpoints ---

// CreateDocument handles POST /api/v1/candidates/:id/documents (for direct TEXT CV input)
func (h *CandidateHandler) CreateDocument(c *gin.Context) {
	candidateID := c.Param("id")
	if _, err := uuid.Parse(candidateID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid candidate ID format")
		return
	}

	var input service.CreateDocumentInput
	if err := c.ShouldBindJSON(&input); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid request body")
		return
	}

	doc, err := h.candidateService.CreateDocument(c.Request.Context(), candidateID, input)
	if err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "candidate not found")
			return
		}
		if errors.Is(err, service.ErrRawTextRequired) ||
			errors.Is(err, service.ErrTextTooLarge) ||
			strings.Contains(err.Error(), "invalid document source type") ||
			strings.Contains(err.Error(), "direct JSON creation is only supported for TEXT") ||
			strings.Contains(err.Error(), "blank") {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to create candidate document")
		return
	}

	RespondSuccess(c, http.StatusCreated, doc)
}

// UploadDocument handles POST /api/v1/candidates/:id/documents/upload
func (h *CandidateHandler) UploadDocument(c *gin.Context) {
	candidateID := c.Param("id")
	if _, err := uuid.Parse(candidateID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid candidate ID format")
		return
	}

	// Limit multipart request size
	if err := c.Request.ParseMultipartForm(21 << 20); err != nil {
		RespondError(c, http.StatusBadRequest, "CV_FILE_TOO_LARGE", "file exceeds maximum allowed size of 20 MB")
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "file form field is required")
		return
	}

	if fileHeader.Size > service.MaxUploadFileSize {
		RespondError(c, http.StatusBadRequest, "CV_FILE_TOO_LARGE", "file exceeds maximum allowed size of 20 MB")
		return
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if ext != ".pdf" && ext != ".docx" {
		RespondError(c, http.StatusBadRequest, "INVALID_CV_FILE", "unsupported file format; only .pdf and .docx are supported")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		RespondError(c, http.StatusBadRequest, "INVALID_CV_FILE", "failed to open uploaded file")
		return
	}
	defer file.Close()

	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" {
		if ext == ".pdf" {
			mimeType = "application/pdf"
		} else {
			mimeType = "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
		}
	}

	doc, err := h.candidateService.UploadDocument(c.Request.Context(), candidateID, fileHeader.Filename, mimeType, fileHeader.Size, file)
	if err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "candidate not found")
			return
		}
		if errors.Is(err, service.ErrFileTooLarge) {
			RespondError(c, http.StatusBadRequest, "CV_FILE_TOO_LARGE", err.Error())
			return
		}
		if errors.Is(err, extractor.ErrInvalidPDF) || errors.Is(err, extractor.ErrInvalidDOCX) || errors.Is(err, extractor.ErrInvalidDocument) {
			RespondError(c, http.StatusBadRequest, "INVALID_CV_FILE", err.Error())
			return
		}
		if errors.Is(err, extractor.ErrUnsupportedFormat) || strings.Contains(err.Error(), "unsupported file extension") {
			RespondError(c, http.StatusBadRequest, "UNSUPPORTED_CV_FORMAT", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "CV_PROCESSING_FAILED", "failed to upload candidate CV document")
		return
	}

	RespondSuccess(c, http.StatusCreated, doc)
}

// ProcessDocument handles POST /api/v1/candidates/:id/documents/:documentId/process
func (h *CandidateHandler) ProcessDocument(c *gin.Context) {
	candidateID := c.Param("id")
	if _, err := uuid.Parse(candidateID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid candidate ID format")
		return
	}

	docID := c.Param("documentId")
	if _, err := uuid.Parse(docID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid document ID format")
		return
	}

	result, err := h.candidateService.ProcessDocument(c.Request.Context(), candidateID, docID)
	if err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "candidate not found")
			return
		}
		if errors.Is(err, repository.ErrDocumentNotFound) {
			RespondError(c, http.StatusNotFound, "CV_DOCUMENT_NOT_FOUND", "document not found")
			return
		}
		if errors.Is(err, service.ErrDocumentCandidateMismatch) {
			RespondError(c, http.StatusBadRequest, "CV_DOCUMENT_CANDIDATE_MISMATCH", err.Error())
			return
		}
		if errors.Is(err, extractor.ErrExtractionFailed) {
			RespondError(c, http.StatusBadRequest, "CV_TEXT_EXTRACTION_FAILED", err.Error())
			return
		}
		if errors.Is(err, extractor.ErrCVTextTooLarge) {
			RespondError(c, http.StatusBadRequest, "CV_TEXT_TOO_LARGE", err.Error())
			return
		}
		if errors.Is(err, extractor.ErrInvalidPDF) || errors.Is(err, extractor.ErrInvalidDOCX) || errors.Is(err, extractor.ErrInvalidDocument) {
			RespondError(c, http.StatusBadRequest, "INVALID_CV_FILE", err.Error())
			return
		}
		if errors.Is(err, extractor.ErrUnsupportedFormat) {
			RespondError(c, http.StatusBadRequest, "UNSUPPORTED_CV_FORMAT", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "CV_PROCESSING_FAILED", "failed to process candidate CV document")
		return
	}

	RespondSuccess(c, http.StatusOK, result)
}

// ListDocuments handles GET /api/v1/candidates/:id/documents
func (h *CandidateHandler) ListDocuments(c *gin.Context) {
	candidateID := c.Param("id")
	if _, err := uuid.Parse(candidateID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid candidate ID format")
		return
	}

	docs, err := h.candidateService.ListDocuments(c.Request.Context(), candidateID)
	if err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "candidate not found")
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve documents")
		return
	}

	RespondSuccess(c, http.StatusOK, docs)
}

// GetDocumentByID handles GET /api/v1/candidates/:id/documents/:documentId
func (h *CandidateHandler) GetDocumentByID(c *gin.Context) {
	candidateID := c.Param("id")
	if _, err := uuid.Parse(candidateID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid candidate ID format")
		return
	}

	docID := c.Param("documentId")
	if _, err := uuid.Parse(docID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid document ID format")
		return
	}

	doc, err := h.candidateService.GetDocumentByID(c.Request.Context(), candidateID, docID)
	if err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) || errors.Is(err, repository.ErrDocumentNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "document not found")
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve document")
		return
	}

	RespondSuccess(c, http.StatusOK, doc)
}

// DeleteDocument handles DELETE /api/v1/candidates/:id/documents/:documentId
func (h *CandidateHandler) DeleteDocument(c *gin.Context) {
	candidateID := c.Param("id")
	if _, err := uuid.Parse(candidateID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid candidate ID format")
		return
	}

	docID := c.Param("documentId")
	if _, err := uuid.Parse(docID); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid document ID format")
		return
	}

	err := h.candidateService.DeleteDocument(c.Request.Context(), candidateID, docID)
	if err != nil {
		if errors.Is(err, repository.ErrCandidateNotFound) || errors.Is(err, repository.ErrDocumentNotFound) {
			RespondError(c, http.StatusNotFound, "NOT_FOUND", "document not found")
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to delete document")
		return
	}

	RespondSuccess(c, http.StatusOK, gin.H{"message": "document deleted successfully"})
}
