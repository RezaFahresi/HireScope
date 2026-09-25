package handler

import (
	"errors"
	"net/http"
	"strings"

	"hirescope/backend/internal/repository"
	"hirescope/backend/internal/service"

	"github.com/gin-gonic/gin"
)

// LoginRequest defines the expected payload for user login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// AuthHandler handles authentication-related HTTP endpoints.
type AuthHandler struct {
	authService service.AuthService
}

// NewAuthHandler creates a new AuthHandler instance.
func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Login handles POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "email and password are required")
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || req.Password == "" {
		RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", "email and password cannot be blank")
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			RespondError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "invalid email or password")
			return
		}
		if errors.Is(err, service.ErrEmailRequired) || errors.Is(err, service.ErrPasswordRequired) || errors.Is(err, service.ErrInvalidEmailFormat) {
			RespondError(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "an internal error occurred while processing login")
		return
	}

	RespondSuccess(c, http.StatusOK, resp)
}

// Logout handles POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	rawTokenVal, exists := c.Get("rawToken")
	if !exists {
		// Fallback: extract from header directly
		authHeader := c.GetHeader("Authorization")
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 {
			rawTokenVal = strings.TrimSpace(parts[1])
		}
	}

	if rawTokenVal != nil {
		if rawToken, ok := rawTokenVal.(string); ok && rawToken != "" {
			_ = h.authService.Logout(c.Request.Context(), rawToken)
		}
	}

	RespondSuccess(c, http.StatusOK, gin.H{
		"message": "successfully logged out",
	})
}

// Me handles GET /api/v1/auth/me
func (h *AuthHandler) Me(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthenticated request")
		return
	}

	userID, ok := userIDVal.(string)
	if !ok || userID == "" {
		RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid user session")
		return
	}

	user, err := h.authService.GetCurrentUser(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			RespondError(c, http.StatusNotFound, "USER_NOT_FOUND", "user account not found")
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to retrieve user profile")
		return
	}

	RespondSuccess(c, http.StatusOK, user)
}
