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

type MicrosoftCalendarConfig struct {
	ClientID     string
	ClientSecret string
	TenantID     string
	RedirectURL  string
	AuthURL      string
	TokenURL     string
	GraphAPI     string
}

type microsoftCalendarProvider struct {
	cfg        MicrosoftCalendarConfig
	httpClient *http.Client
}

// NewMicrosoftCalendarProvider constructs a MicrosoftCalendarProvider.
func NewMicrosoftCalendarProvider(cfg MicrosoftCalendarConfig, client *http.Client) CalendarProvider {
	if client == nil {
		client = &http.Client{Timeout: 15 * time.Second}
	}
	tenant := strings.TrimSpace(cfg.TenantID)
	if tenant == "" {
		tenant = "common"
	}
	if cfg.AuthURL == "" {
		cfg.AuthURL = fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", tenant)
	}
	if cfg.TokenURL == "" {
		cfg.TokenURL = fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", tenant)
	}
	if cfg.GraphAPI == "" {
		cfg.GraphAPI = "https://graph.microsoft.com/v1.0"
	}

	return &microsoftCalendarProvider{
		cfg:        cfg,
		httpClient: client,
	}
}

func (p *microsoftCalendarProvider) Name() model.CalendarProviderType {
	return model.CalendarProviderMicrosoft
}

func (p *microsoftCalendarProvider) IsConfigured() bool {
	return strings.TrimSpace(p.cfg.ClientID) != "" && strings.TrimSpace(p.cfg.ClientSecret) != ""
}

func (p *microsoftCalendarProvider) GetAuthURL(state string) string {
	if !p.IsConfigured() {
		return ""
	}

	params := url.Values{}
	params.Set("client_id", p.cfg.ClientID)
	params.Set("response_type", "code")
	params.Set("redirect_uri", p.cfg.RedirectURL)
	params.Set("response_mode", "query")
	params.Set("scope", "Calendars.ReadWrite offline_access User.Read openid email profile")
	params.Set("state", state)

	return fmt.Sprintf("%s?%s", p.cfg.AuthURL, params.Encode())
}

func (p *microsoftCalendarProvider) ExchangeCode(ctx context.Context, code string) (*TokenResponse, *UserInfo, error) {
	if !p.IsConfigured() {
		return nil, nil, ErrProviderNotConfigured
	}

	data := url.Values{}
	data.Set("client_id", p.cfg.ClientID)
	data.Set("client_secret", p.cfg.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", p.cfg.RedirectURL)
	data.Set("grant_type", "authorization_code")
	data.Set("scope", "Calendars.ReadWrite offline_access User.Read openid email profile")

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

	userInfo, err := p.fetchUserInfo(ctx, tokenResp.AccessToken)
	if err != nil {
		return nil, nil, err
	}

	return &tokenResp, userInfo, nil
}

func (p *microsoftCalendarProvider) fetchUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	reqURL := fmt.Sprintf("%s/me", p.cfg.GraphAPI)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create me request: %w", err)
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
		ID                string `json:"id"`
		Mail              string `json:"mail"`
		UserPrincipalName string `json:"userPrincipalName"`
		DisplayName       string `json:"displayName"`
	}
	if err := json.Unmarshal(bodyBytes, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse me info: %w", err)
	}

	email := raw.Mail
	if email == "" {
		email = raw.UserPrincipalName
	}

	return &UserInfo{
		ID:    raw.ID,
		Email: email,
		Name:  raw.DisplayName,
	}, nil
}

func (p *microsoftCalendarProvider) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	if !p.IsConfigured() {
		return nil, ErrProviderNotConfigured
	}

	data := url.Values{}
	data.Set("client_id", p.cfg.ClientID)
	data.Set("client_secret", p.cfg.ClientSecret)
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")
	data.Set("scope", "Calendars.ReadWrite offline_access User.Read openid email profile")

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

func (p *microsoftCalendarProvider) CreateEvent(ctx context.Context, accessToken string, event *CalendarEventData) (string, error) {
	if !p.IsConfigured() {
		return "", ErrProviderNotConfigured
	}

	payload := p.buildEventPayload(event)
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to encode event payload: %w", err)
	}

	targetURL := fmt.Sprintf("%s/me/calendar/events", p.cfg.GraphAPI)
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
		return "", fmt.Errorf("failed to parse event ID from Microsoft response: %w", err)
	}

	return result.ID, nil
}

func (p *microsoftCalendarProvider) UpdateEvent(ctx context.Context, accessToken string, externalEventID string, event *CalendarEventData) error {
	if !p.IsConfigured() {
		return ErrProviderNotConfigured
	}

	payload := p.buildEventPayload(event)
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to encode event payload: %w", err)
	}

	targetURL := fmt.Sprintf("%s/me/events/%s", p.cfg.GraphAPI, url.PathEscape(externalEventID))
	req, err := http.NewRequestWithContext(ctx, "PATCH", targetURL, bytes.NewReader(jsonBytes))
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

func (p *microsoftCalendarProvider) DeleteEvent(ctx context.Context, accessToken string, externalEventID string) error {
	if !p.IsConfigured() {
		return ErrProviderNotConfigured
	}

	targetURL := fmt.Sprintf("%s/me/events/%s", p.cfg.GraphAPI, url.PathEscape(externalEventID))
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
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
		return nil
	}

	return p.mapHTTPError(resp.StatusCode, bodyBytes)
}

func (p *microsoftCalendarProvider) buildEventPayload(event *CalendarEventData) map[string]interface{} {
	tz := event.Timezone
	if tz == "" {
		tz = "Asia/Jakarta"
	}

	if parsedLoc, err := time.LoadLocation(tz); err == nil {
		event.Start = event.Start.In(parsedLoc)
		event.End = event.End.In(parsedLoc)
	}

	payload := map[string]interface{}{
		"subject": event.Title,
		"body": map[string]string{
			"contentType": "Text",
			"content":     event.Description,
		},
		"start": map[string]string{
			"dateTime": event.Start.Format("2006-01-02T15:04:05"),
			"timeZone": tz,
		},
		"end": map[string]string{
			"dateTime": event.End.Format("2006-01-02T15:04:05"),
			"timeZone": tz,
		},
	}

	locationName := ""
	if strings.TrimSpace(event.Location) != "" {
		locationName = strings.TrimSpace(event.Location)
	} else if strings.TrimSpace(event.MeetingURL) != "" {
		locationName = strings.TrimSpace(event.MeetingURL)
	}
	if locationName != "" {
		payload["location"] = map[string]string{
			"displayName": locationName,
		}
	}

	if len(event.Attendees) > 0 {
		var attendees []map[string]interface{}
		for _, a := range event.Attendees {
			if strings.TrimSpace(a.Email) != "" {
				attendees = append(attendees, map[string]interface{}{
					"emailAddress": map[string]string{
						"address": strings.TrimSpace(a.Email),
						"name":    strings.TrimSpace(a.Name),
					},
					"type": "required",
				})
			}
		}
		if len(attendees) > 0 {
			payload["attendees"] = attendees
		}
	}

	return payload
}

func (p *microsoftCalendarProvider) mapHTTPError(statusCode int, body []byte) error {
	switch statusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("%w: unauthorized from Microsoft Graph", ErrAuthExpired)
	case http.StatusForbidden:
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
