package db

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/agentgateway/quota-management/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// RateLimitRepository provides database operations for rate limit allocations.
type RateLimitRepository struct {
	db *DB
}

// NewRateLimitRepository creates a new rate limit repository.
func NewRateLimitRepository(db *DB) *RateLimitRepository {
	return &RateLimitRepository{db: db}
}

// ListAllocations returns all rate limit allocations.
func (r *RateLimitRepository) ListAllocations(ctx context.Context) ([]models.RateLimitAllocation, error) {
	query := `
		SELECT id, org_id, team_id, model_pattern, token_limit, token_unit,
		       request_limit, request_unit, burst_percentage, enforcement,
		       enabled, disabled_by_user_id, disabled_by_email, disabled_by_is_org, disabled_at,
		       approval_status, approved_by, approved_at,
		       created_by_user_id, created_by_email, description,
		       rejection_count, version, created_at, updated_at
		FROM rate_limit_allocations
		ORDER BY org_id, team_id, model_pattern
	`

	rows, err := r.db.Pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query rate limit allocations: %w", err)
	}
	defer rows.Close()

	return r.scanAllocations(rows)
}

// GetAllocationByID returns an allocation by ID.
func (r *RateLimitRepository) GetAllocationByID(ctx context.Context, id uuid.UUID) (*models.RateLimitAllocation, error) {
	query := `
		SELECT id, org_id, team_id, model_pattern, token_limit, token_unit,
		       request_limit, request_unit, burst_percentage, enforcement,
		       enabled, disabled_by_user_id, disabled_by_email, disabled_by_is_org, disabled_at,
		       approval_status, approved_by, approved_at,
		       created_by_user_id, created_by_email, description,
		       rejection_count, version, created_at, updated_at
		FROM rate_limit_allocations
		WHERE id = $1
	`

	var a models.RateLimitAllocation
	err := r.db.Pool.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.OrgID, &a.TeamID, &a.ModelPattern, &a.TokenLimit, &a.TokenUnit,
		&a.RequestLimit, &a.RequestUnit, &a.BurstPercentage, &a.Enforcement,
		&a.Enabled, &a.DisabledByUserID, &a.DisabledByEmail, &a.DisabledByIsOrg, &a.DisabledAt,
		&a.ApprovalStatus, &a.ApprovedBy, &a.ApprovedAt,
		&a.CreatedByUserID, &a.CreatedByEmail, &a.Description,
		&a.RejectionCount, &a.Version, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get rate limit allocation: %w", err)
	}

	return &a, nil
}

// GetAllocationsForTeamModel returns all matching allocations for a team and model,
// ordered by specificity: exact match > pattern > wildcard.
// Multiple allocations may match (e.g. one with token limit, one with request limit),
// and callers should merge them to get the effective limits.
func (r *RateLimitRepository) GetAllocationsForTeamModel(ctx context.Context, teamID, model string) ([]models.RateLimitAllocation, error) {
	query := `
		SELECT id, org_id, team_id, model_pattern, token_limit, token_unit,
		       request_limit, request_unit, burst_percentage, enforcement,
		       enabled, disabled_by_user_id, disabled_by_email, disabled_by_is_org, disabled_at,
		       approval_status, approved_by, approved_at,
		       created_by_user_id, created_by_email, description,
		       rejection_count, version, created_at, updated_at
		FROM rate_limit_allocations
		WHERE team_id = $1
		  AND enabled = true
		  AND approval_status = 'approved'
		  AND (model_pattern = $2 OR model_pattern = '*' OR $2 LIKE replace(model_pattern, '*', '%'))
		ORDER BY
		  CASE WHEN model_pattern = $2 THEN 0
		       WHEN model_pattern = '*' THEN 2
		       ELSE 1 END
	`

	rows, err := r.db.Pool.Query(ctx, query, teamID, model)
	if err != nil {
		return nil, fmt.Errorf("failed to get allocations for team/model: %w", err)
	}
	defer rows.Close()

	var allocations []models.RateLimitAllocation
	for rows.Next() {
		var a models.RateLimitAllocation
		if err := rows.Scan(
			&a.ID, &a.OrgID, &a.TeamID, &a.ModelPattern, &a.TokenLimit, &a.TokenUnit,
			&a.RequestLimit, &a.RequestUnit, &a.BurstPercentage, &a.Enforcement,
			&a.Enabled, &a.DisabledByUserID, &a.DisabledByEmail, &a.DisabledByIsOrg, &a.DisabledAt,
			&a.ApprovalStatus, &a.ApprovedBy, &a.ApprovedAt,
			&a.CreatedByUserID, &a.CreatedByEmail, &a.Description,
			&a.RejectionCount, &a.Version, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan allocation: %w", err)
		}
		allocations = append(allocations, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate allocations: %w", err)
	}

	return allocations, nil
}

// GetAllocationByTeamAndPattern returns an existing allocation for a team and model pattern.
// Returns nil if no allocation exists.
func (r *RateLimitRepository) GetAllocationByTeamAndPattern(ctx context.Context, teamID, modelPattern string) (*models.RateLimitAllocation, error) {
	query := `
		SELECT id, org_id, team_id, model_pattern, token_limit, token_unit,
		       request_limit, request_unit, burst_percentage, enforcement,
		       enabled, disabled_by_user_id, disabled_by_email, disabled_by_is_org, disabled_at,
		       approval_status, approved_by, approved_at,
		       created_by_user_id, created_by_email, description,
		       rejection_count, version, created_at, updated_at
		FROM rate_limit_allocations
		WHERE team_id = $1 AND model_pattern = $2
	`

	var a models.RateLimitAllocation
	err := r.db.Pool.QueryRow(ctx, query, teamID, modelPattern).Scan(
		&a.ID, &a.OrgID, &a.TeamID, &a.ModelPattern, &a.TokenLimit, &a.TokenUnit,
		&a.RequestLimit, &a.RequestUnit, &a.BurstPercentage, &a.Enforcement,
		&a.Enabled, &a.DisabledByUserID, &a.DisabledByEmail, &a.DisabledByIsOrg, &a.DisabledAt,
		&a.ApprovalStatus, &a.ApprovedBy, &a.ApprovedAt,
		&a.CreatedByUserID, &a.CreatedByEmail, &a.Description,
		&a.RejectionCount, &a.Version, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get allocation by team and pattern: %w", err)
	}

	return &a, nil
}

// CreateAllocation creates a new rate limit allocation.
// If a rejected allocation exists for the same org/team/model_pattern, it is deleted first.
// Returns ErrDuplicateEntity if an active/pending allocation already exists for the team+pattern.
func (r *RateLimitRepository) CreateAllocation(ctx context.Context, a *models.RateLimitAllocation) error {
	a.ID = uuid.New()
	a.Version = 1
	a.CreatedAt = time.Now()
	a.UpdatedAt = time.Now()
	if a.Enforcement == "" {
		a.Enforcement = models.EnforcementEnforced
	}
	if a.ApprovalStatus == "" {
		a.ApprovalStatus = models.ApprovalStatusPending
	}

	// Check for existing non-rejected allocation for the same team+pattern
	existingQuery := `
		SELECT id FROM rate_limit_allocations
		WHERE team_id = $1 AND model_pattern = $2 AND approval_status != 'rejected'
		LIMIT 1
	`
	var existingID uuid.UUID
	err := r.db.Pool.QueryRow(ctx, existingQuery, a.TeamID, a.ModelPattern).Scan(&existingID)
	if err == nil {
		// Found existing allocation
		return ErrDuplicateEntity
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to check for existing allocation: %w", err)
	}

	// Delete any rejected allocations for the same org/team/model_pattern to allow resubmission
	deleteQuery := `
		DELETE FROM rate_limit_allocations
		WHERE org_id = $1 AND team_id = $2 AND model_pattern = $3 AND approval_status = 'rejected'
	`
	_, _ = r.db.Pool.Exec(ctx, deleteQuery, a.OrgID, a.TeamID, a.ModelPattern)

	query := `
		INSERT INTO rate_limit_allocations (
			id, org_id, team_id, model_pattern, token_limit, token_unit,
			request_limit, request_unit, burst_percentage, enforcement,
			enabled, disabled_by_user_id, disabled_by_email, disabled_by_is_org, disabled_at,
			approval_status, approved_by, approved_at,
			created_by_user_id, created_by_email, description,
			version, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
	`

	_, err = r.db.Pool.Exec(ctx, query,
		a.ID, a.OrgID, a.TeamID, a.ModelPattern, a.TokenLimit, a.TokenUnit,
		a.RequestLimit, a.RequestUnit, a.BurstPercentage, a.Enforcement,
		a.Enabled, a.DisabledByUserID, a.DisabledByEmail, a.DisabledByIsOrg, a.DisabledAt,
		a.ApprovalStatus, a.ApprovedBy, a.ApprovedAt,
		a.CreatedByUserID, a.CreatedByEmail, a.Description,
		a.Version, a.CreatedAt, a.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create rate limit allocation: %w", err)
	}

	return nil
}

// UpdateAllocation updates a rate limit allocation with optimistic locking.
func (r *RateLimitRepository) UpdateAllocation(ctx context.Context, a *models.RateLimitAllocation) error {
	a.UpdatedAt = time.Now()

	query := `
		UPDATE rate_limit_allocations
		SET model_pattern = $2, token_limit = $3, token_unit = $4,
		    request_limit = $5, request_unit = $6, burst_percentage = $7,
		    enforcement = $8, enabled = $9, disabled_by_user_id = $10,
		    disabled_by_email = $11, disabled_by_is_org = $12, disabled_at = $13,
		    description = $14, updated_at = $15, version = version + 1
		WHERE id = $1 AND version = $16
	`

	result, err := r.db.Pool.Exec(ctx, query,
		a.ID, a.ModelPattern, a.TokenLimit, a.TokenUnit,
		a.RequestLimit, a.RequestUnit, a.BurstPercentage,
		a.Enforcement, a.Enabled, a.DisabledByUserID,
		a.DisabledByEmail, a.DisabledByIsOrg, a.DisabledAt,
		a.Description, a.UpdatedAt, a.Version,
	)
	if err != nil {
		return fmt.Errorf("failed to update rate limit allocation: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrOptimisticLock
	}

	return nil
}

// ApproveAllocation approves a rate limit allocation.
func (r *RateLimitRepository) ApproveAllocation(ctx context.Context, id uuid.UUID, approvedBy string, enforcement models.Enforcement) error {
	query := `
		UPDATE rate_limit_allocations
		SET approval_status = 'approved', approved_by = $2, approved_at = $3,
		    enforcement = $4, updated_at = $3, version = version + 1
		WHERE id = $1 AND approval_status = 'pending'
	`

	now := time.Now()
	result, err := r.db.Pool.Exec(ctx, query, id, approvedBy, now, enforcement)
	if err != nil {
		return fmt.Errorf("failed to approve allocation: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// RejectAllocation rejects a rate limit allocation and increments rejection count.
func (r *RateLimitRepository) RejectAllocation(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE rate_limit_allocations
		SET approval_status = 'rejected', rejection_count = rejection_count + 1,
		    updated_at = NOW(), version = version + 1
		WHERE id = $1 AND approval_status = 'pending'
	`

	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to reject allocation: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// DeleteAllocation deletes a rate limit allocation.
func (r *RateLimitRepository) DeleteAllocation(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM rate_limit_allocations WHERE id = $1`

	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete rate limit allocation: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// CountPendingAllocations returns the count of pending allocations for an org.
func (r *RateLimitRepository) CountPendingAllocations(ctx context.Context, orgID string) (int, error) {
	query := `SELECT COUNT(*) FROM rate_limit_allocations WHERE org_id = $1 AND approval_status = 'pending'`
	var count int
	err := r.db.Pool.QueryRow(ctx, query, orgID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count pending allocations: %w", err)
	}
	return count, nil
}

// ListPendingAllocations returns all pending allocations for an org.
func (r *RateLimitRepository) ListPendingAllocations(ctx context.Context, orgID string) ([]models.RateLimitAllocation, error) {
	query := `
		SELECT id, org_id, team_id, model_pattern, token_limit, token_unit,
		       request_limit, request_unit, burst_percentage, enforcement,
		       enabled, disabled_by_user_id, disabled_by_email, disabled_by_is_org, disabled_at,
		       approval_status, approved_by, approved_at,
		       created_by_user_id, created_by_email, description,
		       rejection_count, version, created_at, updated_at
		FROM rate_limit_allocations
		WHERE org_id = $1 AND approval_status = 'pending'
		ORDER BY created_at DESC
	`

	rows, err := r.db.Pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending allocations: %w", err)
	}
	defer rows.Close()

	return r.scanAllocations(rows)
}

// RateLimitListFilter holds filter params for listing rate limit allocations.
type RateLimitListFilter struct {
	EnabledOnly bool
}

// ListAllocationsByOrg returns all allocations for an org.
func (r *RateLimitRepository) ListAllocationsByOrg(ctx context.Context, orgID string) ([]models.RateLimitAllocation, error) {
	return r.ListAllocationsByOrgFiltered(ctx, orgID, RateLimitListFilter{})
}

// ListAllocationsByOrgFiltered returns allocations for an org with optional filtering.
func (r *RateLimitRepository) ListAllocationsByOrgFiltered(ctx context.Context, orgID string, filter RateLimitListFilter) ([]models.RateLimitAllocation, error) {
	where := "WHERE org_id = $1"
	if filter.EnabledOnly {
		where += " AND enabled = true AND approval_status = 'approved'"
	}

	query := fmt.Sprintf(`
		SELECT id, org_id, team_id, model_pattern, token_limit, token_unit,
		       request_limit, request_unit, burst_percentage, enforcement,
		       enabled, disabled_by_user_id, disabled_by_email, disabled_by_is_org, disabled_at,
		       approval_status, approved_by, approved_at,
		       created_by_user_id, created_by_email, description,
		       rejection_count, version, created_at, updated_at
		FROM rate_limit_allocations
		%s
		ORDER BY team_id, model_pattern
	`, where)

	rows, err := r.db.Pool.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to query allocations by org: %w", err)
	}
	defer rows.Close()

	return r.scanAllocations(rows)
}

func (r *RateLimitRepository) scanAllocations(rows pgx.Rows) ([]models.RateLimitAllocation, error) {
	var allocations []models.RateLimitAllocation
	for rows.Next() {
		var a models.RateLimitAllocation
		err := rows.Scan(
			&a.ID, &a.OrgID, &a.TeamID, &a.ModelPattern, &a.TokenLimit, &a.TokenUnit,
			&a.RequestLimit, &a.RequestUnit, &a.BurstPercentage, &a.Enforcement,
			&a.Enabled, &a.DisabledByUserID, &a.DisabledByEmail, &a.DisabledByIsOrg, &a.DisabledAt,
			&a.ApprovalStatus, &a.ApprovedBy, &a.ApprovedAt,
			&a.CreatedByUserID, &a.CreatedByEmail, &a.Description,
			&a.RejectionCount, &a.Version, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan allocation: %w", err)
		}
		allocations = append(allocations, a)
	}
	return allocations, nil
}

// CreateApproval creates a new approval record.
func (r *RateLimitRepository) CreateApproval(ctx context.Context, approval *models.RateLimitApproval) error {
	approval.ID = uuid.New()
	approval.CreatedAt = time.Now()
	approval.UpdatedAt = time.Now()

	query := `
		INSERT INTO rate_limit_approvals (id, allocation_id, attempt_number, action, actor_user_id, actor_email, reason, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.Pool.Exec(ctx, query,
		approval.ID, approval.AllocationID, approval.AttemptNumber, approval.Action,
		approval.ActorUserID, approval.ActorEmail, approval.Reason,
		approval.CreatedAt, approval.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create approval: %w", err)
	}
	return nil
}

// GetLatestAttemptNumber returns the latest attempt number for an allocation.
func (r *RateLimitRepository) GetLatestAttemptNumber(ctx context.Context, allocationID uuid.UUID) (int, error) {
	query := `SELECT COALESCE(MAX(attempt_number), 0) FROM rate_limit_approvals WHERE allocation_id = $1`
	var attemptNumber int
	err := r.db.Pool.QueryRow(ctx, query, allocationID).Scan(&attemptNumber)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest attempt number: %w", err)
	}
	return attemptNumber, nil
}

// RateLimitApprovalHistoryFilter holds filter params for listing rate limit approval history.
type RateLimitApprovalHistoryFilter struct {
	OrgID  string
	TeamID string
}

// ListApprovalHistory returns approval history filtered by org (for org admins) or team (for team members).
func (r *RateLimitRepository) ListApprovalHistory(ctx context.Context, filter RateLimitApprovalHistoryFilter, page, pageSize int) ([]models.RateLimitApprovalWithAllocation, int, error) {
	offset := (page - 1) * pageSize

	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if filter.TeamID != "" {
		// Team members see only their team's approval history
		where += fmt.Sprintf(" AND r.team_id = $%d", argIdx)
		args = append(args, filter.TeamID)
		argIdx++
	} else if filter.OrgID != "" {
		// Org admins see all approval history in their org
		where += fmt.Sprintf(" AND r.org_id = $%d", argIdx)
		args = append(args, filter.OrgID)
		argIdx++
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM rate_limit_approvals a
		JOIN rate_limit_allocations r ON a.allocation_id = r.id
		%s
	`, where)
	var total int
	if err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count approval history: %w", err)
	}

	query := fmt.Sprintf(`
		SELECT a.id, a.allocation_id, a.attempt_number, a.action, a.actor_user_id, a.actor_email,
		       a.reason, a.created_at, a.updated_at, r.team_id, r.model_pattern, r.org_id
		FROM rate_limit_approvals a
		JOIN rate_limit_allocations r ON a.allocation_id = r.id
		%s
		ORDER BY a.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	dataArgs := append(args, pageSize, offset)
	rows, err := r.db.Pool.Query(ctx, query, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query approval history: %w", err)
	}
	defer rows.Close()

	var approvals []models.RateLimitApprovalWithAllocation
	for rows.Next() {
		var a models.RateLimitApprovalWithAllocation
		err := rows.Scan(
			&a.ID, &a.AllocationID, &a.AttemptNumber, &a.Action, &a.ActorUserID, &a.ActorEmail,
			&a.Reason, &a.CreatedAt, &a.UpdatedAt, &a.TeamID, &a.ModelPattern, &a.OrgID,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan approval: %w", err)
		}
		approvals = append(approvals, a)
	}

	return approvals, total, nil
}

// ResetAllocationForResubmit resets an allocation's approval status to pending for resubmission.
func (r *RateLimitRepository) ResetAllocationForResubmit(ctx context.Context, id uuid.UUID) error {
	query := `
		UPDATE rate_limit_allocations
		SET approval_status = 'pending', approved_by = NULL, approved_at = NULL,
		    updated_at = NOW(), version = version + 1
		WHERE id = $1 AND approval_status = 'rejected'
	`

	result, err := r.db.Pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to reset allocation for resubmit: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}
