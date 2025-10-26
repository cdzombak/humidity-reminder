package mailgun

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const defaultBaseURL = "https://api.mailgun.net/v3"

// Client sends messages through the Mailgun API.
type Client struct {
	baseURL    string
	domain     string
	apiKey     string
	httpClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the Mailgun API base URL. Intended for tests.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		c.baseURL = strings.TrimRight(baseURL, "/")
	}
}

// NewClient constructs a Mailgun client.
func NewClient(domain, apiKey string, httpClient *http.Client, opts ...Option) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	client := &Client{
		baseURL:    defaultBaseURL,
		domain:     domain,
		apiKey:     apiKey,
		httpClient: httpClient,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

// Send transmits a simple text email via Mailgun.
func (c *Client) Send(ctx context.Context, from, to, subject, text string) error {
	endpoint := fmt.Sprintf("%s/%s/messages", c.baseURL, c.domain)

	body := url.Values{}
	body.Set("from", from)
	body.Set("to", to)
	body.Set("subject", subject)
	body.Set("text", text)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(body.Encode()))
	if err != nil {
		return fmt.Errorf("create mailgun request: %w", err)
	}
	req.SetBasicAuth("api", c.apiKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute mailgun request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("mailgun request failed: %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}

	return nil
}
