// Package envloped provides a Go client for the Envloped email API.
//
// Usage:
//
//	client := envloped.NewClient("ev_your_api_key")
//	resp, err := client.Emails.Send(&envloped.SendEmailRequest{
//	    From:    "hello@yourdomain.com",
//	    To:      []string{"user@example.com"},
//	    Subject: "Hello from Envloped",
//	    Html:    "<p>Welcome!</p>",
//	})
package envloped

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
)

const (
	// version is the current SDK version. Keep in sync with Git tags.
	version = "1.0.0"

	// userAgent is sent with every request for server-side tracking.
	userAgent = "envloped-go/" + version

	// contentType is the Content-Type header value for all requests.
	contentType = "application/json"

	// defaultBaseURL is the production Envloped API endpoint.
	defaultBaseURL = "https://api.envloped.com"
)

// Client handles communication with the Envloped API.
type Client struct {
	// httpClient is the underlying HTTP client used for requests.
	httpClient *http.Client

	// apiKey is the Bearer token for authentication.
	apiKey string

	// baseURL is the API base URL (without trailing slash).
	baseURL *url.URL

	// userAgent is the User-Agent header value.
	userAgent string

	// Emails provides access to the email sending API.
	Emails EmailsSvc
}

// NewClient creates a new Envloped API client with the given API key.
// The client defaults to the production API at https://api.envloped.com.
func NewClient(apiKey string) *Client {
	key := strings.TrimSpace(apiKey)
	baseURL, _ := url.Parse(defaultBaseURL)

	c := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     key,
		baseURL:    baseURL,
		userAgent:  userAgent,
	}

	c.Emails = &emailsSvcImpl{client: c}

	return c
}

// WithBaseURL sets a custom base URL for the API client.
// This is useful for testing or self-hosted deployments.
// Returns the client for method chaining.
func (c *Client) WithBaseURL(rawURL string) *Client {
	u, err := url.Parse(rawURL)
	if err == nil {
		c.baseURL = u
	}
	return c
}

// WithHTTPClient sets a custom HTTP client for the API client.
// This is useful for configuring timeouts, transports, or proxies.
// Returns the client for method chaining.
func (c *Client) WithHTTPClient(httpClient *http.Client) *Client {
	if httpClient != nil {
		c.httpClient = httpClient
	}
	return c
}

// PingResponse is the response from the Ping endpoint.
type PingResponse struct {
	Message   string `json:"message"`
	CompanyID string `json:"companyId"`
}

// Ping checks connectivity and API key validity.
// Returns a PingResponse on success or an error on failure.
func (c *Client) Ping() (*PingResponse, error) {
	return c.PingWithContext(context.Background())
}

// PingWithContext checks connectivity and API key validity using the given context.
func (c *Client) PingWithContext(ctx context.Context) (*PingResponse, error) {
	req, err := c.newRequest(ctx, http.MethodGet, "/api/v1/ping", nil)
	if err != nil {
		return nil, fmt.Errorf("envloped: failed to create ping request: %w", err)
	}

	var resp PingResponse
	if err := c.do(req, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// Version returns the SDK version string.
func Version() string {
	return version
}

// newRequest builds a new HTTP request with authentication and standard headers.
func (c *Client) newRequest(ctx context.Context, method, path string, body interface{}) (*http.Request, error) {
	u, err := c.baseURL.Parse(path)
	if err != nil {
		return nil, fmt.Errorf("invalid path %q: %w", path, err)
	}

	var reqBody io.Reader
	if body != nil {
		buf := new(bytes.Buffer)
		if err := json.NewEncoder(buf).Encode(body); err != nil {
			return nil, fmt.Errorf("failed to encode request body: %w", err)
		}
		reqBody = buf
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), reqBody)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Accept", contentType)

	if body != nil {
		req.Header.Set("Content-Type", contentType)
	}

	return req, nil
}

// do executes the request and decodes the response body into target.
// If the response status is not 2xx, it returns a typed error.
func (c *Client) do(req *http.Request, target interface{}) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("envloped: request failed: %w", err)
	}

	// Handle non-2xx responses.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return handleErrorResponse(resp)
	}

	defer resp.Body.Close()

	if target != nil && resp.StatusCode != http.StatusNoContent {
		if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
			return fmt.Errorf("envloped: failed to decode response: %w", err)
		}
	}

	return nil
}
