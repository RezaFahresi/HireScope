package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HealthResponse represents the health check response body.
type HealthResponse struct {
	Status string `json:"status"`
}

// HealthHandler handles health check requests.
type HealthHandler struct{}

// NewHealthHandler creates a new instance of HealthHandler.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Check handles GET /api/v1/health
func (h *HealthHandler) Check(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status: "ok",
	})
}
