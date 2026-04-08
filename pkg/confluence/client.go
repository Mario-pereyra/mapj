package confluence

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type Client struct {
	BaseURL  string
	Token    string // For Bearer token (PAT)
	Username string // For Basic auth (email)
	Password string // For Basic auth (API token)
	UseBasic bool   // Use Basic auth instead of Bearer

	httpClient *http.Client
	basicUser  string
	basicPass  string

	// manifestMu protects concurrent writes to the manifest file during parallel exports
	manifestMu sync.Mutex
}

type APIResponse struct {
	Results interface{} `json:"results,omitempty"`
	Size    int         `json:"size,omitempty"`
	Total   int         `json:"total,omitempty"`
	Start   int         `json:"start,omitempty"`
	Links   *Links      `json:"_links,omitempty"`
}

type Links struct {
	Next    string `json:"next,omitempty"`
	Base    string `json:"base,omitempty"`
	Context string `json:"context,omitempty"`
}

func NewClient(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) SetBasicAuth(username, apiToken string) {
	c.Username = username
	c.Password = apiToken
	c.UseBasic = true
	c.basicUser = username
	c.basicPass = apiToken
}

func (c *Client) _getHeaders() map[string]string {
	return map[string]string{
		"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Accept":     "application/json",
	}
}

func (c *Client) do(ctx context.Context, method, path string, params map[string]string) ([]byte, error) {
	u, err := url.Parse(c.BaseURL + path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	if params != nil {
		q := u.Query()
		for k, v := range params {
			q.Set(k, v)
		}
		u.RawQuery = q.Encode()
	}

	maxRetries := 3
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, u.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		for k, v := range c._getHeaders() {
			req.Header.Set(k, v)
		}

		if c.UseBasic && c.Username != "" && c.Password != "" {
			auth := base64.StdEncoding.EncodeToString([]byte(c.Username + ":" + c.Password))
			req.Header.Set("Authorization", "Basic "+auth)
		} else if c.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			backoffWithJitter(attempt)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			backoffWithJitter(attempt)
			continue
		}

		if resp.StatusCode >= 400 {
			apiErr := NewAPIError(resp.StatusCode, string(body))
			lastErr = apiErr

			if apiErr.Retryable {
				if attempt < maxRetries {
					backoffWithJitter(attempt)
					continue
				}
			}
			return nil, lastErr
		}

		return body, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func backoffWithJitter(attempt int) {
	base := time.Duration(1<<attempt) * time.Second
	jitter := time.Duration(rand.Intn(500)) * time.Millisecond
	time.Sleep(base + jitter)
}

func (c *Client) Ping(ctx context.Context) error {
	_, err := c.do(ctx, "GET", "/rest/api/space", map[string]string{"limit": "1"})
	return err
}
