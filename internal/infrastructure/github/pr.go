package github

import (
	"context"
	"fmt"
	"time"
)

// PullRequest represents a GitHub pull request.
type PullRequest struct {
	ID          int64      `json:"id"`
	Number      int        `json:"number"`
	State       string     `json:"state"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	HTMLURL     string     `json:"html_url"`
	DiffURL     string     `json:"diff_url"`
	PatchURL    string     `json:"patch_url"`
	Draft       bool       `json:"draft"`
	Merged      bool       `json:"merged"`
	Mergeable   *bool      `json:"mergeable,omitempty"`
	MergeableState string  `json:"mergeable_state,omitempty"`
	Head        PRBranch   `json:"head"`
	Base        PRBranch   `json:"base"`
	User        User       `json:"user"`
	Labels      []Label    `json:"labels"`
	Reviewers   []User     `json:"requested_reviewers"`
	Commits     int        `json:"commits"`
	Additions   int        `json:"additions"`
	Deletions   int        `json:"deletions"`
	ChangedFiles int       `json:"changed_files"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
	MergedAt    *time.Time `json:"merged_at,omitempty"`
}

// PRBranch represents a branch reference in a pull request.
type PRBranch struct {
	Label string     `json:"label"`
	Ref   string     `json:"ref"`
	SHA   string     `json:"sha"`
	User  User       `json:"user"`
	Repo  Repository `json:"repo"`
}

// PRFile represents a file changed in a pull request.
type PRFile struct {
	SHA         string `json:"sha"`
	Filename    string `json:"filename"`
	Status      string `json:"status"`
	Additions   int    `json:"additions"`
	Deletions   int    `json:"deletions"`
	Changes     int    `json:"changes"`
	Patch       string `json:"patch,omitempty"`
	BlobURL     string `json:"blob_url"`
	RawURL      string `json:"raw_url"`
	ContentsURL string `json:"contents_url"`
}

// PRComment represents a comment on a pull request.
type PRComment struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	User      User      `json:"user"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	HTMLURL   string    `json:"html_url"`
}

// PRReview represents a review on a pull request.
type PRReview struct {
	ID          int64     `json:"id"`
	Body        string    `json:"body"`
	State       string    `json:"state"` // APPROVED, CHANGES_REQUESTED, COMMENTED, PENDING
	User        User      `json:"user"`
	SubmittedAt time.Time `json:"submitted_at"`
	HTMLURL     string    `json:"html_url"`
}

// ReviewComment represents a comment on a specific line in a PR review.
type ReviewComment struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	Path      string    `json:"path"`
	Position  *int      `json:"position,omitempty"`
	Line      *int      `json:"line,omitempty"`
	Side      string    `json:"side,omitempty"` // LEFT or RIGHT
	StartLine *int      `json:"start_line,omitempty"`
	StartSide string    `json:"start_side,omitempty"`
	User      User      `json:"user"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetPullRequest returns a pull request by number.
func (c *Client) GetPullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, error) {
	var pr PullRequest
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d", owner, repo, number)
	if err := c.get(ctx, path, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// ListPullRequests returns a list of pull requests.
func (c *Client) ListPullRequests(ctx context.Context, owner, repo string, state string) ([]PullRequest, error) {
	var prs []PullRequest
	path := fmt.Sprintf("/repos/%s/%s/pulls?state=%s", owner, repo, state)
	if err := c.get(ctx, path, &prs); err != nil {
		return nil, err
	}
	return prs, nil
}

// GetPullRequestFiles returns the files changed in a pull request.
func (c *Client) GetPullRequestFiles(ctx context.Context, owner, repo string, number int) ([]PRFile, error) {
	var files []PRFile
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/files", owner, repo, number)
	if err := c.get(ctx, path, &files); err != nil {
		return nil, err
	}
	return files, nil
}

// GetPullRequestDiff returns the diff for a pull request.
func (c *Client) GetPullRequestDiff(ctx context.Context, owner, repo string, number int) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d", c.baseURL, owner, repo, number)

	req, err := c.httpClient.Get(url)
	if err != nil {
		return "", err
	}
	// Re-create proper request with headers
	req.Header.Set("Accept", "application/vnd.github.diff")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	// Use simpler approach - the diff is in the diff_url field
	pr, err := c.GetPullRequest(ctx, owner, repo, number)
	if err != nil {
		return "", err
	}

	// Fetch the diff
	diffReq, err := c.httpClient.Get(pr.DiffURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch diff: %w", err)
	}
	defer func() { _ = diffReq.Body.Close() }()

	var diff []byte
	if _, err := diffReq.Body.Read(diff); err != nil {
		return "", err
	}

	return string(diff), nil
}

// CreatePRComment creates a comment on a pull request.
func (c *Client) CreatePRComment(ctx context.Context, owner, repo string, number int, body string) (*PRComment, error) {
	var comment PRComment
	path := fmt.Sprintf("/repos/%s/%s/issues/%d/comments", owner, repo, number)
	if err := c.post(ctx, path, map[string]string{"body": body}, &comment); err != nil {
		return nil, err
	}
	return &comment, nil
}

// CreateReviewInput contains the input for creating a review.
type CreateReviewInput struct {
	Body     string           `json:"body,omitempty"`
	Event    string           `json:"event"` // APPROVE, REQUEST_CHANGES, COMMENT
	Comments []ReviewCommentInput `json:"comments,omitempty"`
}

// ReviewCommentInput contains input for a review comment.
type ReviewCommentInput struct {
	Path      string `json:"path"`
	Body      string `json:"body"`
	Position  *int   `json:"position,omitempty"`
	Line      *int   `json:"line,omitempty"`
	Side      string `json:"side,omitempty"`
	StartLine *int   `json:"start_line,omitempty"`
	StartSide string `json:"start_side,omitempty"`
}

// CreateReview creates a review on a pull request.
func (c *Client) CreateReview(ctx context.Context, owner, repo string, number int, input CreateReviewInput) (*PRReview, error) {
	var review PRReview
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews", owner, repo, number)
	if err := c.post(ctx, path, input, &review); err != nil {
		return nil, err
	}
	return &review, nil
}

// ListReviews returns the reviews on a pull request.
func (c *Client) ListReviews(ctx context.Context, owner, repo string, number int) ([]PRReview, error) {
	var reviews []PRReview
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/reviews", owner, repo, number)
	if err := c.get(ctx, path, &reviews); err != nil {
		return nil, err
	}
	return reviews, nil
}

// AddLabels adds labels to a pull request.
func (c *Client) AddLabels(ctx context.Context, owner, repo string, number int, labels []string) ([]Label, error) {
	var result []Label
	path := fmt.Sprintf("/repos/%s/%s/issues/%d/labels", owner, repo, number)
	if err := c.post(ctx, path, map[string][]string{"labels": labels}, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// RemoveLabel removes a label from a pull request.
func (c *Client) RemoveLabel(ctx context.Context, owner, repo string, number int, label string) error {
	path := fmt.Sprintf("/repos/%s/%s/issues/%d/labels/%s", owner, repo, number, label)
	return c.delete(ctx, path)
}

// RequestReviewers requests reviewers for a pull request.
func (c *Client) RequestReviewers(ctx context.Context, owner, repo string, number int, reviewers []string) (*PullRequest, error) {
	var pr PullRequest
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/requested_reviewers", owner, repo, number)
	if err := c.post(ctx, path, map[string][]string{"reviewers": reviewers}, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// MergePullRequest merges a pull request.
func (c *Client) MergePullRequest(ctx context.Context, owner, repo string, number int, commitTitle, commitMessage, mergeMethod string) error {
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/merge", owner, repo, number)
	body := map[string]string{
		"merge_method": mergeMethod, // merge, squash, rebase
	}
	if commitTitle != "" {
		body["commit_title"] = commitTitle
	}
	if commitMessage != "" {
		body["commit_message"] = commitMessage
	}
	return c.request(ctx, "PUT", path, body, nil)
}

// Commit represents a Git commit.
type Commit struct {
	SHA     string     `json:"sha"`
	Message string     `json:"message"`
	Author  CommitUser `json:"author"`
	HTMLURL string     `json:"html_url"`
}

// CommitUser represents the author/committer of a commit.
type CommitUser struct {
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Date  time.Time `json:"date"`
}

// ListPRCommits returns the commits in a pull request.
func (c *Client) ListPRCommits(ctx context.Context, owner, repo string, number int) ([]Commit, error) {
	var commits []Commit
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d/commits", owner, repo, number)
	if err := c.get(ctx, path, &commits); err != nil {
		return nil, err
	}
	return commits, nil
}

// GetFileContent returns the content of a file at a specific ref.
func (c *Client) GetFileContent(ctx context.Context, owner, repo, path, ref string) (string, error) {
	type fileContent struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}

	var content fileContent
	apiPath := fmt.Sprintf("/repos/%s/%s/contents/%s?ref=%s", owner, repo, path, ref)
	if err := c.get(ctx, apiPath, &content); err != nil {
		return "", err
	}

	if content.Encoding == "base64" {
		// Decode base64 content
		// For simplicity, return as-is - can be decoded by caller
		return content.Content, nil
	}

	return content.Content, nil
}

// CheckStatus represents the status of CI checks on a commit.
type CheckStatus struct {
	State    string `json:"state"` // pending, success, failure, error
	Statuses []struct {
		State       string `json:"state"`
		Context     string `json:"context"`
		Description string `json:"description"`
		TargetURL   string `json:"target_url"`
	} `json:"statuses"`
	TotalCount int `json:"total_count"`
}

// GetCheckStatus returns the combined status of all checks on a commit.
func (c *Client) GetCheckStatus(ctx context.Context, owner, repo, ref string) (*CheckStatus, error) {
	var status CheckStatus
	path := fmt.Sprintf("/repos/%s/%s/commits/%s/status", owner, repo, ref)
	if err := c.get(ctx, path, &status); err != nil {
		return nil, err
	}
	return &status, nil
}
