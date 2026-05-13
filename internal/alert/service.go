// extras/quota-management/internal/alert/service.go
package alert

import (
	"context"
	"database/sql"
	"time"

	"github.com/agentgateway/quota-management/internal/config"
	"github.com/agentgateway/quota-management/internal/db"
	"github.com/agentgateway/quota-management/internal/models"
	"github.com/rs/zerolog/log"
)

// Service evaluates alert conditions and writes/resolves budget_alerts rows.
type Service struct {
	repo *db.Repository
	cfg  *config.Config
}

// NewService creates a new alert Service.
func NewService(repo *db.Repository, cfg *config.Config) *Service {
	return &Service{repo: repo, cfg: cfg}
}

// evalThresholdWarning returns true when usage meets or exceeds the warning threshold.
func evalThresholdWarning(usageUSD, budgetUSD float64, thresholdPct int) bool {
	if budgetUSD <= 0 {
		return false
	}
	return usageUSD/budgetUSD >= float64(thresholdPct)/100.0
}

// evalBudgetExhausted returns true when usage meets or exceeds the budget amount.
func evalBudgetExhausted(usageUSD, budgetUSD float64) bool {
	if budgetUSD <= 0 {
		return false
	}
	return usageUSD >= budgetUSD
}

// computeAlertForecast returns projected end-of-period spend and days until exhaustion.
// daysUntilExhausted is -1 when the budget will not exhaust before period end.
func computeAlertForecast(
	currentUsageUSD, budgetAmountUSD float64,
	periodStart, periodEnd time.Time,
	totalPeriodUsage float64,
	now time.Time,
) (projectedSpendUSD float64, daysUntilExhausted float64) {
	elapsedHours := now.Sub(periodStart).Hours()
	if elapsedHours <= 0 || totalPeriodUsage <= 0 {
		return currentUsageUSD, -1
	}

	burnRatePerDay := totalPeriodUsage / elapsedHours * 24
	remainingDays := periodEnd.Sub(now).Hours() / 24
	if remainingDays < 0 {
		remainingDays = 0
	}

	projectedSpendUSD = currentUsageUSD + burnRatePerDay*remainingDays
	if projectedSpendUSD <= budgetAmountUSD {
		return projectedSpendUSD, -1
	}

	remainingBudget := budgetAmountUSD - currentUsageUSD
	if remainingBudget <= 0 {
		return projectedSpendUSD, 0
	}
	return projectedSpendUSD, remainingBudget / burnRatePerDay
}

// evalForecastExhaustion returns true when the budget will exhaust within the threshold.
// daysUntilExhausted == -1 means on track.
func evalForecastExhaustion(daysUntilExhausted, thresholdDays float64) bool {
	return daysUntilExhausted >= 0 && daysUntilExhausted < thresholdDays
}

// CheckAndUpdateAlerts evaluates all enabled budgets and writes/resolves alert rows.
func (s *Service) CheckAndUpdateAlerts(ctx context.Context) error {
	budgets, err := s.repo.GetEnabledBudgets(ctx)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, b := range budgets {
		if err := s.checkBudget(ctx, b, now); err != nil {
			log.Warn().Err(err).
				Str("budget_id", b.ID.String()).
				Str("budget_name", b.Name).
				Msg("alert check failed for budget")
		}
	}
	return nil
}

func (s *Service) checkBudget(ctx context.Context, b models.BudgetDefinition, now time.Time) error {
	periodEnd := b.NextPeriodStart()

	// Evaluate threshold and exhaustion conditions.
	thresholdFired := evalThresholdWarning(b.CurrentUsageUSD, b.BudgetAmountUSD, b.WarningThresholdPct)
	exhaustedFired := evalBudgetExhausted(b.CurrentUsageUSD, b.BudgetAmountUSD)

	// Compute burn rate from DB for forecast conditions.
	periodUsage, err := s.repo.GetCurrentPeriodUsage(ctx, b.ID, b.CurrentPeriodStart)
	if err != nil {
		return err
	}
	projected, daysUntil := computeAlertForecast(
		b.CurrentUsageUSD, b.BudgetAmountUSD,
		b.CurrentPeriodStart, periodEnd,
		periodUsage, now,
	)
	overrunFired := projected > b.BudgetAmountUSD
	exhaustionFired := evalForecastExhaustion(daysUntil, s.cfg.ForecastAlertDaysThreshold)

	// Insert alerts for fired conditions (ON CONFLICT DO NOTHING deduplicates).
	if thresholdFired {
		a := &models.BudgetAlert{
			BudgetID:          b.ID,
			AlertType:         models.AlertTypeThresholdWarning,
			TriggeredUsageUSD: b.CurrentUsageUSD,
			BudgetAmountUSD:   b.BudgetAmountUSD,
			PeriodStart:       b.CurrentPeriodStart,
			ThresholdPct:      sql.NullInt32{Int32: int32(b.WarningThresholdPct), Valid: true},
		}
		if err := s.repo.InsertAlertIfNotExists(ctx, a); err != nil {
			log.Warn().Err(err).Msg("insert threshold_warning alert")
		}
	}
	if exhaustedFired {
		a := &models.BudgetAlert{
			BudgetID:          b.ID,
			AlertType:         models.AlertTypeBudgetExhausted,
			TriggeredUsageUSD: b.CurrentUsageUSD,
			BudgetAmountUSD:   b.BudgetAmountUSD,
			PeriodStart:       b.CurrentPeriodStart,
		}
		if err := s.repo.InsertAlertIfNotExists(ctx, a); err != nil {
			log.Warn().Err(err).Msg("insert budget_exhausted alert")
		}
	}
	if overrunFired {
		a := &models.BudgetAlert{
			BudgetID:          b.ID,
			AlertType:         models.AlertTypeForecastOverrun,
			TriggeredUsageUSD: b.CurrentUsageUSD,
			BudgetAmountUSD:   b.BudgetAmountUSD,
			PeriodStart:       b.CurrentPeriodStart,
			ProjectedSpendUSD: sql.NullFloat64{Float64: projected, Valid: true},
		}
		if err := s.repo.InsertAlertIfNotExists(ctx, a); err != nil {
			log.Warn().Err(err).Msg("insert forecast_overrun alert")
		}
	}
	if exhaustionFired {
		a := &models.BudgetAlert{
			BudgetID:           b.ID,
			AlertType:          models.AlertTypeForecastExhaustion,
			TriggeredUsageUSD:  b.CurrentUsageUSD,
			BudgetAmountUSD:    b.BudgetAmountUSD,
			PeriodStart:        b.CurrentPeriodStart,
			ProjectedSpendUSD:  sql.NullFloat64{Float64: projected, Valid: true},
			DaysUntilExhausted: sql.NullFloat64{Float64: daysUntil, Valid: true},
		}
		if err := s.repo.InsertAlertIfNotExists(ctx, a); err != nil {
			log.Warn().Err(err).Msg("insert forecast_exhaustion alert")
		}
	}

	return nil
}
