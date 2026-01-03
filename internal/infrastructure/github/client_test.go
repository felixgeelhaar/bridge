package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/felixgeelhaar/bolt"
)

func testLogger(t *testing.T) *bolt.Logger {
	t.Helper()
	handler := bolt.NewConsoleHandler(os.Stderr)
	return bolt.New(handler).SetLevel(bolt.ERROR)
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.BaseURL != defaultBaseURL {
		t.Errorf("BaseURL = %v, want %v", cfg.BaseURL, defaultBaseURL)
	}
	if cfg.Timeout != defaultTimeout {
		t.Errorf("Timeout = %v, want %v", cfg.Timeout, defaultTimeout)
	}
}

func TestNewClient(t *testing.T) {
	logger := testLogger(t)

	tests := []struct {
		name    string
		cfg     Config
		wantURL string
	}{
		{
			name:    "default config",
			cfg:     DefaultConfig(),
			wantURL: defaultBaseURL,
		},
		{
			name: "custom base URL",
			cfg: Config{
				BaseURL: "https://github.example.com/api/v3",
				Timeout: 60 * time.Second,
			},
			wantURL: "https://github.example.com/api/v3",
		},
		{
			name: "with token",
			cfg: Config{
				Token: "ghp_test_token",
			},
			wantURL: defaultBaseURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(logger, tt.cfg)
			if client == nil {
				t.Fatal("NewClient returned nil")
			}
			if client.baseURL != tt.wantURL {
				t.Errorf("baseURL = %v, want %v", client.baseURL, tt.wantURL)
			}
		})
	}
}

func TestClient_GetUser(t *testing.T) {
	logger := testLogger(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user" {
			t.Errorf("Path = %v, want /user", r.URL.Path)
		}
		if r.Method != "GET" {
			t.Errorf("Method = %v, want GET", r.Method)
		}

		// Check headers
		if got := r.Header.Get("Accept"); got != "application/vnd.github+json" {
			t.Errorf("Accept header = %v, want application/vnd.github+json", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization header = %v, want Bearer test-token", got)
		}

		user := User{
			Login:     "testuser",
			ID:        123,
			AvatarURL: "https://avatars.githubusercontent.com/u/123",
			Name:      "Test User",
			Email:     "test@example.com",
		}
		json.NewEncoder(w).Encode(user)
	}))
	defer server.Close()

	client := NewClient(logger, Config{
		BaseURL: server.URL,
		Token:   "test-token",
	})

	user, err := client.GetUser(context.Background())
	if err != nil {
		t.Fatalf("GetUser() error = %v", err)
	}

	if user.Login != "testuser" {
		t.Errorf("Login = %v, want testuser", user.Login)
	}
	if user.ID != 123 {
		t.Errorf("ID = %v, want 123", user.ID)
	}
	if user.Email != "test@example.com" {
		t.Errorf("Email = %v, want test@example.com", user.Email)
	}
}

func TestClient_GetRepository(t *testing.T) {
	logger := testLogger(t)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo" {
			t.Errorf("Path = %v, want /repos/owner/repo", r.URL.Path)
		}

		repo := Repository{
			ID:            456,
			Name:          "repo",
			FullName:      "owner/repo",
			Description:   "Test repository",
			Private:       false,
			HTMLURL:       "https://github.com/owner/repo",
			CloneURL:      "https://github.com/owner/repo.git",
			DefaultBranch: "main",
			Owner: User{
				Login: "owner",
			},
		}
		json.NewEncoder(w).Encode(repo)
	}))
	defer server.Close()

	client := NewClient(logger, Config{
		BaseURL: server.URL,
		Token:   "test-token",
	})

	repo, err := client.GetRepository(context.Background(), "owner", "repo")
	if err != nil {
		t.Fatalf("GetRepository() error = %v", err)
	}

	if repo.FullName != "owner/repo" {
		t.Errorf("FullName = %v, want owner/repo", repo.FullName)
	}
	if repo.DefaultBranch != "main" {
		t.Errorf("DefaultBranch = %v, want main", repo.DefaultBranch)
	}
}

func TestClient_APIError(t *testing.T) {
	logger := testLogger(t)

	tests := []struct {
		name       string
		statusCode int
		message    string
		checkFn    func(error) bool
	}{
		{
			name:       "not found",
			statusCode: 404,
			message:    "Not Found",
			checkFn:    IsNotFound,
		},
		{
			name:       "unauthorized",
			statusCode: 401,
			message:    "Bad credentials",
			checkFn:    IsUnauthorized,
		},
		{
			name:       "forbidden",
			statusCode: 403,
			message:    "Resource not accessible",
			checkFn:    IsForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				json.NewEncoder(w).Encode(APIError{
					Message: tt.message,
				})
			}))
			defer server.Close()

			client := NewClient(logger, Config{
				BaseURL: server.URL,
			})

			_, err := client.GetUser(context.Background())
			if err == nil {
				t.Fatal("Expected error, got nil")
			}

			if !tt.checkFn(err) {
				t.Errorf("Error check function returned false for status %d", tt.statusCode)
			}

			// Verify error message
			if apiErr, ok := err.(*APIError); ok {
				if apiErr.Message != tt.message {
					t.Errorf("Message = %v, want %v", apiErr.Message, tt.message)
				}
			}
		})
	}
}

func TestAPIError_Error(t *testing.T) {
	err := &APIError{
		StatusCode: 404,
		Message:    "Not Found",
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() should return non-empty string")
	}

	// Should contain status code
	if !containsString(errStr, "404") {
		t.Error("Error() should contain status code")
	}
}

func TestUserType(t *testing.T) {
	user := User{
		Login:     "testuser",
		ID:        123,
		AvatarURL: "https://example.com/avatar.png",
		Name:      "Test User",
		Email:     "test@example.com",
	}

	if user.Login != "testuser" {
		t.Error("Login not set correctly")
	}
	if user.ID != 123 {
		t.Error("ID not set correctly")
	}
}

func TestRepositoryType(t *testing.T) {
	repo := Repository{
		ID:            456,
		Name:          "repo",
		FullName:      "owner/repo",
		Description:   "Test repo",
		Private:       true,
		HTMLURL:       "https://github.com/owner/repo",
		CloneURL:      "https://github.com/owner/repo.git",
		DefaultBranch: "develop",
		Owner: User{
			Login: "owner",
		},
	}

	if repo.FullName != "owner/repo" {
		t.Error("FullName not set correctly")
	}
	if !repo.Private {
		t.Error("Private should be true")
	}
	if repo.Owner.Login != "owner" {
		t.Error("Owner.Login not set correctly")
	}
}

func TestLabelType(t *testing.T) {
	label := Label{
		ID:          789,
		Name:        "bug",
		Description: "Something isn't working",
		Color:       "d73a4a",
	}

	if label.Name != "bug" {
		t.Error("Name not set correctly")
	}
	if label.Color != "d73a4a" {
		t.Error("Color not set correctly")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStringHelper(s, substr))
}

func containsStringHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
