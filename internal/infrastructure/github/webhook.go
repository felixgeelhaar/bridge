package github

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/felixgeelhaar/bolt"
)

// WebhookEvent represents the type of GitHub webhook event.
type WebhookEvent string

const (
	EventPullRequest       WebhookEvent = "pull_request"
	EventPullRequestReview WebhookEvent = "pull_request_review"
	EventIssueComment      WebhookEvent = "issue_comment"
	EventPush              WebhookEvent = "push"
	EventCheckRun          WebhookEvent = "check_run"
	EventCheckSuite        WebhookEvent = "check_suite"
	EventPing              WebhookEvent = "ping"
)

// WebhookAction represents the action that triggered the webhook.
type WebhookAction string

const (
	ActionOpened      WebhookAction = "opened"
	ActionClosed      WebhookAction = "closed"
	ActionReopened    WebhookAction = "reopened"
	ActionSynchronize WebhookAction = "synchronize"
	ActionEdited      WebhookAction = "edited"
	ActionSubmitted   WebhookAction = "submitted"
	ActionCreated     WebhookAction = "created"
	ActionDeleted     WebhookAction = "deleted"
	ActionCompleted   WebhookAction = "completed"
	ActionRequested   WebhookAction = "requested"
)

// WebhookPayload represents the common fields in a webhook payload.
type WebhookPayload struct {
	Action       WebhookAction `json:"action"`
	Repository   Repository    `json:"repository"`
	Sender       User          `json:"sender"`
	Installation *Installation `json:"installation,omitempty"`
}

// Installation represents a GitHub App installation.
type Installation struct {
	ID int64 `json:"id"`
}

// PullRequestPayload represents a pull_request webhook payload.
type PullRequestPayload struct {
	WebhookPayload
	Number      int         `json:"number"`
	PullRequest PullRequest `json:"pull_request"`
	Changes     *PRChanges  `json:"changes,omitempty"`
}

// PRChanges represents changes made to a pull request (for edited action).
type PRChanges struct {
	Title *struct {
		From string `json:"from"`
	} `json:"title,omitempty"`
	Body *struct {
		From string `json:"from"`
	} `json:"body,omitempty"`
}

// PullRequestReviewPayload represents a pull_request_review webhook payload.
type PullRequestReviewPayload struct {
	WebhookPayload
	PullRequest PullRequest `json:"pull_request"`
	Review      PRReview    `json:"review"`
}

// IssueCommentPayload represents an issue_comment webhook payload.
type IssueCommentPayload struct {
	WebhookPayload
	Issue   Issue     `json:"issue"`
	Comment PRComment `json:"comment"`
}

// Issue represents a GitHub issue (PR comments come as issue_comment events).
type Issue struct {
	ID          int64  `json:"id"`
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	State       string `json:"state"`
	HTMLURL     string `json:"html_url"`
	User        User   `json:"user"`
	PullRequest *struct {
		URL string `json:"url"`
	} `json:"pull_request,omitempty"`
}

// PushPayload represents a push webhook payload.
type PushPayload struct {
	WebhookPayload
	Ref        string       `json:"ref"`
	Before     string       `json:"before"`
	After      string       `json:"after"`
	Commits    []PushCommit `json:"commits"`
	HeadCommit *PushCommit  `json:"head_commit,omitempty"`
	Pusher     User         `json:"pusher"`
	Forced     bool         `json:"forced"`
}

// PushCommit represents a commit in a push event.
type PushCommit struct {
	ID        string   `json:"id"`
	Message   string   `json:"message"`
	Timestamp string   `json:"timestamp"`
	URL       string   `json:"url"`
	Author    User     `json:"author"`
	Committer User     `json:"committer"`
	Added     []string `json:"added"`
	Removed   []string `json:"removed"`
	Modified  []string `json:"modified"`
}

// PingPayload represents a ping webhook payload (sent when webhook is created).
type PingPayload struct {
	Zen    string `json:"zen"`
	HookID int64  `json:"hook_id"`
	Hook   struct {
		Type   string   `json:"type"`
		Events []string `json:"events"`
		Active bool     `json:"active"`
	} `json:"hook"`
	WebhookPayload
}

// WebhookHandler handles incoming GitHub webhooks.
type WebhookHandler struct {
	secret   string
	logger   *bolt.Logger
	handlers map[WebhookEvent][]EventHandler
}

// EventHandler is a function that handles a specific webhook event.
type EventHandler func(ctx context.Context, event WebhookEvent, payload any) error

// NewWebhookHandler creates a new webhook handler.
func NewWebhookHandler(logger *bolt.Logger, secret string) *WebhookHandler {
	return &WebhookHandler{
		secret:   secret,
		logger:   logger,
		handlers: make(map[WebhookEvent][]EventHandler),
	}
}

// On registers a handler for a specific event type.
func (h *WebhookHandler) On(event WebhookEvent, handler EventHandler) {
	h.handlers[event] = append(h.handlers[event], handler)
}

// ServeHTTP handles incoming webhook requests.
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error().Err(err).Msg("Failed to read webhook body")
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer func() { _ = r.Body.Close() }()

	// Verify signature if secret is set
	if h.secret != "" {
		signature := r.Header.Get("X-Hub-Signature-256")
		if !h.verifySignature(body, signature) {
			h.logger.Warn().Msg("Invalid webhook signature")
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// Get event type
	eventType := WebhookEvent(r.Header.Get("X-GitHub-Event"))
	deliveryID := r.Header.Get("X-GitHub-Delivery")

	h.logger.Info().
		Str("event", string(eventType)).
		Str("delivery_id", deliveryID).
		Msg("Received webhook")

	// Parse and handle the event
	payload, err := h.parsePayload(eventType, body)
	if err != nil {
		h.logger.Error().Err(err).Str("event", string(eventType)).Msg("Failed to parse payload")
		http.Error(w, "Failed to parse payload", http.StatusBadRequest)
		return
	}

	// Call registered handlers
	handlers := h.handlers[eventType]
	for _, handler := range handlers {
		if err := handler(ctx, eventType, payload); err != nil {
			h.logger.Error().Err(err).Str("event", string(eventType)).Msg("Handler error")
			// Continue processing other handlers
		}
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// verifySignature verifies the HMAC-SHA256 signature of the payload.
func (h *WebhookHandler) verifySignature(payload []byte, signature string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}

	sig, err := hex.DecodeString(signature[7:])
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(payload)
	expectedMAC := mac.Sum(nil)

	return hmac.Equal(sig, expectedMAC)
}

// parsePayload parses the webhook payload based on event type.
func (h *WebhookHandler) parsePayload(event WebhookEvent, body []byte) (any, error) {
	switch event {
	case EventPullRequest:
		var payload PullRequestPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("failed to parse pull_request payload: %w", err)
		}
		return &payload, nil

	case EventPullRequestReview:
		var payload PullRequestReviewPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("failed to parse pull_request_review payload: %w", err)
		}
		return &payload, nil

	case EventIssueComment:
		var payload IssueCommentPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("failed to parse issue_comment payload: %w", err)
		}
		return &payload, nil

	case EventPush:
		var payload PushPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("failed to parse push payload: %w", err)
		}
		return &payload, nil

	case EventPing:
		var payload PingPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("failed to parse ping payload: %w", err)
		}
		return &payload, nil

	default:
		// Return raw payload for unknown events
		var payload WebhookPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			return nil, fmt.Errorf("failed to parse payload: %w", err)
		}
		return &payload, nil
	}
}

// TriggerData extracts workflow trigger data from a webhook payload.
type TriggerData struct {
	Event      string          `json:"event"`
	Action     string          `json:"action"`
	Repository RepoTrigger     `json:"repo"`
	Sender     UserTrigger     `json:"sender"`
	PR         *PRTrigger      `json:"pr,omitempty"`
	Review     *ReviewTrigger  `json:"review,omitempty"`
	Comment    *CommentTrigger `json:"comment,omitempty"`
	Push       *PushTrigger    `json:"push,omitempty"`
}

// RepoTrigger contains repository trigger data.
type RepoTrigger struct {
	Owner    string `json:"owner"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
}

// UserTrigger contains user trigger data.
type UserTrigger struct {
	Login string `json:"login"`
	Email string `json:"email"`
}

// PRTrigger contains pull request trigger data.
type PRTrigger struct {
	Number       int    `json:"number"`
	Title        string `json:"title"`
	Body         string `json:"body"`
	HeadRef      string `json:"head_ref"`
	HeadSHA      string `json:"head_sha"`
	BaseRef      string `json:"base_ref"`
	BaseSHA      string `json:"base_sha"`
	Draft        bool   `json:"draft"`
	HTMLURL      string `json:"html_url"`
	Additions    int    `json:"additions"`
	Deletions    int    `json:"deletions"`
	ChangedFiles int    `json:"changed_files"`
}

// ReviewTrigger contains review trigger data.
type ReviewTrigger struct {
	ID    int64  `json:"id"`
	State string `json:"state"`
	Body  string `json:"body"`
}

// CommentTrigger contains comment trigger data.
type CommentTrigger struct {
	ID      int64  `json:"id"`
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
}

// PushTrigger contains push trigger data.
type PushTrigger struct {
	Ref    string   `json:"ref"`
	Before string   `json:"before"`
	After  string   `json:"after"`
	Forced bool     `json:"forced"`
	Files  []string `json:"files"`
}

// ExtractTriggerData extracts workflow trigger data from a webhook payload.
func ExtractTriggerData(event WebhookEvent, payload any) (*TriggerData, error) {
	data := &TriggerData{
		Event: string(event),
	}

	switch p := payload.(type) {
	case *PullRequestPayload:
		data.Action = string(p.Action)
		data.Repository = RepoTrigger{
			Owner:    p.Repository.Owner.Login,
			Name:     p.Repository.Name,
			FullName: p.Repository.FullName,
		}
		data.Sender = UserTrigger{
			Login: p.Sender.Login,
			Email: p.Sender.Email,
		}
		data.PR = &PRTrigger{
			Number:       p.PullRequest.Number,
			Title:        p.PullRequest.Title,
			Body:         p.PullRequest.Body,
			HeadRef:      p.PullRequest.Head.Ref,
			HeadSHA:      p.PullRequest.Head.SHA,
			BaseRef:      p.PullRequest.Base.Ref,
			BaseSHA:      p.PullRequest.Base.SHA,
			Draft:        p.PullRequest.Draft,
			HTMLURL:      p.PullRequest.HTMLURL,
			Additions:    p.PullRequest.Additions,
			Deletions:    p.PullRequest.Deletions,
			ChangedFiles: p.PullRequest.ChangedFiles,
		}

	case *PullRequestReviewPayload:
		data.Action = string(p.Action)
		data.Repository = RepoTrigger{
			Owner:    p.Repository.Owner.Login,
			Name:     p.Repository.Name,
			FullName: p.Repository.FullName,
		}
		data.Sender = UserTrigger{
			Login: p.Sender.Login,
			Email: p.Sender.Email,
		}
		data.PR = &PRTrigger{
			Number:  p.PullRequest.Number,
			Title:   p.PullRequest.Title,
			Body:    p.PullRequest.Body,
			HeadRef: p.PullRequest.Head.Ref,
			HeadSHA: p.PullRequest.Head.SHA,
			BaseRef: p.PullRequest.Base.Ref,
			BaseSHA: p.PullRequest.Base.SHA,
			Draft:   p.PullRequest.Draft,
			HTMLURL: p.PullRequest.HTMLURL,
		}
		data.Review = &ReviewTrigger{
			ID:    p.Review.ID,
			State: p.Review.State,
			Body:  p.Review.Body,
		}

	case *IssueCommentPayload:
		data.Action = string(p.Action)
		data.Repository = RepoTrigger{
			Owner:    p.Repository.Owner.Login,
			Name:     p.Repository.Name,
			FullName: p.Repository.FullName,
		}
		data.Sender = UserTrigger{
			Login: p.Sender.Login,
			Email: p.Sender.Email,
		}
		if p.Issue.PullRequest != nil {
			data.PR = &PRTrigger{
				Number:  p.Issue.Number,
				Title:   p.Issue.Title,
				Body:    p.Issue.Body,
				HTMLURL: p.Issue.HTMLURL,
			}
		}
		data.Comment = &CommentTrigger{
			ID:      p.Comment.ID,
			Body:    p.Comment.Body,
			HTMLURL: p.Comment.HTMLURL,
		}

	case *PushPayload:
		data.Action = "push"
		data.Repository = RepoTrigger{
			Owner:    p.Repository.Owner.Login,
			Name:     p.Repository.Name,
			FullName: p.Repository.FullName,
		}
		data.Sender = UserTrigger{
			Login: p.Pusher.Login,
			Email: p.Pusher.Email,
		}
		files := make([]string, 0)
		for _, commit := range p.Commits {
			files = append(files, commit.Added...)
			files = append(files, commit.Modified...)
		}
		data.Push = &PushTrigger{
			Ref:    p.Ref,
			Before: p.Before,
			After:  p.After,
			Forced: p.Forced,
			Files:  files,
		}

	default:
		return nil, fmt.Errorf("unsupported payload type: %T", payload)
	}

	return data, nil
}
