# Envloped Go SDK

The official Go client library for the [Envloped](https://envloped.com) email API.

## Installation

```bash
go get github.com/envloped/envloped-go
```

Requires Go 1.21 or later. Zero third-party dependencies.

## Quick Start

```go
package main

import (
    "fmt"
    "log"

    envloped "github.com/envloped/envloped-go"
)

func main() {
    client := envloped.NewClient("ev_your_api_key")

    resp, err := client.Emails.Send(&envloped.SendEmailRequest{
        From:    "hello@yourdomain.com",
        To:      []string{"user@example.com"},
        Subject: "Hello from Envloped",
        Html:    "<p>Welcome!</p>",
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Sent! Message ID:", resp.MessageId)
}
```

## API Reference

### Creating a Client

```go
// Default client (production API)
client := envloped.NewClient("ev_your_api_key")

// Custom base URL (for testing or self-hosted)
client := envloped.NewClient("ev_your_api_key").
    WithBaseURL("https://your-api.example.com")

// Custom HTTP client (timeouts, proxies, etc.)
client := envloped.NewClient("ev_your_api_key").
    WithHTTPClient(&http.Client{Timeout: 10 * time.Second})
```

### Sending Emails

```go
resp, err := client.Emails.Send(&envloped.SendEmailRequest{
    From:    "My App <hello@yourdomain.com>",
    To:      []string{"user@example.com"},
    Subject: "Welcome!",
    Html:    "<h1>Hello</h1><p>Welcome to our app.</p>",
    Text:    "Hello\n\nWelcome to our app.", // optional fallback
})
```

**Fields:**

| Field     | Type       | Required | Description                                |
| --------- | ---------- | -------- | ------------------------------------------ |
| `From`    | `string`   | Yes      | Sender address. Domain must be verified.   |
| `To`      | `[]string` | Yes      | Recipient addresses.                       |
| `Subject` | `string`   | Yes      | Email subject line.                        |
| `Html`    | `string`   | *        | HTML body. At least one of Html/Text required. |
| `Text`    | `string`   | *        | Plain text body. At least one of Html/Text required. |

**Response:**

```go
type SendEmailResponse struct {
    Success   bool   `json:"success"`
    MessageId string `json:"messageId"`
}
```

### Ping

Check connectivity and API key validity:

```go
pong, err := client.Ping()
fmt.Println(pong.Message)   // "pong"
fmt.Println(pong.CompanyID) // your company ID
```

### Context Support

Every method has a `WithContext` variant for cancellation and deadlines:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

resp, err := client.Emails.SendWithContext(ctx, &envloped.SendEmailRequest{
    // ...
})
```

## Error Handling

All API errors are returned as typed errors that support `errors.Is()` and `errors.As()`:

```go
resp, err := client.Emails.Send(params)
if err != nil {
    // Sentinel error checks
    if errors.Is(err, envloped.ErrUnauthorized) {
        log.Fatal("Invalid API key")
    }
    if errors.Is(err, envloped.ErrForbidden) {
        log.Fatal("Domain not verified")
    }
    if errors.Is(err, envloped.ErrValidation) {
        log.Fatal("Invalid request:", err)
    }

    // Rate limit with usage details
    if errors.Is(err, envloped.ErrRateLimited) {
        var rle *envloped.RateLimitError
        if errors.As(err, &rle) {
            fmt.Printf("Rate limited: %s\n", rle.Reason)
            if rle.Usage != nil {
                fmt.Printf("Monthly: %d/%d\n", rle.Usage.MonthlyCount, rle.Usage.MonthlyLimit)
            }
        }
    }

    // Generic API error with status code
    var apiErr *envloped.APIError
    if errors.As(err, &apiErr) {
        fmt.Printf("API error %d: %s\n", apiErr.StatusCode, apiErr.Message)
    }
}
```

**Error types:**

| Type              | HTTP Status | Sentinel           | Description                    |
| ----------------- | ----------- | ------------------ | ------------------------------ |
| `*ValidationError`| 400         | `ErrValidation`    | Invalid request fields         |
| `*APIError`       | 401         | `ErrUnauthorized`  | Missing or invalid API key     |
| `*APIError`       | 403         | `ErrForbidden`     | Domain not registered/verified |
| `*RateLimitError` | 429         | `ErrRateLimited`   | Usage limits exceeded          |
| `*APIError`       | 500         | --                 | Server error                   |

## Mocking in Tests

The `EmailsSvc` interface makes it easy to mock the SDK in your tests:

```go
type mockEmailsSvc struct{}

func (m *mockEmailsSvc) Send(params *envloped.SendEmailRequest) (*envloped.SendEmailResponse, error) {
    return &envloped.SendEmailResponse{Success: true, MessageId: "mock_123"}, nil
}

func (m *mockEmailsSvc) SendWithContext(ctx context.Context, params *envloped.SendEmailRequest) (*envloped.SendEmailResponse, error) {
    return m.Send(params)
}
```

## Version

```go
fmt.Println(envloped.Version()) // "1.0.0"
```

## License

MIT -- see [LICENSE](LICENSE).
