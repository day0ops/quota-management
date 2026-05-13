package db

import (
	"context"
	"fmt"
	"time"

	"github.com/agentgateway/quota-management/internal/models"
	"github.com/google/uuid"
)

// InsertAlertIfNotExists inserts an alert row, silently skipping if one already
// exists for the same (budget_id, alert_type, period_start).
func (r *Repository) InsertAlertIfNotExists(ctx context.Context, a *models.BudgetAlert) error {
	query := `
		INSERT INTO budget_alerts
			(budget_id, alert_type, status, period_start,
			 triggered_usage_usd, budget_amount_usd, threshold_pct,
			 projected_spend_usd, days_until_exhausted)
		VALUES ($1, $2, 'active', $3, $4, $5, $6, $7, $8)
		ON CONFLICT (budget_id, alert_type, period_start) DO NOTHING`
	_, err := r.db.Pool.Exec(ctx, query,
		a.BudgetID, string(a.AlertType), a.PeriodStart,
		a.TriggeredUsageUSD, a.BudgetAmountUSD,
		a.ThresholdPct, a.ProjectedSpendUSD, a.DaysUntilExhausted,
	)
	if err != nil {
		return fmt.Errorf("insert alert if not exists: %w", err)
	}
	return nil
}

// AlertListFilter holds query parameters for listing alerts.
type AlertListFilter struct {
	Status       string // "active" | "resolved" | "dismissed" | "all"
	AlertType    string
	BudgetID     *uuid.UUID
	IsOrgAdmin   bool
	CallerOrgID  string
	CallerTeamID string
	Page         int
	PageSize     int
}

// ListAlerts returns paginated alerts joined with budget name and entity_type.
func (r *Repository) ListAlerts(ctx context.Context, f AlertListFilter) ([]models.BudgetAlert, int64, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PageSize < 1 || f.PageSize > 100 {
		f.PageSize = 20
	}
	offset := (f.Page - 1) * f.PageSize

	where := "1=1"
	args := []any{}
	argIdx := 1

	if f.Status != "" && f.Status != "all" {
		where += fmt.Sprintf(" AND a.status = $%d", argIdx)
		args = append(args, f.Status)
		argIdx++
	}
	if f.AlertType != "" {
		where += fmt.Sprintf(" AND a.alert_type = $%d", argIdx)
		args = append(args, f.AlertType)
		argIdx++
	}
	if f.BudgetID != nil {
		where += fmt.Sprintf(" AND a.budget_id = $%d", argIdx)
		args = append(args, *f.BudgetID)
		argIdx++
	}
	if !f.IsOrgAdmin {
		where += fmt.Sprintf(" AND b.owner_org_id = $%d AND b.owner_team_id = $%d", argIdx, argIdx+1)
		args = append(args, f.CallerOrgID, f.CallerTeamID)
		argIdx += 2
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM budget_alerts a
		JOIN budget_definitions b ON b.id = a.budget_id
		WHERE %s`, where)

	var total int64
	if err := r.db.Pool.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count alerts: %w", err)
	}

	listArgs := make([]any, len(args), len(args)+2)
	copy(listArgs, args)
	listArgs = append(listArgs, f.PageSize, offset)
	query := fmt.Sprintf(`
		SELECT
			a.id, a.budget_id, a.alert_type, a.status,
			a.triggered_usage_usd, a.budget_amount_usd, a.threshold_pct,
			a.projected_spend_usd, a.days_until_exhausted,
			a.dismissed_by_user_id, a.dismissed_at, a.resolved_at,
			a.period_start, a.created_at, a.updated_at,
			b.name, b.entity_type
		FROM budget_alerts a
		JOIN budget_definitions b ON b.id = a.budget_id
		WHERE %s
		ORDER BY a.created_at DESC
		LIMIT $%d OFFSET $%d`, where, argIdx, argIdx+1)

	rows, err := r.db.Pool.Query(ctx, query, listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("list alerts: %w", err)
	}
	defer rows.Close()

	var alerts []models.BudgetAlert
	for rows.Next() {
		var a models.BudgetAlert
		var alertType, alertStatus, entityType string
		if err := rows.Scan(
			&a.ID, &a.BudgetID, &alertType, &alertStatus,
			&a.TriggeredUsageUSD, &a.BudgetAmountUSD, &a.ThresholdPct,
			&a.ProjectedSpendUSD, &a.DaysUntilExhausted,
			&a.DismissedByUserID, &a.DismissedAt, &a.ResolvedAt,
			&a.PeriodStart, &a.CreatedAt, &a.UpdatedAt,
			&a.BudgetName, &entityType,
		); err != nil {
			return nil, 0, fmt.Errorf("scan alert: %w", err)
		}
		a.AlertType = models.AlertType(alertType)
		a.Status = models.AlertStatus(alertStatus)
		a.EntityType = entityType
		alerts = append(alerts, a)
	}
	return alerts, total, rows.Err()
}

// CountActiveAlerts returns the count of active alerts visible to the caller.
func (r *Repository) CountActiveAlerts(ctx context.Context, isOrgAdmin bool, callerOrgID, callerTeamID string) (int64, error) {
	var count int64
	if isOrgAdmin {
		err := r.db.Pool.QueryRow(ctx,
			`SELECT COUNT(*) FROM budget_alerts WHERE status = 'active'`,
		).Scan(&count)
		if err != nil {
			return count, fmt.Errorf("count active alerts: %w", err)
		}
		return count, nil
	}
	err := r.db.Pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM budget_alerts a
		JOIN budget_definitions b ON b.id = a.budget_id
		WHERE a.status = 'active'
		  AND b.owner_org_id = $1
		  AND b.owner_team_id = $2`,
		callerOrgID, callerTeamID,
	).Scan(&count)
	if err != nil {
		return count, fmt.Errorf("count active alerts: %w", err)
	}
	return count, nil
}

// DismissAlert sets a single alert to dismissed status.
// Returns ErrNotFound if the alert does not exist or is not active.
func (r *Repository) DismissAlert(ctx context.Context, id uuid.UUID, userID string) error {
	result, err := r.db.Pool.Exec(ctx, `
		UPDATE budget_alerts
		SET status = 'dismissed', dismissed_by_user_id = $2,
		    dismissed_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'active'`,
		id, userID,
	)
	if err != nil {
		return fmt.Errorf("dismiss alert: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DismissAllAlerts dismisses all active alerts visible to the caller.
func (r *Repository) DismissAllAlerts(ctx context.Context, isOrgAdmin bool, callerOrgID, callerTeamID, userID string) (int64, error) {
	var rowsAffected int64
	if isOrgAdmin {
		res, err := r.db.Pool.Exec(ctx, `
			UPDATE budget_alerts
			SET status = 'dismissed', dismissed_by_user_id = $1,
			    dismissed_at = NOW(), updated_at = NOW()
			WHERE status = 'active'`,
			userID,
		)
		if err != nil {
			return 0, fmt.Errorf("dismiss all alerts: %w", err)
		}
		rowsAffected = res.RowsAffected()
	} else {
		res, err := r.db.Pool.Exec(ctx, `
			UPDATE budget_alerts
			SET status = 'dismissed', dismissed_by_user_id = $1,
			    dismissed_at = NOW(), updated_at = NOW()
			FROM budget_definitions b
			WHERE budget_alerts.budget_id = b.id
			  AND budget_alerts.status = 'active'
			  AND b.owner_org_id = $2
			  AND b.owner_team_id = $3`,
			userID, callerOrgID, callerTeamID,
		)
		if err != nil {
			return 0, fmt.Errorf("dismiss all alerts: %w", err)
		}
		rowsAffected = res.RowsAffected()
	}
	return rowsAffected, nil
}

// ResolveAlertsForBudgets resolves all active alerts for the given budget IDs.
// Called after ResetExpiredBudgets to clear alerts from the previous period.
func (r *Repository) ResolveAlertsForBudgets(ctx context.Context, budgetIDs []uuid.UUID) error {
	if len(budgetIDs) == 0 {
		return nil
	}
	_, err := r.db.Pool.Exec(ctx, `
		UPDATE budget_alerts
		SET status = 'resolved', resolved_at = NOW(), updated_at = NOW()
		WHERE budget_id = ANY($1) AND status = 'active'`,
		budgetIDs,
	)
	if err != nil {
		return fmt.Errorf("resolve alerts for budgets: %w", err)
	}
	return nil
}

// GetActiveAlertsForBudget returns the ID and type of all active alerts for one budget.
// Used by the alert worker for auto-resolve checks.
func (r *Repository) GetActiveAlertsForBudget(ctx context.Context, budgetID uuid.UUID) ([]models.BudgetAlert, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, alert_type, period_start
		FROM budget_alerts
		WHERE budget_id = $1 AND status = 'active'`,
		budgetID,
	)
	if err != nil {
		return nil, fmt.Errorf("get active alerts for budget: %w", err)
	}
	defer rows.Close()

	var alerts []models.BudgetAlert
	for rows.Next() {
		var a models.BudgetAlert
		var alertType string
		if err := rows.Scan(&a.ID, &alertType, &a.PeriodStart); err != nil {
			return nil, fmt.Errorf("scan active alert: %w", err)
		}
		a.AlertType = models.AlertType(alertType)
		a.BudgetID = budgetID
		alerts = append(alerts, a)
	}
	return alerts, rows.Err()
}

// ResolveAlert resolves a single active alert by ID.
func (r *Repository) ResolveAlert(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.Pool.Exec(ctx, `
		UPDATE budget_alerts
		SET status = 'resolved', resolved_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND status = 'active'`,
		id,
	)
	if err != nil {
		return fmt.Errorf("resolve alert: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetCurrentPeriodUsage returns the sum of cost_usd from usage_records
// for a budget since periodStart. Used by the alert service for burn rate.
func (r *Repository) GetCurrentPeriodUsage(ctx context.Context, budgetID uuid.UUID, periodStart time.Time) (float64, error) {
	var total float64
	err := r.db.Pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(cost_usd), 0)
		FROM usage_records
		WHERE budget_id = $1 AND created_at >= $2`,
		budgetID, periodStart,
	).Scan(&total)
	if err != nil {
		return total, fmt.Errorf("get current period usage: %w", err)
	}
	return total, nil
}
