package github

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/felixgeelhaar/bolt"
)

func webhookTestLogger(t *testing.T) *bolt.Logger {
	t.Helper()
	handler := bolt.NewConsoleHandler(os.Stderr)
	return bolt.New(handler).SetLevel(bolt.ERROR)
}

func generateSignature(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestNewWebhookHandler(t *testing.T) {
	logger := webhookTestLogger(t)
	handler := NewWebhookHandler(logger, "secret")

	if handler == nil {
		t.Fatal("NewWebhookHandler returned nil")
	}
}

func TestWebhookHandler_On(t *testing.T) {
	logger := webhookTestLogger(t)
	handler := NewWebhookHandler(logger, "secret")

	handler.On(EventPullRequest, func(ctx context.Context, event WebhookEvent, payload any) error {
		return nil
	})

	// Verify handler is registered
	if len(handler.handlers[EventPullRequest]) != 1 {
		t.Error("Handler should be registered")
	}
}

func TestWebhookHandler_VerifySignature(t *testing.T) {
	logger := webhookTestLogger(t)
	handler := NewWebhookHandler(logger, "test-secret")

	payload := []byte(`{"action":"opened"}`)

	tests := []struct {
		name      string
		signature string
		want      bool
	}{
		{
			name:      "valid signature",
			signature: generateSignature("test-secret", payload),
			want:      true,
		},
		{
			name:      "invalid signature",
			signature: "sha256=invalid",
			want:      false,
		},
		{
			name:      "missing prefix",
			signature: "abc123",
			want:      false,
		},
		{
			name:      "empty signature",
			signature: "",
			want:      false,
		},
		{
			name:      "wrong secret",
			signature: generateSignature("wrong-secret", payload),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := handler.verifySignature(payload, tt.signature)
			if got != tt.want {
				t.Errorf("verifySignature() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWebhookHandler_ServeHTTP_PullRequest(t *testing.T) {
	logger := webhookTestLogger(t)
	secret := "test-secret"
	handler := NewWebhookHandler(logger, secret)

	var receivedEvent WebhookEvent
	var receivedPayload any
	handler.On(EventPullRequest, func(ctx context.Context, event WebhookEvent, payload any) error {
		receivedEvent = event
		receivedPayload = payload
		return nil
	})

	payload := PullRequestPayload{
		WebhookPayload: WebhookPayload{
			Action: ActionOpened,
			Repository: Repository{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner:    User{Login: "owner"},
			},
			Sender: User{Login: "testuser"},
		},
		Number: 123,
		PullRequest: PullRequest{
			Number: 123,
			Title:  "Test PR",
			State:  "open",
			Head:   PRBranch{Ref: "feature", SHA: "abc123"},
			Base:   PRBranch{Ref: "main", SHA: "def456"},
		},
	}

	body, _ := json.Marshal(payload)
	signature := generateSignature(secret, body)

	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-GitHub-Delivery", "delivery-123")
	req.Header.Set("X-Hub-Signature-256", signature)

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Status code = %v, want %v", rr.Code, http.StatusOK)
	}

	if receivedEvent != EventPullRequest {
		t.Errorf("Received event = %v, want %v", receivedEvent, EventPullRequest)
	}

	if receivedPayload == nil {
		t.Error("Payload should not be nil")
	}

	prPayload, ok := receivedPayload.(*PullRequestPayload)
	if !ok {
		t.Fatal("Payload should be *PullRequestPayload")
	}

	if prPayload.Number != 123 {
		t.Errorf("PR number = %v, want 123", prPayload.Number)
	}
}

func TestWebhookHandler_ServeHTTP_InvalidSignature(t *testing.T) {
	logger := webhookTestLogger(t)
	handler := NewWebhookHandler(logger, "correct-secret")

	body := []byte(`{"action":"opened"}`)

	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "pull_request")
	req.Header.Set("X-Hub-Signature-256", generateSignature("wrong-secret", body))

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Status code = %v, want %v", rr.Code, http.StatusUnauthorized)
	}
}

func TestWebhookHandler_ServeHTTP_NoSecret(t *testing.T) {
	logger := webhookTestLogger(t)
	handler := NewWebhookHandler(logger, "") // No secret configured

	payload := PingPayload{
		Zen:    "Test zen",
		HookID: 456,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))
	req.Header.Set("X-GitHub-Event", "ping")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Should succeed without signature verification
	if rr.Code != http.StatusOK {
		t.Errorf("Status code = %v, want %v", rr.Code, http.StatusOK)
	}
}

func TestWebhookEvent_Constants(t *testing.T) {
	tests := []struct {
		event WebhookEvent
		want  string
	}{
		{EventPullRequest, "pull_request"},
		{EventPullRequestReview, "pull_request_review"},
		{EventIssueComment, "issue_comment"},
		{EventPush, "push"},
		{EventCheckRun, "check_run"},
		{EventCheckSuite, "check_suite"},
		{EventPing, "ping"},
	}

	for _, tt := range tests {
		if string(tt.event) != tt.want {
			t.Errorf("Event %v = %v, want %v", tt.event, string(tt.event), tt.want)
		}
	}
}

func TestWebhookAction_Constants(t *testing.T) {
	tests := []struct {
		action WebhookAction
		want   string
	}{
		{ActionOpened, "opened"},
		{ActionClosed, "closed"},
		{ActionReopened, "reopened"},
		{ActionSynchronize, "synchronize"},
		{ActionEdited, "edited"},
		{ActionSubmitted, "submitted"},
		{ActionCreated, "created"},
		{ActionDeleted, "deleted"},
		{ActionCompleted, "completed"},
		{ActionRequested, "requested"},
	}

	for _, tt := range tests {
		if string(tt.action) != tt.want {
			t.Errorf("Action %v = %v, want %v", tt.action, string(tt.action), tt.want)
		}
	}
}

func TestExtractTriggerData_PullRequest(t *testing.T) {
	payload := &PullRequestPayload{
		WebhookPayload: WebhookPayload{
			Action: ActionOpened,
			Repository: Repository{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner:    User{Login: "owner"},
			},
			Sender: User{Login: "author", Email: "author@example.com"},
		},
		Number: 42,
		PullRequest: PullRequest{
			Number:       42,
			Title:        "Add feature",
			Body:         "Description of changes",
			HTMLURL:      "https://github.com/owner/test-repo/pull/42",
			Draft:        false,
			Additions:    10,
			Deletions:    5,
			ChangedFiles: 3,
			Head:         PRBranch{Ref: "feature-branch", SHA: "abc123"},
			Base:         PRBranch{Ref: "main", SHA: "def456"},
		},
	}

	data, err := ExtractTriggerData(EventPullRequest, payload)
	if err != nil {
		t.Fatalf("ExtractTriggerData() error = %v", err)
	}

	if data.Event != "pull_request" {
		t.Errorf("Event = %v, want pull_request", data.Event)
	}

	if data.Action != "opened" {
		t.Errorf("Action = %v, want opened", data.Action)
	}

	if data.Repository.FullName != "owner/test-repo" {
		t.Errorf("Repository.FullName = %v, want owner/test-repo", data.Repository.FullName)
	}

	if data.Sender.Login != "author" {
		t.Errorf("Sender.Login = %v, want author", data.Sender.Login)
	}

	if data.PR == nil {
		t.Fatal("PR should not be nil")
	}

	if data.PR.Number != 42 {
		t.Errorf("PR.Number = %v, want 42", data.PR.Number)
	}

	if data.PR.HeadRef != "feature-branch" {
		t.Errorf("PR.HeadRef = %v, want feature-branch", data.PR.HeadRef)
	}

	if data.PR.Additions != 10 {
		t.Errorf("PR.Additions = %v, want 10", data.PR.Additions)
	}
}

func TestExtractTriggerData_Push(t *testing.T) {
	payload := &PushPayload{
		WebhookPayload: WebhookPayload{
			Action: "",
			Repository: Repository{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner:    User{Login: "owner"},
			},
			Sender: User{Login: "pusher"},
		},
		Ref:    "refs/heads/main",
		Before: "abc123",
		After:  "def456",
		Forced: false,
		Pusher: User{Login: "pusher", Email: "pusher@example.com"},
		Commits: []PushCommit{
			{
				ID:       "commit1",
				Message:  "First commit",
				Added:    []string{"file1.go"},
				Modified: []string{"file2.go"},
				Removed:  []string{},
			},
		},
	}

	data, err := ExtractTriggerData(EventPush, payload)
	if err != nil {
		t.Fatalf("ExtractTriggerData() error = %v", err)
	}

	if data.Event != "push" {
		t.Errorf("Event = %v, want push", data.Event)
	}

	if data.Push == nil {
		t.Fatal("Push should not be nil")
	}

	if data.Push.Ref != "refs/heads/main" {
		t.Errorf("Push.Ref = %v, want refs/heads/main", data.Push.Ref)
	}

	if data.Push.After != "def456" {
		t.Errorf("Push.After = %v, want def456", data.Push.After)
	}

	// Should contain files from commits
	if len(data.Push.Files) != 2 {
		t.Errorf("Push.Files length = %v, want 2", len(data.Push.Files))
	}
}

func TestExtractTriggerData_IssueComment(t *testing.T) {
	payload := &IssueCommentPayload{
		WebhookPayload: WebhookPayload{
			Action: ActionCreated,
			Repository: Repository{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner:    User{Login: "owner"},
			},
			Sender: User{Login: "commenter"},
		},
		Issue: Issue{
			Number:  42,
			Title:   "Issue title",
			Body:    "Issue body",
			HTMLURL: "https://github.com/owner/test-repo/issues/42",
			PullRequest: &struct {
				URL string `json:"url"`
			}{URL: "https://api.github.com/repos/owner/test-repo/pulls/42"},
		},
		Comment: PRComment{
			ID:      789,
			Body:    "Great work!",
			HTMLURL: "https://github.com/owner/test-repo/issues/42#issuecomment-789",
		},
	}

	data, err := ExtractTriggerData(EventIssueComment, payload)
	if err != nil {
		t.Fatalf("ExtractTriggerData() error = %v", err)
	}

	if data.Event != "issue_comment" {
		t.Errorf("Event = %v, want issue_comment", data.Event)
	}

	if data.Comment == nil {
		t.Fatal("Comment should not be nil")
	}

	if data.Comment.Body != "Great work!" {
		t.Errorf("Comment.Body = %v, want 'Great work!'", data.Comment.Body)
	}

	// Since issue has PullRequest field, PR should be populated
	if data.PR == nil {
		t.Fatal("PR should not be nil for PR comment")
	}

	if data.PR.Number != 42 {
		t.Errorf("PR.Number = %v, want 42", data.PR.Number)
	}
}

func TestExtractTriggerData_PullRequestReview(t *testing.T) {
	payload := &PullRequestReviewPayload{
		WebhookPayload: WebhookPayload{
			Action: ActionSubmitted,
			Repository: Repository{
				Name:     "test-repo",
				FullName: "owner/test-repo",
				Owner:    User{Login: "owner"},
			},
			Sender: User{Login: "reviewer"},
		},
		PullRequest: PullRequest{
			Number:  42,
			Title:   "Add feature",
			HTMLURL: "https://github.com/owner/test-repo/pull/42",
			Head:    PRBranch{Ref: "feature", SHA: "abc123"},
			Base:    PRBranch{Ref: "main", SHA: "def456"},
		},
		Review: PRReview{
			ID:    999,
			State: "approved",
			Body:  "Looks good!",
		},
	}

	data, err := ExtractTriggerData(EventPullRequestReview, payload)
	if err != nil {
		t.Fatalf("ExtractTriggerData() error = %v", err)
	}

	if data.Event != "pull_request_review" {
		t.Errorf("Event = %v, want pull_request_review", data.Event)
	}

	if data.Review == nil {
		t.Fatal("Review should not be nil")
	}

	if data.Review.State != "approved" {
		t.Errorf("Review.State = %v, want approved", data.Review.State)
	}

	if data.Review.Body != "Looks good!" {
		t.Errorf("Review.Body = %v, want 'Looks good!'", data.Review.Body)
	}
}

func TestExtractTriggerData_UnsupportedPayload(t *testing.T) {
	_, err := ExtractTriggerData(EventPing, "unsupported type")
	if err == nil {
		t.Error("Expected error for unsupported payload type")
	}
}

func TestTriggerDataTypes(t *testing.T) {
	// Test RepoTrigger
	repo := RepoTrigger{
		Owner:    "owner",
		Name:     "repo",
		FullName: "owner/repo",
	}
	if repo.FullName != "owner/repo" {
		t.Error("RepoTrigger.FullName not set correctly")
	}

	// Test UserTrigger
	user := UserTrigger{
		Login: "user",
		Email: "user@example.com",
	}
	if user.Login != "user" {
		t.Error("UserTrigger.Login not set correctly")
	}

	// Test PRTrigger
	pr := PRTrigger{
		Number:       1,
		Title:        "Test PR",
		HeadRef:      "feature",
		HeadSHA:      "abc123",
		BaseRef:      "main",
		BaseSHA:      "def456",
		ChangedFiles: 5,
	}
	if pr.Number != 1 {
		t.Error("PRTrigger.Number not set correctly")
	}

	// Test ReviewTrigger
	review := ReviewTrigger{
		ID:    123,
		State: "approved",
		Body:  "LGTM",
	}
	if review.State != "approved" {
		t.Error("ReviewTrigger.State not set correctly")
	}

	// Test CommentTrigger
	comment := CommentTrigger{
		ID:      456,
		Body:    "Nice work",
		HTMLURL: "https://github.com/...",
	}
	if comment.Body != "Nice work" {
		t.Error("CommentTrigger.Body not set correctly")
	}

	// Test PushTrigger
	push := PushTrigger{
		Ref:    "refs/heads/main",
		Before: "abc",
		After:  "def",
		Forced: true,
		Files:  []string{"a.go", "b.go"},
	}
	if !push.Forced {
		t.Error("PushTrigger.Forced should be true")
	}
}
