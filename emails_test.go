package envloped

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSendEmail_Success(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/emails" {
			t.Errorf("expected path /v1/emails, got %s", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != contentType {
			t.Errorf("expected Content-Type %q, got %q", contentType, ct)
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read request body: %v", err)
		}
		defer r.Body.Close()

		var req SendEmailRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("failed to unmarshal request body: %v", err)
		}

		if req.From != "sender@example.com" {
			t.Errorf("expected from %q, got %q", "sender@example.com", req.From)
		}
		if len(req.To) != 1 || req.To[0] != "recipient@example.com" {
			t.Errorf("unexpected to: %v", req.To)
		}
		if req.Subject != "Test Subject" {
			t.Errorf("expected subject %q, got %q", "Test Subject", req.Subject)
		}
		if req.Html != "<p>Hello</p>" {
			t.Errorf("expected html %q, got %q", "<p>Hello</p>", req.Html)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SendEmailResponse{
			Success:   true,
			MessageId: "msg_abc123",
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	resp, err := client.Emails.Send(&SendEmailRequest{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test Subject",
		Html:    "<p>Hello</p>",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Error("expected success to be true")
	}
	if resp.MessageId != "msg_abc123" {
		t.Errorf("expected messageId %q, got %q", "msg_abc123", resp.MessageId)
	}
}

func TestSendEmail_TextBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req SendEmailRequest
		json.Unmarshal(body, &req)

		if req.Text != "Plain text content" {
			t.Errorf("expected text %q, got %q", "Plain text content", req.Text)
		}
		if req.Html != "" {
			t.Errorf("expected empty html, got %q", req.Html)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SendEmailResponse{Success: true, MessageId: "msg_text"})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	resp, err := client.Emails.Send(&SendEmailRequest{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test",
		Text:    "Plain text content",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.MessageId != "msg_text" {
		t.Errorf("expected messageId %q, got %q", "msg_text", resp.MessageId)
	}
}

func TestSendEmail_MultipleRecipients(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req SendEmailRequest
		json.Unmarshal(body, &req)

		if len(req.To) != 3 {
			t.Errorf("expected 3 recipients, got %d", len(req.To))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SendEmailResponse{Success: true, MessageId: "msg_multi"})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	_, err := client.Emails.Send(&SendEmailRequest{
		From:    "sender@example.com",
		To:      []string{"a@example.com", "b@example.com", "c@example.com"},
		Subject: "Test",
		Html:    "<p>Hi</p>",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// Table-driven validation tests
func TestSendEmail_Validation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		params  *SendEmailRequest
		wantErr string
	}{
		{
			name:    "nil params",
			params:  nil,
			wantErr: "params must not be nil",
		},
		{
			name:    "missing from",
			params:  &SendEmailRequest{To: []string{"a@b.com"}, Subject: "s", Html: "<p>x</p>"},
			wantErr: "from address is required",
		},
		{
			name:    "missing to",
			params:  &SendEmailRequest{From: "a@b.com", Subject: "s", Html: "<p>x</p>"},
			wantErr: "at least one to address is required",
		},
		{
			name:    "empty to slice",
			params:  &SendEmailRequest{From: "a@b.com", To: []string{}, Subject: "s", Html: "<p>x</p>"},
			wantErr: "at least one to address is required",
		},
		{
			name:    "missing subject",
			params:  &SendEmailRequest{From: "a@b.com", To: []string{"b@c.com"}, Html: "<p>x</p>"},
			wantErr: "subject is required",
		},
		{
			name:    "missing body",
			params:  &SendEmailRequest{From: "a@b.com", To: []string{"b@c.com"}, Subject: "s"},
			wantErr: "html or text body is required",
		},
	}

	// Validation happens before any HTTP call, so no server needed.
	client := NewClient("key")

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := client.Emails.Send(tt.params)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if got := err.Error(); !contains(got, tt.wantErr) {
				t.Errorf("expected error to contain %q, got %q", tt.wantErr, got)
			}
		})
	}
}

// Table-driven API error tests
func TestSendEmail_APIErrors(t *testing.T) {
	t.Parallel()

	validParams := &SendEmailRequest{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test",
		Html:    "<p>Hello</p>",
	}

	tests := []struct {
		name         string
		statusCode   int
		responseBody interface{}
		sentinelErr  error
		errType      string // "api", "validation", "ratelimit"
	}{
		{
			name:         "400 validation error",
			statusCode:   http.StatusBadRequest,
			responseBody: map[string]string{"error": "from address is required"},
			sentinelErr:  ErrValidation,
			errType:      "validation",
		},
		{
			name:         "401 unauthorized",
			statusCode:   http.StatusUnauthorized,
			responseBody: map[string]string{"error": "Invalid API key"},
			sentinelErr:  ErrUnauthorized,
			errType:      "api",
		},
		{
			name:         "403 forbidden domain",
			statusCode:   http.StatusForbidden,
			responseBody: map[string]string{"error": "Domain 'test.com' is not registered"},
			sentinelErr:  ErrForbidden,
			errType:      "api",
		},
		{
			name:       "429 rate limit",
			statusCode: http.StatusTooManyRequests,
			responseBody: map[string]interface{}{
				"error":   "Rate limit exceeded",
				"message": "Monthly email limit reached (4000 emails).",
				"usage": map[string]interface{}{
					"dailyCount":   150,
					"monthlyCount": 4000,
					"dailyLimit":   200,
					"monthlyLimit": 4000,
				},
			},
			sentinelErr: ErrRateLimited,
			errType:     "ratelimit",
		},
		{
			name:         "500 server error",
			statusCode:   http.StatusInternalServerError,
			responseBody: map[string]string{"error": "Failed to send email", "details": "SES timeout"},
			sentinelErr:  nil, // no sentinel for 500
			errType:      "api",
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(tt.responseBody)
			}))
			defer server.Close()

			client := newTestClient(t, server)
			_, err := client.Emails.Send(validParams)
			if err == nil {
				t.Fatal("expected error, got nil")
			}

			// Check sentinel error
			if tt.sentinelErr != nil {
				if !errors.Is(err, tt.sentinelErr) {
					t.Errorf("expected errors.Is(%v) to be true, got false. Error: %v", tt.sentinelErr, err)
				}
			}

			// Check error type
			switch tt.errType {
			case "validation":
				var ve *ValidationError
				if !errors.As(err, &ve) {
					t.Errorf("expected *ValidationError, got %T: %v", err, err)
				}
			case "ratelimit":
				var rle *RateLimitError
				if !errors.As(err, &rle) {
					t.Errorf("expected *RateLimitError, got %T: %v", err, err)
				} else {
					if rle.Reason == "" {
						t.Error("expected rate limit reason to be populated")
					}
					if rle.Usage == nil {
						t.Error("expected rate limit usage to be populated")
					} else if rle.Usage.MonthlyLimit != 4000 {
						t.Errorf("expected monthly limit 4000, got %d", rle.Usage.MonthlyLimit)
					}
				}
			case "api":
				var apiErr *APIError
				if !errors.As(err, &apiErr) {
					t.Errorf("expected *APIError, got %T: %v", err, err)
				} else if apiErr.StatusCode != tt.statusCode {
					t.Errorf("expected status %d, got %d", tt.statusCode, apiErr.StatusCode)
				}
			}
		})
	}
}

func TestSendEmailWithContext_Cancellation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(t, server)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.Emails.SendWithContext(ctx, &SendEmailRequest{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Test",
		Html:    "<p>Hello</p>",
	})
	if err == nil {
		t.Fatal("expected error due to context cancellation, got nil")
	}
}

// contains checks if s contains substr (simple helper to avoid importing strings).
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
