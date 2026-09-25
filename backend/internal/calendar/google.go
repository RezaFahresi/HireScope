package calendar

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"hirescope/backend/internal/model"
)

type GoogleCalendarConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
	CalendarAPI  string
}

type googleCalendarProvider struct {
	cfg        GoogleCalendarConfig
	httpClient *http.Client
}

// NewGoogleCalendarProvider constructs a GoogleCalendarProvider.
func NewGoogleCalendarProvider(cfg GoogleCalendarConfig, client *http.Client) CalendarProvider {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	if cfg.AuthURL == "" {
		cfg.AuthURL = "https://accounts.google.com/o/oauth2/v2/auth"
	}
	if cfg.TokenURL == "" {
		cfg.TokenURL = "https://oauth2.googleapis.com/token"
	}
	if cfg.UserInfoURL == "" {
		cfg.UserInfoURL = "https://www.googleapis.com/oauth2/v2/userinfo"
	}
	if cfg.CalendarAPI == "" {
		cfg.CalendarAPI = "https://www.googleapis.com/calendar/v3"
	}

	return &googleCalendarProvider{
		cfg:        cfg,
		httpClient: client,
	}
}

func (p *googleCalendarProvider) Name() model.CalendarProviderType {
	return model.CalendarProviderGoogle
}

func (p *googleCalendarProvider) IsConfigured() bool {
	return strings.TrimSpace(p.cfg.ClientID) != "" && strings.TrimSpace(p.cfg.ClientSecret) != ""
}

func (p *googleCalendarProvider) GetAuthURL(state string) string {
	if !p.IsConfigured() {
		return ""
	}

	params := url.Values{}
	params.Set("client_id", p.cfg.ClientID)
	params.Set("redirect_uri", p.cfg.RedirectURL)
	params.Set("response_type", "code")
	params.Set("scope", "https://www.googleapis.com/auth/calendar.events https://www.googleapis.com/auth/userinfo.email https://www.googleapis.com/auth/userinfo.profile")
	params.Set("access_type", "offline")
	params.Set("prompt", "consent")
	params.Set("state", state)

	return fmt.Sprintf("%s?%s", p.cfg.AuthURL, params.Encode())
}

func (p *googleCalendarProvider) ExchangeCode(ctx context.Context, code string) (*TokenResponse, *UserInfo, error) {
	if !p.IsConfigured() {
		return nil, nil, ErrProviderNotConfigured
	}

	data := url.Values{}
	data.Set("code", code)
	data.Set("client_id", p.cfg.ClientID)
	data.Set("client_secret", p.cfg.ClientSecret)
	data.Set("redirect_uri", p.cfg.RedirectURL)
	data.Set("grant_type", "authorization_code")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, nil, p.mapHTTPError(resp.StatusCode, bodyBytes)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(bodyBytes, &tokenResp); err != nil {
		return nil, nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	// Fetch user info with access token
	userInfo, err := p.fetchUserInfo(ctx, tokenResp.AccessToken)
	if err != nil {
		return nil, nil, err
	}

	return &tokenResp, userInfo, nil
}

func (p *googleCalendarProvider) fetchUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.cfg.UserInfoURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create userinfo request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, p.mapHTTPError(resp.StatusCode, bodyBytes)
	}

	var raw struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	if err := json.Unmarshal(bodyBytes, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse userinfo: %w", err)
	}

	return &UserInfo{
		ID:    raw.ID,
		Email: raw.Email,
		Name:  raw.Name,
	}, nil
}

func (p *googleCalendarProvider) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	if !p.IsConfigured() {
		return nil, ErrProviderNotConfigured
	}

	data := url.Values{}
	data.Set("refresh_token", refreshToken)
	data.Set("client_id", p.cfg.ClientID)
	data.Set("client_secret", p.cfg.ClientSecret)
	data.Set("grant_type", "refresh_token")

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("failed to create refresh token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, p.mapHTTPError(resp.StatusCode, bodyBytes)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(bodyBytes, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse refresh token response: %w", err)
	}

	return &tokenResp, nil
}

func (p *googleCalendarProvider) CreateEvent(ctx context.Context, accessToken string, event *CalendarEventData) (string, error) {
	if !p.IsConfigured() {
		return "", ErrProviderNotConfigured
	}

	payload := p.buildEventPayload(event)
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to encode event payload: %w", err)
	}

	targetURL := fmt.Sprintf("%s/calendars/primary/events", p.cfg.CalendarAPI)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return "", fmt.Errorf("failed to create event request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", p.mapHTTPError(resp.StatusCode, bodyBytes)
	}

	var result struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(bodyBytes, &result); err != nil || result.ID == "" {
		return "", fmt.Errorf("failed to parse event ID from Google response: %w", err)
	}

	return result.ID, nil
}

func (p *googleCalendarProvider) UpdateEvent(ctx context.Context, accessToken string, externalEventID string, event *CalendarEventData) error {
	if !p.IsConfigured() {
		return ErrProviderNotConfigured
	}

	payload := p.buildEventPayload(event)
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to encode event payload: %w", err)
	}

	targetURL := fmt.Sprintf("%s/calendars/primary/events/%s", p.cfg.CalendarAPI, url.PathEscape(externalEventID))
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, targetURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return fmt.Errorf("failed to create update request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return p.mapHTTPError(resp.StatusCode, bodyBytes)
	}

	return nil
}

func (p *googleCalendarProvider) DeleteEvent(ctx context.Context, accessToken string, externalEventID string) error {
	if !p.IsConfigured() {
		return ErrProviderNotConfigured
	}

	targetURL := fmt.Sprintf("%s/calendars/primary/events/%s", p.cfg.CalendarAPI, url.PathEscape(externalEventID))
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, targetURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrProviderUnavailable, err)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	// 204 No Content or 200 OK or 410 Gone / 404 are acceptable outcomes for deletion
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone || resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
		return nil
	}

	return p.mapHTTPError(resp.StatusCode, bodyBytes)
}

func (p *googleCalendarProvider) buildEventPayload(event *CalendarEventData) map[string]interface{} {
	tz := event.Timezone
	if tz == "" {
		tz = "Asia/Jakarta"
	}

	payload := map[string]interface{}{
		"summary":     event.Title,
		"description": event.Description,
		"start": map[string]string{
			"dateTime": event.Start.Format(time.RFC3339),
			"timeZone": tz,
		},
		"end": map[string]string{
			"dateTime": event.End.Format(time.RFC3339),
			"timeZone": tz,
		},
	}

	if strings.TrimSpace(event.Location) != "" {
		payload["location"] = strings.TrimSpace(event.Location)
	} else if strings.TrimSpace(event.MeetingURL) != "" {
		payload["location"] = strings.TrimSpace(event.MeetingURL)
	}

	if len(event.Attendees) > 0 {
		var attendees []map[string]string
		for _, a := range event.Attendees {
			if strings.TrimSpace(a.Email) != "" {
				attendees = append(attendees, map[string]string{
					"email":       strings.TrimSpace(a.Email),
					"displayName": strings.TrimSpace(a.Name),
				})
			}
		}
		if len(attendees) > 0 {
			payload["attendees"] = attendees
		}
	}

	return payload
}

func (p *googleCalendarProvider) mapHTTPError(statusCode int, body []byte) error {
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("%w: unauthorized from Google", ErrAuthExpired)
	case http.StatusForbidden:
		if strings.Contains(string(body), "rateLimitExceeded") {
			return ErrRateLimited
		}
		return ErrPermissionDenied
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusTooManyRequests:
		return ErrRateLimited
	case http.StatusBadRequest:
		return fmt.Errorf("%w: %s", ErrInvalidEvent, string(body))
	default:
		if statusCode >= 500 {
			return ErrProviderUnavailable
		}
		return fmt.Errorf("%w (HTTP %d): %s", ErrUnknownProviderError, statusCode, string(body))
	}
}
