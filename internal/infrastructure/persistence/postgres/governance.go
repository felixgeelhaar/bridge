package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/felixgeelhaar/bolt"
	"github.com/felixgeelhaar/bridge/internal/domain/governance"
	"github.com/felixgeelhaar/bridge/internal/infrastructure/persistence/postgres/sqlc"
	"github.com/felixgeelhaar/bridge/pkg/types"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// GovernanceRepository implements governance.Repository using PostgreSQL.
type GovernanceRepository struct {
	pool    *pgxpool.Pool
	queries *sqlc.Queries
	logger  *bolt.Logger
}

// NewGovernanceRepository creates a new PostgreSQL governance repository.
func NewGovernanceRepository(pool *pgxpool.Pool, logger *bolt.Logger) *GovernanceRepository {
	return &GovernanceRepository{
		pool:    pool,
		queries: sqlc.New(pool),
		logger:  logger,
	}
}

// Create creates a new policy bundle.
func (r *GovernanceRepository) Create(ctx context.Context, bundle *governance.PolicyBundle) error {
	rules, err := json.Marshal(bundle.Rules)
	if err != nil {
		return fmt.Errorf("failed to marshal rules: %w", err)
	}

	_, err = r.queries.CreatePolicyBundle(ctx, sqlc.CreatePolicyBundleParams{
		ID:          bundle.ID.String(),
		Name:        bundle.Name,
		Version:     bundle.Version,
		Description: strPtr(bundle.Description),
		Rules:       rules,
		Checksum:    strPtr(bundle.Checksum),
		Active:      bundle.Active,
		CreatedAt:   timeToPgTimestamptzValue(bundle.CreatedAt),
		UpdatedAt:   timeToPgTimestamptzValue(bundle.UpdatedAt),
	})
	if err != nil {
		return fmt.Errorf("failed to create policy bundle: %w", err)
	}

	r.logger.Debug().
		Str("policy_id", bundle.ID.String()).
		Str("name", bundle.Name).
		Msg("Created policy bundle")

	return nil
}

// Get retrieves a policy bundle by ID.
func (r *GovernanceRepository) Get(ctx context.Context, id types.PolicyID) (*governance.PolicyBundle, error) {
	row, err := r.queries.GetPolicyBundle(ctx, id.String())
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("policy bundle not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get policy bundle: %w", err)
	}

	return r.rowToBundle(row)
}

// GetByName retrieves a policy bundle by name.
func (r *GovernanceRepository) GetByName(ctx context.Context, name string) (*governance.PolicyBundle, error) {
	row, err := r.queries.GetPolicyBundleByName(ctx, name)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("policy bundle not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get policy bundle: %w", err)
	}

	return r.rowToBundle(row)
}

// List lists policy bundles.
func (r *GovernanceRepository) List(ctx context.Context, activeOnly bool) ([]*governance.PolicyBundle, error) {
	var rows []sqlc.PolicyBundle
	var err error

	if activeOnly {
		rows, err = r.queries.ListActivePolicyBundles(ctx)
	} else {
		rows, err = r.queries.ListPolicyBundles(ctx, sqlc.ListPolicyBundlesParams{
			Limit:  1000,
			Offset: 0,
		})
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list policy bundles: %w", err)
	}

	bundles := make([]*governance.PolicyBundle, 0, len(rows))
	for _, row := range rows {
		bundle, err := r.rowToBundle(row)
		if err != nil {
			return nil, err
		}
		bundles = append(bundles, bundle)
	}

	return bundles, nil
}

// Update updates a policy bundle.
func (r *GovernanceRepository) Update(ctx context.Context, bundle *governance.PolicyBundle) error {
	rules, err := json.Marshal(bundle.Rules)
	if err != nil {
		return fmt.Errorf("failed to marshal rules: %w", err)
	}

	_, err = r.queries.UpdatePolicyBundle(ctx, sqlc.UpdatePolicyBundleParams{
		ID:          bundle.ID.String(),
		Name:        bundle.Name,
		Version:     bundle.Version,
		Description: strPtr(bundle.Description),
		Rules:       rules,
		Checksum:    strPtr(bundle.Checksum),
		Active:      bundle.Active,
	})
	if err != nil {
		return fmt.Errorf("failed to update policy bundle: %w", err)
	}

	return nil
}

// Delete deletes a policy bundle.
func (r *GovernanceRepository) Delete(ctx context.Context, id types.PolicyID) error {
	if err := r.queries.DeletePolicyBundle(ctx, id.String()); err != nil {
		return fmt.Errorf("failed to delete policy bundle: %w", err)
	}
	return nil
}

func (r *GovernanceRepository) rowToBundle(row sqlc.PolicyBundle) (*governance.PolicyBundle, error) {
	var rules []governance.PolicyRule
	if len(row.Rules) > 0 {
		if err := json.Unmarshal(row.Rules, &rules); err != nil {
			return nil, fmt.Errorf("failed to unmarshal rules: %w", err)
		}
	}

	return &governance.PolicyBundle{
		ID:          types.PolicyID(row.ID),
		Name:        row.Name,
		Version:     row.Version,
		Description: ptrStr(row.Description),
		Rules:       rules,
		Checksum:    ptrStr(row.Checksum),
		Active:      row.Active,
		CreatedAt:   pgTimestamptzToTime(row.CreatedAt),
		UpdatedAt:   pgTimestamptzToTime(row.UpdatedAt),
	}, nil
}

// Ensure GovernanceRepository implements governance.Repository.
var _ governance.Repository = (*GovernanceRepository)(nil)
