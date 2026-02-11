// Package envloped provides a Go client for the Envloped email API.
package envloped

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Sentinel errors for use with errors.Is().
var (
	// ErrUnauthorized is returned when the API key is missing or invalid (HTTP 401).
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden is returned when the request is not allowed (HTTP 403).
	// This typically means the domain is not registered or not verified.
	ErrForbidden = errors.New("forbidden")

	// ErrRateLimited is returned when usage limits have been exceeded (HTTP 429).
	ErrRateLimited = errors.New("rate limit exceeded")

	// ErrValidation is returned when the request body is invalid (HTTP 400).
	ErrValidation = errors.New("validation error")
)

// APIError represents a generic error response from the Envloped API.
type APIError struct {
	// StatusCode is the HTTP status code returned by the API.
	StatusCode int `json:"statusCode"`

	// Message is the primary error message.
	Message string `json:"error"`

	// Details provides additional context about the error (present on 500 responses).
	Details string `json:"details,omitempty"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("envloped: %s (status %d): %s", e.Message, e.StatusCode, e.Details)
	}
	return fmt.Sprintf("envloped: %s (status %d)", e.Message, e.StatusCode)
}

// Is enables sentinel error matching via errors.Is().
func (e *APIError) Is(target error) bool {
	switch {
	case target == ErrUnauthorized:
		return e.StatusCode == http.StatusUnauthorized
	case target == ErrForbidden:
		return e.StatusCode == http.StatusForbidden
	case target == ErrRateLimited:
		return e.StatusCode == http.StatusTooManyRequests
	case target == ErrValidation:
		return e.StatusCode == http.StatusBadRequest
	default:
		return false
	}
}

// EmailUsage contains email usage counters and limits returned with rate limit errors.
type EmailUsage struct {
	DailyCount   int  `json:"dailyCount"`
	MonthlyCount int  `json:"monthlyCount"`
	DailyLimit   *int `json:"dailyLimit"` // nil means unlimited
	MonthlyLimit int  `json:"monthlyLimit"`
}

// RateLimitError is returned when the API responds with HTTP 429.
// It embeds APIError and adds usage details.
type RateLimitError struct {
	APIError

	// Reason is a human-readable explanation of which limit was exceeded.
	Reason string `json:"message,omitempty"`

	// Usage contains the current usage counters and limits.
	Usage *EmailUsage `json:"usage,omitempty"`
}

// Error implements the error interface.
func (e *RateLimitError) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("envloped: rate limit exceeded (status %d): %s", e.StatusCode, e.Reason)
	}
	return fmt.Sprintf("envloped: rate limit exceeded (status %d)", e.StatusCode)
}

// Is enables sentinel error matching via errors.Is().
func (e *RateLimitError) Is(target error) bool {
	if target == ErrRateLimited {
		return true
	}
	return e.APIError.Is(target)
}

// Unwrap returns the underlying APIError for errors.Unwrap() support.
func (e *RateLimitError) Unwrap() error {
	return &e.APIError
}

// ValidationError is returned when the API responds with HTTP 400.
// It embeds APIError for consistency.
type ValidationError struct {
	APIError
}

// Is enables sentinel error matching via errors.Is().
func (e *ValidationError) Is(target error) bool {
	if target == ErrValidation {
		return true
	}
	return e.APIError.Is(target)
}

// Unwrap returns the underlying APIError for errors.Unwrap() support.
func (e *ValidationError) Unwrap() error {
	return &e.APIError
}

// handleErrorResponse parses an error response body and returns a typed error
// based on the HTTP status code.
func handleErrorResponse(resp *http.Response) error {
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusTooManyRequests:
		rateLimitErr := &RateLimitError{}
		rateLimitErr.StatusCode = resp.StatusCode
		if err := json.NewDecoder(resp.Body).Decode(rateLimitErr); err != nil {
			rateLimitErr.Message = http.StatusText(resp.StatusCode)
		}
		rateLimitErr.APIError.Message = rateLimitErr.Message
		if rateLimitErr.APIError.Message == "" {
			rateLimitErr.APIError.Message = "Rate limit exceeded"
		}
		return rateLimitErr

	case http.StatusBadRequest:
		apiErr := &APIError{}
		if err := json.NewDecoder(resp.Body).Decode(apiErr); err != nil {
			apiErr.Message = http.StatusText(resp.StatusCode)
		}
		apiErr.StatusCode = resp.StatusCode
		return &ValidationError{APIError: *apiErr}

	default:
		apiErr := &APIError{}
		if err := json.NewDecoder(resp.Body).Decode(apiErr); err != nil {
			apiErr.Message = http.StatusText(resp.StatusCode)
		}
		apiErr.StatusCode = resp.StatusCode
		if apiErr.Message == "" {
			apiErr.Message = http.StatusText(resp.StatusCode)
		}
		return apiErr
	}
}
