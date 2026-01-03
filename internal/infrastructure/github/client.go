package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/felixgeelhaar/bolt"
)

const (
	defaultBaseURL = "https://api.github.com"
	defaultTimeout = 30 * time.Second
)

// Client provides access to the GitHub API.
type Client struct {
	httpClient *http.Client
	baseURL    string
	token      string
	logger     *bolt.Logger
}

// Config contains GitHub client configuration.
type Config struct {
	Token   string
	BaseURL string
	Timeout time.Duration
}

// DefaultConfig returns the default GitHub client configuration.
func DefaultConfig() Config {
	return Config{
		BaseURL: defaultBaseURL,
		Timeout: defaultTimeout,
	}
}

// NewClient creates a new GitHub client.
func NewClient(logger *bolt.Logger, cfg Config) *Client {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		baseURL: baseURL,
		token:   cfg.Token,
		logger:  logger,
	}
}

// User represents a GitHub user.
type User struct {
	Login     string `json:"login"`
	ID        int64  `json:"id"`
	AvatarURL string `json:"avatar_url"`
	Name      string `json:"name"`
	Email     string `json:"email"`
}

// Repository represents a GitHub repository.
type Repository struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	Private     bool   `json:"private"`
	HTMLURL     string `json:"html_url"`
	CloneURL    string `json:"clone_url"`
	DefaultBranch string `json:"default_branch"`
	Owner       User   `json:"owner"`
}

// Label represents a GitHub label.
type Label struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

// GetUser returns the authenticated user.
func (c *Client) GetUser(ctx context.Context) (*User, error) {
	var user User
	if err := c.get(ctx, "/user", &user); err != nil {
		return nil, err
	}
	return &user, nil
}

// GetRepository returns a repository by owner and name.
func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*Repository, error) {
	var repository Repository
	path := fmt.Sprintf("/repos/%s/%s", owner, repo)
	if err := c.get(ctx, path, &repository); err != nil {
		return nil, err
	}
	return &repository, nil
}

// get performs a GET request to the GitHub API.
func (c *Client) get(ctx context.Context, path string, result any) error {
	return c.request(ctx, "GET", path, nil, result)
}

// post performs a POST request to the GitHub API.
func (c *Client) post(ctx context.Context, path string, body, result any) error {
	return c.request(ctx, "POST", path, body, result)
}

// patch performs a PATCH request to the GitHub API.
func (c *Client) patch(ctx context.Context, path string, body, result any) error {
	return c.request(ctx, "PATCH", path, body, result)
}

// delete performs a DELETE request to the GitHub API.
func (c *Client) delete(ctx context.Context, path string) error {
	return c.request(ctx, "DELETE", path, nil, nil)
}

// request performs an HTTP request to the GitHub API.
func (c *Client) request(ctx context.Context, method, path string, body, result any) error {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	c.logger.Debug().
		Str("method", method).
		Str("path", path).
		Msg("GitHub API request")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	c.logger.Debug().
		Int("status", resp.StatusCode).
		Int("body_len", len(respBody)).
		Msg("GitHub API response")

	if resp.StatusCode >= 400 {
		var apiError APIError
		if err := json.Unmarshal(respBody, &apiError); err == nil {
			apiError.StatusCode = resp.StatusCode
			return &apiError
		}
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to parse response: %w", err)
		}
	}

	return nil
}

// APIError represents a GitHub API error.
type APIError struct {
	StatusCode int    `json:"-"`
	Message    string `json:"message"`
	DocURL     string `json:"documentation_url,omitempty"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("github api error (status %d): %s", e.StatusCode, e.Message)
}

// IsNotFound returns true if the error is a 404 Not Found error.
func IsNotFound(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == 404
	}
	return false
}

// IsUnauthorized returns true if the error is a 401 Unauthorized error.
func IsUnauthorized(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == 401
	}
	return false
}

// IsForbidden returns true if the error is a 403 Forbidden error.
func IsForbidden(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == 403
	}
	return false
}
