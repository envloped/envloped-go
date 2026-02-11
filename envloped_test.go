package envloped

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newTestClient creates a client pointed at the given httptest server.
func newTestClient(t *testing.T, server *httptest.Server) *Client {
	t.Helper()
	client := NewClient("test_api_key_123")
	client.WithBaseURL(server.URL)
	return client
}

func TestNewClient(t *testing.T) {
	t.Parallel()

	client := NewClient("ev_test_key")

	if client.apiKey != "ev_test_key" {
		t.Errorf("expected apiKey %q, got %q", "ev_test_key", client.apiKey)
	}
	if client.baseURL.String() != defaultBaseURL {
		t.Errorf("expected baseURL %q, got %q", defaultBaseURL, client.baseURL.String())
	}
	if client.Emails == nil {
		t.Error("expected Emails service to be initialized")
	}
}

func TestNewClient_TrimsWhitespace(t *testing.T) {
	t.Parallel()

	client := NewClient("  ev_test_key  ")
	if client.apiKey != "ev_test_key" {
		t.Errorf("expected trimmed apiKey %q, got %q", "ev_test_key", client.apiKey)
	}
}

func TestWithBaseURL(t *testing.T) {
	t.Parallel()

	client := NewClient("key").WithBaseURL("https://custom.example.com")
	if client.baseURL.String() != "https://custom.example.com" {
		t.Errorf("expected custom base URL, got %q", client.baseURL.String())
	}
}

func TestWithHTTPClient(t *testing.T) {
	t.Parallel()

	custom := &http.Client{Timeout: 5 * time.Second}
	client := NewClient("key").WithHTTPClient(custom)
	if client.httpClient != custom {
		t.Error("expected custom HTTP client to be set")
	}
}

func TestWithHTTPClient_Nil(t *testing.T) {
	t.Parallel()

	original := NewClient("key")
	originalClient := original.httpClient
	original.WithHTTPClient(nil)
	if original.httpClient != originalClient {
		t.Error("expected nil to be ignored, HTTP client should not change")
	}
}

func TestVersion(t *testing.T) {
	t.Parallel()

	if v := Version(); v != "1.0.0" {
		t.Errorf("expected version %q, got %q", "1.0.0", v)
	}
}

func TestPing_Success(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/ping" {
			t.Errorf("expected path /api/v1/ping, got %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test_api_key_123" {
			t.Errorf("unexpected Authorization header: %s", auth)
		}
		if ua := r.Header.Get("User-Agent"); ua != userAgent {
			t.Errorf("unexpected User-Agent header: %s", ua)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PingResponse{
			Message:   "pong",
			CompanyID: "company_123",
		})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	resp, err := client.Ping()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Message != "pong" {
		t.Errorf("expected message %q, got %q", "pong", resp.Message)
	}
	if resp.CompanyID != "company_123" {
		t.Errorf("expected companyId %q, got %q", "company_123", resp.CompanyID)
	}
}

func TestPing_Unauthorized(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid API key"})
	}))
	defer server.Close()

	client := newTestClient(t, server)
	_, err := client.Ping()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("expected status 401, got %d", apiErr.StatusCode)
	}
}

func TestPingWithContext_Cancellation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a slow response
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(t, server)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := client.PingWithContext(ctx)
	if err == nil {
		t.Fatal("expected error due to context cancellation, got nil")
	}
}

func TestAuthorizationHeader(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer my_secret_key" {
			t.Errorf("expected Authorization 'Bearer my_secret_key', got %q", auth)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PingResponse{Message: "pong"})
	}))
	defer server.Close()

	client := NewClient("my_secret_key").WithBaseURL(server.URL)
	_, err := client.Ping()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChainedBuilderMethods(t *testing.T) {
	t.Parallel()

	custom := &http.Client{Timeout: 10 * time.Second}
	client := NewClient("key").
		WithBaseURL("https://custom.example.com").
		WithHTTPClient(custom)

	if client.baseURL.String() != "https://custom.example.com" {
		t.Errorf("expected custom base URL, got %q", client.baseURL.String())
	}
	if client.httpClient != custom {
		t.Error("expected custom HTTP client")
	}
}
