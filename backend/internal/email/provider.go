package email

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	ErrEmailDisabled         = errors.New("email delivery is disabled in this environment (EMAIL_ENABLED=false)")
	ErrMissingRecipientEmail = errors.New("recipient email address cannot be blank")
	ErrMissingSubject        = errors.New("email subject cannot be blank")
	ErrMissingContent        = errors.New("email content (HTML or plain text) cannot be blank")
	ErrProviderFailed        = errors.New("email provider call failed")
)

// EmailMessage contains the payload for an outgoing transactional email.
type EmailMessage struct {
	From           string
	To             string
	ReplyTo        string
	Subject        string
	HTML           string
	Text           string
	IdempotencyKey string
}

// EmailResult encapsulates the outcome of an email transmission attempt.
type EmailResult struct {
	ProviderMessageID string
	Provider          string
	Status            string // e.g. "SENT", "SKIPPED", "FAILED"
}

// EmailProvider defines the abstract interface for transactional email delivery.
type EmailProvider interface {
	Send(ctx context.Context, msg EmailMessage) (*EmailResult, error)
}

// ResendEmailProvider delivers transactional emails via Resend's REST API.
type ResendEmailProvider struct {
	apiKey     string
	apiURL     string
	httpClient *http.Client
}

// NewResendEmailProvider creates a new Resend provider instance.
func NewResendEmailProvider(apiKey string) *ResendEmailProvider {
	return &ResendEmailProvider{
		apiKey: apiKey,
		apiURL: "https://api.resend.com/emails",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SetAPIURL overrides the default endpoint (useful for testing).
func (p *ResendEmailProvider) SetAPIURL(url string) {
	p.apiURL = url
}

// SetHTTPClient overrides the HTTP client (useful for mock transports).
func (p *ResendEmailProvider) SetHTTPClient(client *http.Client) {
	p.httpClient = client
}

type resendRequestPayload struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	ReplyTo *string  `json:"reply_to,omitempty"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html,omitempty"`
	Text    string   `json:"text,omitempty"`
}

type resendSuccessResponse struct {
	ID string `json:"id"`
}

type resendErrorResponse struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"message"`
	Name       string `json:"name"`
}

// Send dispatches the email to the Resend API with HTTP header idempotency support.
func (p *ResendEmailProvider) Send(ctx context.Context, msg EmailMessage) (*EmailResult, error) {
	if strings.TrimSpace(msg.To) == "" {
		return nil, ErrMissingRecipientEmail
	}
	if strings.TrimSpace(msg.Subject) == "" {
		return nil, ErrMissingSubject
	}
	if strings.TrimSpace(msg.HTML) == "" && strings.TrimSpace(msg.Text) == "" {
		return nil, ErrMissingContent
	}

	payload := resendRequestPayload{
		From:    msg.From,
		To:      []string{strings.TrimSpace(msg.To)},
		Subject: msg.Subject,
		HTML:    msg.HTML,
		Text:    msg.Text,
	}
	if strings.TrimSpace(msg.ReplyTo) != "" {
		replyTo := strings.TrimSpace(msg.ReplyTo)
		payload.ReplyTo = &replyTo
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode resend payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.apiURL, bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(msg.IdempotencyKey) != "" {
		req.Header.Set("Idempotency-Key", strings.TrimSpace(msg.IdempotencyKey))
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error during resend request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read resend response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp resendErrorResponse
		if jsonErr := json.Unmarshal(bodyBytes, &errResp); jsonErr == nil && errResp.Message != "" {
			return nil, fmt.Errorf("%w: status %d (%s: %s)", ErrProviderFailed, resp.StatusCode, errResp.Name, errResp.Message)
		}
		return nil, fmt.Errorf("%w: status %d (body: %s)", ErrProviderFailed, resp.StatusCode, string(bodyBytes))
	}

	var successResp resendSuccessResponse
	if err := json.Unmarshal(bodyBytes, &successResp); err != nil {
		return nil, fmt.Errorf("failed to parse resend success response: %w", err)
	}

	return &EmailResult{
		ProviderMessageID: successResp.ID,
		Provider:          "resend",
		Status:            "SENT",
	}, nil
}

// DisabledEmailProvider is used when EMAIL_ENABLED=false.
type DisabledEmailProvider struct{}

// NewDisabledEmailProvider returns a new disabled email provider.
func NewDisabledEmailProvider() *DisabledEmailProvider {
	return &DisabledEmailProvider{}
}

// Send records the delivery as SKIPPED without contacting external networks.
func (p *DisabledEmailProvider) Send(ctx context.Context, msg EmailMessage) (*EmailResult, error) {
	return &EmailResult{
		ProviderMessageID: "",
		Provider:          "disabled",
		Status:            "SKIPPED",
	}, ErrEmailDisabled
}

// MockEmailProvider is an in-memory provider for unit and integration testing.
type MockEmailProvider struct {
	mu           sync.Mutex
	SentMessages []EmailMessage
	SimulateErr  error
	CallCount    int
}

// NewMockEmailProvider returns a new MockEmailProvider.
func NewMockEmailProvider() *MockEmailProvider {
	return &MockEmailProvider{
		SentMessages: make([]EmailMessage, 0),
	}
}

// Send records the message in-memory and returns simulated results.
func (p *MockEmailProvider) Send(ctx context.Context, msg EmailMessage) (*EmailResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.CallCount++
	if p.SimulateErr != nil {
		return nil, p.SimulateErr
	}

	p.SentMessages = append(p.SentMessages, msg)
	return &EmailResult{
		ProviderMessageID: fmt.Sprintf("mock-msg-%d", len(p.SentMessages)),
		Provider:          "mock",
		Status:            "SENT",
	}, nil
}

// GetSentMessages returns a thread-safe copy of sent messages.
func (p *MockEmailProvider) GetSentMessages() []EmailMessage {
	p.mu.Lock()
	defer p.mu.Unlock()
	copied := make([]EmailMessage, len(p.SentMessages))
	copy(copied, p.SentMessages)
	return copied
}

// Reset clears recorded messages and simulated errors.
func (p *MockEmailProvider) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.SentMessages = make([]EmailMessage, 0)
	p.SimulateErr = nil
	p.CallCount = 0
}
