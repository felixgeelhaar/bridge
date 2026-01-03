package governance

import (
	"context"
	"time"

	"github.com/felixgeelhaar/bridge/pkg/types"
)

// ApprovalStatus represents the status of an approval request.
type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"
	ApprovalStatusExpired  ApprovalStatus = "expired"
)

// Approval represents an approval request for a workflow.
type Approval struct {
	ID           types.ApprovalID
	RunID        types.RunID
	WorkflowID   types.WorkflowID
	WorkflowName string
	StepName     string
	Status       ApprovalStatus
	RequiredBy   string // Who/what triggered the approval requirement
	Reason       string // Why approval is needed
	Approvers    []string // List of allowed approvers
	ApprovedBy   string
	ApprovalNote string
	ExpiresAt    *time.Time
	RequestedAt  time.Time
	RespondedAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// NewApproval creates a new approval request.
func NewApproval(runID types.RunID, workflowID types.WorkflowID, workflowName, stepName, requiredBy, reason string) *Approval {
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour) // Default 24 hour expiry

	return &Approval{
		ID:           types.NewApprovalID(),
		RunID:        runID,
		WorkflowID:   workflowID,
		WorkflowName: workflowName,
		StepName:     stepName,
		Status:       ApprovalStatusPending,
		RequiredBy:   requiredBy,
		Reason:       reason,
		Approvers:    make([]string, 0),
		ExpiresAt:    &expiresAt,
		RequestedAt:  now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// Approve approves the request.
func (a *Approval) Approve(approvedBy, note string) error {
	if a.Status != ApprovalStatusPending {
		return types.ErrApprovalPending
	}

	if a.IsExpired() {
		a.Status = ApprovalStatusExpired
		return types.ErrApprovalExpired
	}

	now := time.Now()
	a.Status = ApprovalStatusApproved
	a.ApprovedBy = approvedBy
	a.ApprovalNote = note
	a.RespondedAt = &now
	a.UpdatedAt = now

	return nil
}

// Reject rejects the request.
func (a *Approval) Reject(rejectedBy, note string) error {
	if a.Status != ApprovalStatusPending {
		return types.ErrApprovalPending
	}

	now := time.Now()
	a.Status = ApprovalStatusRejected
	a.ApprovedBy = rejectedBy
	a.ApprovalNote = note
	a.RespondedAt = &now
	a.UpdatedAt = now

	return nil
}

// IsExpired returns true if the approval has expired.
func (a *Approval) IsExpired() bool {
	if a.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*a.ExpiresAt)
}

// IsPending returns true if the approval is pending.
func (a *Approval) IsPending() bool {
	return a.Status == ApprovalStatusPending && !a.IsExpired()
}

// CanApprove checks if a user can approve this request.
func (a *Approval) CanApprove(user string) bool {
	if len(a.Approvers) == 0 {
		return true // Anyone can approve if no specific approvers
	}

	for _, approver := range a.Approvers {
		if approver == user {
			return true
		}
	}
	return false
}

// SetExpiry sets the expiration time.
func (a *Approval) SetExpiry(duration time.Duration) {
	expiresAt := time.Now().Add(duration)
	a.ExpiresAt = &expiresAt
	a.UpdatedAt = time.Now()
}

// AddApprover adds an allowed approver.
func (a *Approval) AddApprover(approver string) {
	a.Approvers = append(a.Approvers, approver)
	a.UpdatedAt = time.Now()
}

// ApprovalRepository provides persistence for approvals.
type ApprovalRepository interface {
	Create(ctx context.Context, approval *Approval) error
	Get(ctx context.Context, id types.ApprovalID) (*Approval, error)
	GetByRunID(ctx context.Context, runID types.RunID) ([]*Approval, error)
	GetPending(ctx context.Context) ([]*Approval, error)
	Update(ctx context.Context, approval *Approval) error
}

// ApprovalService manages approval workflows.
type ApprovalService struct {
	repo      ApprovalRepository
	notifier  ApprovalNotifier
}

// ApprovalNotifier sends notifications about approval requests.
type ApprovalNotifier interface {
	NotifyApprovalRequired(ctx context.Context, approval *Approval) error
	NotifyApprovalResolved(ctx context.Context, approval *Approval) error
}

// NewApprovalService creates a new approval service.
func NewApprovalService(repo ApprovalRepository, notifier ApprovalNotifier) *ApprovalService {
	return &ApprovalService{
		repo:     repo,
		notifier: notifier,
	}
}

// RequestApproval creates a new approval request.
func (s *ApprovalService) RequestApproval(ctx context.Context, approval *Approval) error {
	if err := s.repo.Create(ctx, approval); err != nil {
		return err
	}

	if s.notifier != nil {
		return s.notifier.NotifyApprovalRequired(ctx, approval)
	}
	return nil
}

// Approve approves an approval request.
func (s *ApprovalService) Approve(ctx context.Context, id types.ApprovalID, approvedBy, note string) error {
	approval, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	if err := approval.Approve(approvedBy, note); err != nil {
		return err
	}

	if err := s.repo.Update(ctx, approval); err != nil {
		return err
	}

	if s.notifier != nil {
		return s.notifier.NotifyApprovalResolved(ctx, approval)
	}
	return nil
}

// Reject rejects an approval request.
func (s *ApprovalService) Reject(ctx context.Context, id types.ApprovalID, rejectedBy, note string) error {
	approval, err := s.repo.Get(ctx, id)
	if err != nil {
		return err
	}

	if err := approval.Reject(rejectedBy, note); err != nil {
		return err
	}

	if err := s.repo.Update(ctx, approval); err != nil {
		return err
	}

	if s.notifier != nil {
		return s.notifier.NotifyApprovalResolved(ctx, approval)
	}
	return nil
}
