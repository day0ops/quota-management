package db

import (
	"context"
	"fmt"
	"time"

	"github.com/agentgateway/quota-management/internal/models"
	"github.com/google/uuid"
)

// CreateBudgetApproval creates a new budget approval record.
func (r *Repository) CreateBudgetApproval(ctx context.Context, a *models.BudgetApproval) error {
	a.ID = uuid.New()
	a.CreatedAt = time.Now()
	a.UpdatedAt = time.Now()

	query := `
		INSERT INTO budget_approvals (id, budget_id, attempt_number, action, actor_user_id, actor_email, reason, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.Pool.Exec(ctx, query,
		a.ID, a.BudgetID, a.AttemptNumber, a.Action, a.ActorUserID, a.ActorEmail, a.Reason, a.CreatedAt, a.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create budget approval: %w", err)
	}

	return nil
}

// ListPendingApprovals returns budgets awaiting approval, optionally filtered by org.
func (r *Repository) ListPendingApprovals(ctx context.Context, orgID string, offset, limit int) ([]models.ApprovalWithBudget, int, error) {
	where := "WHERE bd.approval_status = 'pending'"
	args := []interface{}{}
	argIdx := 1

	if orgID != "" {
		where += fmt.Sprintf(" AND bd.owner_org_id = $%d", argIdx)
		args = append(args, orgID)
		argIdx++
	}

	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM budget_definitions bd %s`, where)
	var totalCount int
	if err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count pending approvals: %w", err)
	}

	dataQuery := fmt.Sprintf(`
		SELECT bd.id, bd.name, bd.budget_amount_usd, bd.period,
		       COALESCE(bd.owner_org_id, ''), COALESCE(bd.owner_team_id, ''),
		       bd.created_by_email, bd.approval_status, bd.rejection_count,
		       bd.created_at, bd.updated_at
		FROM budget_definitions bd
		%s
		ORDER BY bd.created_at ASC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	dataArgs := append(args, limit, offset)
	rows, err := r.db.Pool.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query pending approvals: %w", err)
	}
	defer rows.Close()

	var results []models.ApprovalWithBudget
	for rows.Next() {
		var a models.ApprovalWithBudget
		err := rows.Scan(
			&a.BudgetID, &a.BudgetName, &a.BudgetAmount, &a.BudgetPeriod,
			&a.OwnerOrgID, &a.OwnerTeamID,
			&a.CreatedByEmail, &a.ApprovalStatus, &a.RejectionCount,
			&a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan pending approval: %w", err)
		}
		results = append(results, a)
	}

	return results, totalCount, nil
}

// CountPendingApprovals returns the count of pending approvals for an org.
func (r *Repository) CountPendingApprovals(ctx context.Context, orgID string) (int, error) {
	query := `SELECT COUNT(*) FROM budget_definitions WHERE approval_status = 'pending' AND owner_org_id = $1`
	var count int
	if err := r.db.Pool.QueryRow(ctx, query, orgID).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to count pending approvals: %w", err)
	}
	return count, nil
}

// UpdateBudgetApprovalStatus updates the approval_status and rejection_count on a budget.
func (r *Repository) UpdateBudgetApprovalStatus(ctx context.Context, budgetID uuid.UUID, status models.ApprovalStatus, incrementRejection bool) error {
	var query string
	if incrementRejection {
		query = `
			UPDATE budget_definitions
			SET approval_status = $2, rejection_count = rejection_count + 1, updated_at = NOW()
			WHERE id = $1
		`
	} else {
		query = `
			UPDATE budget_definitions
			SET approval_status = $2, updated_at = NOW()
			WHERE id = $1
		`
	}

	result, err := r.db.Pool.Exec(ctx, query, budgetID, status)
	if err != nil {
		return fmt.Errorf("failed to update budget approval status: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// ListApprovalHistory returns the approval history for a budget.
func (r *Repository) ListApprovalHistory(ctx context.Context, budgetID uuid.UUID) ([]models.BudgetApproval, error) {
	query := `
		SELECT id, budget_id, attempt_number, action, actor_user_id, actor_email, reason, created_at, updated_at
		FROM budget_approvals
		WHERE budget_id = $1
		ORDER BY attempt_number ASC, created_at ASC
	`

	rows, err := r.db.Pool.Query(ctx, query, budgetID)
	if err != nil {
		return nil, fmt.Errorf("failed to query approval history: %w", err)
	}
	defer rows.Close()

	var approvals []models.BudgetApproval
	for rows.Next() {
		var a models.BudgetApproval
		err := rows.Scan(
			&a.ID, &a.BudgetID, &a.AttemptNumber, &a.Action, &a.ActorUserID, &a.ActorEmail, &a.Reason, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan approval: %w", err)
		}
		approvals = append(approvals, a)
	}

	return approvals, nil
}

// ApprovalHistoryFilter holds filter params for listing approval history.
type ApprovalHistoryFilter struct {
	OrgID  string
	TeamID string
}

// ListAllApprovalHistory returns paginated global approval history joined with budget details,
// filtered by org (for org admins) or team (for team members).
func (r *Repository) ListAllApprovalHistory(ctx context.Context, filter ApprovalHistoryFilter, offset, limit int) ([]models.ApprovalWithBudget, int, error) {
	where := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1

	if filter.TeamID != "" {
		// Team members see only their team's budget approval history
		where += fmt.Sprintf(" AND bd.owner_team_id = $%d", argIdx)
		args = append(args, filter.TeamID)
		argIdx++
	} else if filter.OrgID != "" {
		// Org admins see all budgets in their org
		where += fmt.Sprintf(" AND bd.owner_org_id = $%d", argIdx)
		args = append(args, filter.OrgID)
		argIdx++
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM budget_approvals ba
		JOIN budget_definitions bd ON ba.budget_id = bd.id
		%s
	`, where)
	var totalCount int
	if err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count approval history: %w", err)
	}

	dataQuery := fmt.Sprintf(`
		SELECT ba.id, ba.budget_id, ba.attempt_number, ba.action,
		       ba.actor_user_id, ba.actor_email, ba.reason, ba.created_at, ba.updated_at,
		       bd.name, bd.budget_amount_usd, bd.period,
		       COALESCE(bd.owner_org_id, ''), COALESCE(bd.owner_team_id, ''),
		       COALESCE(bd.created_by_email, ''), bd.approval_status, bd.rejection_count
		FROM budget_approvals ba
		JOIN budget_definitions bd ON ba.budget_id = bd.id
		%s
		ORDER BY ba.created_at DESC
		LIMIT $%d OFFSET $%d
	`, where, argIdx, argIdx+1)

	dataArgs := append(args, limit, offset)
	rows, err := r.db.Pool.Query(ctx, dataQuery, dataArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query approval history: %w", err)
	}
	defer rows.Close()

	var results []models.ApprovalWithBudget
	for rows.Next() {
		var a models.ApprovalWithBudget
		err := rows.Scan(
			&a.ID, &a.BudgetID, &a.AttemptNumber, &a.Action,
			&a.ActorUserID, &a.ActorEmail, &a.Reason, &a.CreatedAt, &a.UpdatedAt,
			&a.BudgetName, &a.BudgetAmount, &a.BudgetPeriod,
			&a.OwnerOrgID, &a.OwnerTeamID,
			&a.CreatedByEmail, &a.ApprovalStatus, &a.RejectionCount,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan approval history: %w", err)
		}
		results = append(results, a)
	}

	return results, totalCount, nil
}
