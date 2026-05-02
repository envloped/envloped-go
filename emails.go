package envloped

import (
	"context"
	"fmt"
	"net/http"
)

// Attachment represents a file attachment to include with an email.
type Attachment struct {
	// Filename is the name of the attached file (e.g., "invoice.pdf"). Required.
	Filename string `json:"filename"`

	// Content is the base64-encoded content of the file. Required.
	Content string `json:"content"`

	// ContentType is the MIME type of the attachment (e.g., "application/pdf", "text/calendar").
	// Optional — omitted if empty.
	ContentType string `json:"contentType,omitempty"`
}

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

	// Attachments is an optional list of file attachments. At most 10 attachments;
	// combined decoded payload must not exceed 40 MB (validated here and by the API).
	Attachments []Attachment `json:"attachments,omitempty"`
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
	if err := validateAttachments(params.Attachments); err != nil {
		return err
	}
	return nil
}

const (
	maxAttachments     = 10
	maxAttachmentBytes = 40 * 1024 * 1024 // 40 MB decoded (approximate from base64 length)
)

// validateAttachments checks attachment count, required fields, and estimated total
// decoded size from base64 content. The API enforces the same limit; this avoids
// sending oversized requests.
func validateAttachments(attachments []Attachment) error {
	if len(attachments) == 0 {
		return nil
	}
	if len(attachments) > maxAttachments {
		return fmt.Errorf("envloped: maximum %d attachments allowed, got %d", maxAttachments, len(attachments))
	}
	var totalDecoded int
	for i, a := range attachments {
		if a.Filename == "" {
			return fmt.Errorf("envloped: attachment[%d] must have a filename", i)
		}
		if a.Content == "" {
			return fmt.Errorf("envloped: attachment[%d] must have base64-encoded content", i)
		}
		// Approximate decoded size: standard base64 encodes every 3 bytes as 4 chars.
		totalDecoded += len(a.Content) * 3 / 4
	}
	if totalDecoded > maxAttachmentBytes {
		return fmt.Errorf("envloped: total attachment size must not exceed 40 MB")
	}
	return nil
}
