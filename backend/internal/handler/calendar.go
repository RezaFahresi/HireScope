package handler

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"hirescope/backend/internal/calendar"
	"hirescope/backend/internal/config"
	"hirescope/backend/internal/model"
	"hirescope/backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CalendarHandler handles HTTP endpoints for external calendar integrations.
type CalendarHandler struct {
	calendarService service.CalendarIntegrationService
	cfg             *config.Config
}

// NewCalendarHandler creates a new CalendarHandler instance.
func NewCalendarHandler(calendarService service.CalendarIntegrationService, cfg *config.Config) *CalendarHandler {
	return &CalendarHandler{
		calendarService: calendarService,
		cfg:             cfg,
	}
}

// GetConnections handles GET /api/v1/calendar/connections
func (h *CalendarHandler) GetConnections(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthenticated request")
		return
	}
	userID := userIDVal.(string)

	conns, err := h.calendarService.GetConnections(c.Request.Context(), userID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	RespondSuccess(c, http.StatusOK, conns)
}

// ConnectGoogle handles GET /api/v1/calendar/google/connect
func (h *CalendarHandler) ConnectGoogle(c *gin.Context) {
	h.handleConnect(c, model.CalendarProviderGoogle)
}

// CallbackGoogle handles GET /api/v1/calendar/google/callback (Public endpoint)
func (h *CalendarHandler) CallbackGoogle(c *gin.Context) {
	h.handleCallback(c, model.CalendarProviderGoogle, "google")
}

// DisconnectGoogle handles DELETE /api/v1/calendar/google
func (h *CalendarHandler) DisconnectGoogle(c *gin.Context) {
	h.handleDisconnect(c, model.CalendarProviderGoogle)
}

// ConnectMicrosoft handles GET /api/v1/calendar/microsoft/connect
func (h *CalendarHandler) ConnectMicrosoft(c *gin.Context) {
	h.handleConnect(c, model.CalendarProviderMicrosoft)
}

// CallbackMicrosoft handles GET /api/v1/calendar/microsoft/callback (Public endpoint)
func (h *CalendarHandler) CallbackMicrosoft(c *gin.Context) {
	h.handleCallback(c, model.CalendarProviderMicrosoft, "microsoft")
}

// DisconnectMicrosoft handles DELETE /api/v1/calendar/microsoft
func (h *CalendarHandler) DisconnectMicrosoft(c *gin.Context) {
	h.handleDisconnect(c, model.CalendarProviderMicrosoft)
}

// GetInterviewCalendarEvents handles GET /api/v1/interviews/:id/calendar/events
func (h *CalendarHandler) GetInterviewCalendarEvents(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthenticated request")
		return
	}
	userID := userIDVal.(string)

	interviewID := c.Param("id")
	if _, err := uuid.Parse(interviewID); err != nil {
		RespondError(c, http.StatusBadRequest, "INVALID_ID", "invalid interview ID")
		return
	}

	events, err := h.calendarService.GetInterviewCalendarEvents(c.Request.Context(), userID, interviewID)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	RespondSuccess(c, http.StatusOK, events)
}

// SyncInterview handles POST /api/v1/interviews/:id/calendar/sync/:provider
func (h *CalendarHandler) SyncInterview(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthenticated request")
		return
	}
	userID := userIDVal.(string)

	interviewID := c.Param("id")
	if _, err := uuid.Parse(interviewID); err != nil {
		RespondError(c, http.StatusBadRequest, "INVALID_ID", "invalid interview ID")
		return
	}

	provType, err := parseProvider(c.Param("provider"))
	if err != nil {
		RespondError(c, http.StatusBadRequest, "INVALID_PROVIDER", err.Error())
		return
	}

	dto, err := h.calendarService.SyncInterview(c.Request.Context(), userID, interviewID, provType)
	if err != nil {
		h.handleSyncError(c, err)
		return
	}

	RespondSuccess(c, http.StatusOK, dto)
}

// RetrySync handles POST /api/v1/interviews/:id/calendar/retry/:provider
func (h *CalendarHandler) RetrySync(c *gin.Context) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthenticated request")
		return
	}
	userID := userIDVal.(string)

	interviewID := c.Param("id")
	if _, err := uuid.Parse(interviewID); err != nil {
		RespondError(c, http.StatusBadRequest, "INVALID_ID", "invalid interview ID")
		return
	}

	provType, err := parseProvider(c.Param("provider"))
	if err != nil {
		RespondError(c, http.StatusBadRequest, "INVALID_PROVIDER", err.Error())
		return
	}

	dto, err := h.calendarService.RetrySync(c.Request.Context(), userID, interviewID, provType)
	if err != nil {
		h.handleSyncError(c, err)
		return
	}

	RespondSuccess(c, http.StatusOK, dto)
}

func (h *CalendarHandler) handleConnect(c *gin.Context, provider model.CalendarProviderType) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthenticated request")
		return
	}
	userID := userIDVal.(string)

	authURL, err := h.calendarService.InitiateOAuth(c.Request.Context(), userID, provider)
	if err != nil {
		if errors.Is(err, calendar.ErrProviderNotConfigured) {
			RespondError(c, http.StatusBadRequest, "PROVIDER_NOT_CONFIGURED", fmt.Sprintf("%s calendar is not configured on this server", provider))
			return
		}
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	RespondSuccess(c, http.StatusOK, gin.H{"auth_url": authURL})
}

func (h *CalendarHandler) handleCallback(c *gin.Context, provider model.CalendarProviderType, providerSlug string) {
	frontendBase := "http://localhost:5173"
	if h.cfg != nil && h.cfg.EmailBaseURL != "" {
		frontendBase = h.cfg.EmailBaseURL
	}

	// Check if OAuth provider returned an error
	if oauthErr := c.Query("error"); oauthErr != "" {
		errDesc := c.DefaultQuery("error_description", oauthErr)
		redirectURL := fmt.Sprintf("%s/settings/integrations?calendar=%s&status=error&message=%s",
			frontendBase, providerSlug, url.QueryEscape(errDesc))
		c.Redirect(http.StatusTemporaryRedirect, redirectURL)
		return
	}

	code := c.Query("code")
	state := c.Query("state")

	if code == "" || state == "" {
		redirectURL := fmt.Sprintf("%s/settings/integrations?calendar=%s&status=error&message=%s",
			frontendBase, providerSlug, url.QueryEscape("missing authorization code or state"))
		c.Redirect(http.StatusTemporaryRedirect, redirectURL)
		return
	}

	_, err := h.calendarService.HandleOAuthCallback(c.Request.Context(), provider, state, code)
	if err != nil {
		redirectURL := fmt.Sprintf("%s/settings/integrations?calendar=%s&status=error&message=%s",
			frontendBase, providerSlug, url.QueryEscape(err.Error()))
		c.Redirect(http.StatusTemporaryRedirect, redirectURL)
		return
	}

	redirectURL := fmt.Sprintf("%s/settings/integrations?calendar=%s&status=success", frontendBase, providerSlug)
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}

func (h *CalendarHandler) handleDisconnect(c *gin.Context, provider model.CalendarProviderType) {
	userIDVal, exists := c.Get("userID")
	if !exists {
		RespondError(c, http.StatusUnauthorized, "UNAUTHORIZED", "unauthenticated request")
		return
	}
	userID := userIDVal.(string)

	if err := h.calendarService.Disconnect(c.Request.Context(), userID, provider); err != nil {
		RespondError(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	RespondSuccess(c, http.StatusOK, gin.H{"message": fmt.Sprintf("%s calendar disconnected successfully", provider)})
}

func (h *CalendarHandler) handleSyncError(c *gin.Context, err error) {
	if errors.Is(err, service.ErrCalendarNotConnected) {
		RespondError(c, http.StatusBadRequest, "CALENDAR_NOT_CONNECTED", "calendar provider is not connected")
		return
	}
	if errors.Is(err, calendar.ErrProviderNotConfigured) {
		RespondError(c, http.StatusBadRequest, "PROVIDER_NOT_CONFIGURED", "calendar provider is not configured")
		return
	}
	if errors.Is(err, calendar.ErrAuthExpired) {
		RespondError(c, http.StatusUnauthorized, "AUTH_EXPIRED", "calendar authorization expired; please reconnect your calendar")
		return
	}
	if errors.Is(err, calendar.ErrPermissionDenied) {
		RespondError(c, http.StatusForbidden, "PERMISSION_DENIED", "calendar permission denied by external provider")
		return
	}
	if errors.Is(err, calendar.ErrRateLimited) {
		RespondError(c, http.StatusTooManyRequests, "RATE_LIMITED", "rate limited by calendar provider; please try again later")
		return
	}

	RespondError(c, http.StatusInternalServerError, "SYNC_FAILED", err.Error())
}

func parseProvider(val string) (model.CalendarProviderType, error) {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "google":
		return model.CalendarProviderGoogle, nil
	case "microsoft", "outlook":
		return model.CalendarProviderMicrosoft, nil
	default:
		return "", fmt.Errorf("unsupported calendar provider: %s", val)
	}
}
