package envloped

import (
	"context"
	"fmt"
	"net/http"
)

// SendEmailRequest is the request body for sending an email.
//
// See https://docs.envloped.com/api-reference/emails/send-email
type SendEmailRequest struct {
	// From is the sender email address (e.g., "hello@yourdomain.com" or "My App <hello@yourdomain.com>").
	// The domain must be verified in your Envloped dashboard.
	From string `json:"from"`

	// To is the list of recipient email addresses.
	To []string `json:"to"`

	// Subject is the email subject line.
	Subject string `json:"subject"`

	// Html is the HTML body of the email. At least one of Html or Text must be provided.
	Html string `json:"html,omitempty"`

	// Text is the plain text body of the email. At least one of Html or Text must be provided.
	Text string `json:"text,omitempty"`
}

// SendEmailResponse is the response from a successful email send.
type SendEmailResponse struct {
	// Success indicates whether the email was sent successfully.
	Success bool `json:"success"`

	// MessageId is the unique identifier for the sent email (SES Message ID).
	MessageId string `json:"messageId"`
}

// EmailsSvc defines the interface for the email sending service.
// This interface can be mocked in consumer tests.
type EmailsSvc interface {
	// Send sends an email with the given parameters.
	Send(params *SendEmailRequest) (*SendEmailResponse, error)

	// SendWithContext sends an email using the provided context for cancellation and deadlines.
	SendWithContext(ctx context.Context, params *SendEmailRequest) (*SendEmailResponse, error)
}

// emailsSvcImpl implements EmailsSvc.
type emailsSvcImpl struct {
	client *Client
}

// Send sends an email with the given parameters.
// It validates required fields before making the API call.
func (s *emailsSvcImpl) Send(params *SendEmailRequest) (*SendEmailResponse, error) {
	return s.SendWithContext(context.Background(), params)
}

// SendWithContext sends an email using the provided context.
func (s *emailsSvcImpl) SendWithContext(ctx context.Context, params *SendEmailRequest) (*SendEmailResponse, error) {
	if err := validateSendEmailRequest(params); err != nil {
		return nil, err
	}

	req, err := s.client.newRequest(ctx, http.MethodPost, "/v1/emails", params)
	if err != nil {
		return nil, fmt.Errorf("envloped: failed to create send email request: %w", err)
	}

	var resp SendEmailResponse
	if err := s.client.do(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// validateSendEmailRequest checks that all required fields are present
// before making the API call, so the user gets immediate client-side feedback.
func validateSendEmailRequest(params *SendEmailRequest) error {
	if params == nil {
		return fmt.Errorf("envloped: send email params must not be nil")
	}
	if params.From == "" {
		return fmt.Errorf("envloped: from address is required")
	}
	if len(params.To) == 0 {
		return fmt.Errorf("envloped: at least one to address is required")
	}
	if params.Subject == "" {
		return fmt.Errorf("envloped: subject is required")
	}
	if params.Html == "" && params.Text == "" {
		return fmt.Errorf("envloped: html or text body is required")
	}
	return nil
}
