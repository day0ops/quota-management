package api

import (
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/agentgateway/quota-management/internal/auth"
	"github.com/agentgateway/quota-management/internal/db"
	"github.com/agentgateway/quota-management/internal/models"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// forecastCacheEntry holds a cached forecast result.
type forecastCacheEntry struct {
	result    *ForecastResult
	expiresAt time.Time
}

// burnRateSnapshot freezes the burn rate for a given usage level.
// It is only recomputed when currentUsageUSD changes (new spend).
type burnRateSnapshot struct {
	burnRatePerDay float64
	forUsage       float64
}

var (
	forecastCache    sync.Map
	forecastCacheTTL = 5 * time.Minute
	burnRateCache    sync.Map // budgetID → burnRateSnapshot
)

// TimeSeriesPoint represents one point in the forecast time series.
type TimeSeriesPoint struct {
	Date         string   `json:"date"`
	ActualUSD    *float64 `json:"actual_usd"`
	ProjectedUSD *float64 `json:"projected_usd"`
}

// ForecastResult is the response shape for a single budget forecast.
type ForecastResult struct {
	BudgetID           string            `json:"budget_id"`
	PeriodStart        time.Time         `json:"period_start"`
	PeriodEnd          time.Time         `json:"period_end"`
	BudgetAmountUSD    float64           `json:"budget_amount_usd"`
	CurrentUsageUSD    float64           `json:"current_usage_usd"`
	ProjectedSpendUSD  float64           `json:"projected_spend_usd"`
	DaysUntilExhausted float64           `json:"days_until_exhausted"` // -1 = on track this period
	BurnRateUSDPerDay  float64           `json:"burn_rate_usd_per_day"`
	Confidence         float64           `json:"confidence"` // 0.0-1.0
	TimeSeries         []TimeSeriesPoint `json:"time_series"`
	PeriodBoundaries   []string          `json:"period_boundaries,omitempty"`
}

const (
	dateFormatDaily = "2006-01-02"
	dateFormatSub   = "2006-01-02T15:04:05Z" // hourly and minute resolution
)

// timeSeriesStep returns the step duration and date format for the time series based on period duration.
// step == 0 means use AddDate(0,0,1) for calendar-correct daily stepping.
func timeSeriesStep(periodDur time.Duration) (step time.Duration, dateFormat string) {
	switch {
	case periodDur < time.Hour:
		return time.Minute, dateFormatSub
	case periodDur < 24*time.Hour:
		return time.Hour, dateFormatSub
	default:
		return 0, dateFormatDaily
	}
}

// stableBurnRate returns the burn rate for the given budget, frozen per usage level.
// It only recomputes when currentUsageUSD changes (i.e. new spend occurred).
// Uses up to a 7-day rolling window across periods for stable daily averages.
func stableBurnRate(id uuid.UUID, currentUsageUSD float64, elapsedDays float64, dailyUsage []db.DailyUsage, now time.Time) float64 {
	// Return frozen rate if usage hasn't changed.
	if snap, ok := burnRateCache.Load(id); ok {
		s := snap.(burnRateSnapshot)
		if s.forUsage == currentUsageUSD {
			return s.burnRatePerDay
		}
	}

	// Compute fresh burn rate using up to 7 completed days across any period.
	todayStr := now.UTC().Format("2006-01-02")
	windowStart := now.AddDate(0, 0, -7)

	var completedTotal float64
	var completedCount int
	for _, du := range dailyUsage {
		dateKey := du.Day.UTC().Format("2006-01-02")
		if dateKey != todayStr && !du.Day.Before(windowStart) {
			completedTotal += du.TotalUSD
			completedCount++
		}
	}

	var rate float64
	if completedCount > 0 {
		// Stable: average of recent completed days.
		rate = completedTotal / float64(completedCount)
	} else if elapsedDays > 0 {
		// No completed days: extrapolate from today's partial spend.
		// Minimum 1-hour window prevents extreme values at session start.
		const minElapsedDays = 1.0 / 24.0
		rate = currentUsageUSD / math.Max(elapsedDays, minElapsedDays)
	}

	// Freeze this rate until usage changes.
	burnRateCache.Store(id, burnRateSnapshot{burnRatePerDay: rate, forUsage: currentUsageUSD})
	return rate
}

// computeForecast calculates forecast data for a single budget.
// burnRatePerDay is pre-computed and stable (see stableBurnRate).
// historyStart is the beginning of the multi-period history window for the time series.
// periodBoundaries are timestamp strings where historical periods reset.
// step controls time series granularity: 0 = calendar daily (AddDate), >0 = fixed duration.
// dateFormat is the format string used for TimeSeries.Date and boundary keys.
// now is injectable for testability.
func computeForecast(
	budgetAmountUSD float64,
	currentUsageUSD float64,
	periodStart time.Time,
	periodEnd time.Time,
	usageData []db.DailyUsage,
	burnRatePerDay float64,
	now time.Time,
	historyStart time.Time,
	periodBoundaries []string,
	step time.Duration,
	dateFormat string,
) ForecastResult {
	totalPeriodDays := periodEnd.Sub(periodStart).Hours() / 24
	elapsedDays := now.Sub(periodStart).Hours() / 24
	remainingDays := periodEnd.Sub(now).Hours() / 24

	if elapsedDays < 0 {
		elapsedDays = 0
	}
	if remainingDays < 0 {
		remainingDays = 0
	}

	projectedSpend := currentUsageUSD
	if burnRatePerDay > 0 {
		projectedSpend = currentUsageUSD + (burnRatePerDay * remainingDays)
	}

	// Days until exhausted — capped at remaining period days.
	// If the budget won't exhaust before the period resets, return -1 (on track).
	daysUntilExhausted := -1.0
	remaining := budgetAmountUSD - currentUsageUSD
	if burnRatePerDay > 0 && remaining > 0 {
		rawDays := remaining / burnRatePerDay
		if rawDays <= remainingDays {
			daysUntilExhausted = rawDays
		}
		// rawDays > remainingDays: budget survives this period → -1
	}

	confidence := 0.0
	if totalPeriodDays > 0 {
		confidence = math.Min(elapsedDays/totalPeriodDays, 1.0)
	}

	// Build time series indexed by formatted timestamp.
	actualByDate := make(map[string]float64)
	for _, du := range usageData {
		key := du.Day.UTC().Format(dateFormat)
		actualByDate[key] = du.TotalUSD
	}

	// Projected spend per step (scaled from daily burn rate).
	stepFraction := 1.0 // default: one day per step
	if step > 0 {
		stepFraction = step.Hours() / 24.0
	}

	var timeSeries []TimeSeriesPoint
	cursor := historyStart
	if step > 0 {
		cursor = historyStart.Truncate(step)
	}
	for cursor.Before(periodEnd) {
		dateKey := cursor.UTC().Format(dateFormat)
		pt := TimeSeriesPoint{Date: dateKey}

		if !cursor.After(now) {
			if actual, ok := actualByDate[dateKey]; ok {
				v := actual
				pt.ActualUSD = &v
			}
		} else {
			v := burnRatePerDay * stepFraction
			pt.ProjectedUSD = &v
		}
		timeSeries = append(timeSeries, pt)
		if step > 0 {
			cursor = cursor.Add(step)
		} else {
			cursor = cursor.AddDate(0, 0, 1)
		}
	}

	return ForecastResult{
		PeriodStart:        periodStart,
		PeriodEnd:          periodEnd,
		BudgetAmountUSD:    budgetAmountUSD,
		CurrentUsageUSD:    currentUsageUSD,
		ProjectedSpendUSD:  math.Round(projectedSpend*1000000) / 1000000,
		DaysUntilExhausted: math.Round(daysUntilExhausted*100) / 100,
		BurnRateUSDPerDay:  math.Round(burnRatePerDay*1000000) / 1000000,
		Confidence:         math.Round(confidence*100) / 100,
		TimeSeries:         timeSeries,
		PeriodBoundaries:   periodBoundaries,
	}
}

// GetBudgetForecast returns a forecast for a single budget.
func (h *Handler) GetBudgetForecast(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid budget ID")
		return
	}

	budget, err := h.repo.GetBudgetByID(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "budget not found")
		return
	}

	// RBAC: user-type budgets check CanViewUserBudget
	identity := auth.GetIdentity(r.Context())
	if identity != nil && budget.EntityType == models.EntityTypeUser {
		ownerUserID := ""
		if budget.OwnerUserID.Valid {
			ownerUserID = budget.OwnerUserID.String
		}
		ownerTeamID := ""
		if budget.OwnerTeamID.Valid {
			ownerTeamID = budget.OwnerTeamID.String
		}
		if !auth.CanViewUserBudget(identity, ownerUserID, ownerTeamID) {
			writeError(w, http.StatusForbidden, "access denied")
			return
		}
	}

	// Check cache — skip if usage changed since caching.
	if entry, ok := forecastCache.Load(id); ok {
		cached := entry.(*forecastCacheEntry)
		if time.Now().Before(cached.expiresAt) && cached.result.CurrentUsageUSD == budget.CurrentUsageUSD {
			writeJSON(w, http.StatusOK, cached.result)
			return
		}
	}

	periodEnd := budget.NextPeriodStart()
	step, dateFormat := timeSeriesStep(periodEnd.Sub(budget.CurrentPeriodStart))

	var usageData []db.DailyUsage
	var usageErr error
	switch {
	case step == time.Minute:
		usageData, usageErr = h.repo.GetMinuteUsageForPeriod(r.Context(), id, budget.CurrentPeriodStart)
	case step == time.Hour:
		usageData, usageErr = h.repo.GetHourlyUsageForPeriod(r.Context(), id, budget.CurrentPeriodStart)
	default:
		usageData, usageErr = h.repo.GetDailyUsageForPeriod(r.Context(), id, budget.CurrentPeriodStart)
	}
	if usageErr != nil {
		log.Error().Err(usageErr).Str("budget_id", id.String()).Msg("failed to get usage for forecast")
		writeError(w, http.StatusInternalServerError, "failed to compute forecast")
		return
	}

	now := time.Now()
	var burnRate float64
	if step > 0 {
		elapsedHours := now.Sub(budget.CurrentPeriodStart).Hours()
		const minHours = 1.0 / 60.0
		if elapsedHours < minHours {
			elapsedHours = minHours
		}
		burnRate = (budget.CurrentUsageUSD / elapsedHours) * 24
	} else {
		elapsedDays := now.Sub(budget.CurrentPeriodStart).Hours() / 24
		burnRate = stableBurnRate(id, budget.CurrentUsageUSD, elapsedDays, usageData, now)
	}

	result := computeForecast(
		budget.BudgetAmountUSD,
		budget.CurrentUsageUSD,
		budget.CurrentPeriodStart,
		periodEnd,
		usageData,
		burnRate,
		now,
		budget.CurrentPeriodStart,
		nil,
		step,
		dateFormat,
	)
	result.BudgetID = id.String()

	forecastCache.Store(id, &forecastCacheEntry{
		result:    &result,
		expiresAt: time.Now().Add(forecastCacheTTL),
	})

	writeJSON(w, http.StatusOK, result)
}

// ListBudgetForecasts returns forecasts for all budgets visible to the caller.
func (h *Handler) ListBudgetForecasts(w http.ResponseWriter, r *http.Request) {
	var filter db.BudgetListFilter
	identity := auth.GetIdentity(r.Context())
	if identity != nil {
		filter.OrgID = identity.OrgID
		filter.TeamID = identity.TeamID
		filter.IsOrg = identity.IsOrg
		filter.IsTeamAdmin = identity.IsTeamAdmin
		if !identity.IsOrg && !identity.IsTeamAdmin {
			filter.UserID = identity.UserID
		}
	}

	budgets, _, err := h.repo.ListBudgetsPaginated(r.Context(), filter, 0, 1000)
	if err != nil {
		log.Error().Err(err).Msg("failed to list budgets for forecast")
		writeError(w, http.StatusInternalServerError, "failed to list budgets")
		return
	}

	results := make([]ForecastResult, 0, len(budgets))
	for _, budget := range budgets {
		budgetID := budget.ID

		// Check cache — skip if usage changed since caching.
		if entry, ok := forecastCache.Load(budgetID); ok {
			cached := entry.(*forecastCacheEntry)
			if time.Now().Before(cached.expiresAt) && cached.result.CurrentUsageUSD == budget.CurrentUsageUSD {
				results = append(results, *cached.result)
				continue
			}
		}

		listPeriodEnd := budget.NextPeriodStart()
		listStep, listDateFormat := timeSeriesStep(listPeriodEnd.Sub(budget.CurrentPeriodStart))

		var listUsage []db.DailyUsage
		var listUsageErr error
		switch {
		case listStep == time.Minute:
			listUsage, listUsageErr = h.repo.GetMinuteUsageForPeriod(r.Context(), budgetID, budget.CurrentPeriodStart)
		case listStep == time.Hour:
			listUsage, listUsageErr = h.repo.GetHourlyUsageForPeriod(r.Context(), budgetID, budget.CurrentPeriodStart)
		default:
			listUsage, listUsageErr = h.repo.GetDailyUsageForPeriod(r.Context(), budgetID, budget.CurrentPeriodStart)
		}
		if listUsageErr != nil {
			log.Warn().Err(listUsageErr).Str("budget_id", budgetID.String()).Msg("skipping budget in forecast list")
			continue
		}

		listNow := time.Now()
		var listBurnRate float64
		if listStep > 0 {
			elapsedHours := listNow.Sub(budget.CurrentPeriodStart).Hours()
			const minHours = 1.0 / 60.0
			if elapsedHours < minHours {
				elapsedHours = minHours
			}
			listBurnRate = (budget.CurrentUsageUSD / elapsedHours) * 24
		} else {
			listElapsed := listNow.Sub(budget.CurrentPeriodStart).Hours() / 24
			listBurnRate = stableBurnRate(budgetID, budget.CurrentUsageUSD, listElapsed, listUsage, listNow)
		}

		result := computeForecast(
			budget.BudgetAmountUSD,
			budget.CurrentUsageUSD,
			budget.CurrentPeriodStart,
			listPeriodEnd,
			listUsage,
			listBurnRate,
			listNow,
			budget.CurrentPeriodStart,
			nil,
			listStep,
			listDateFormat,
		)
		result.BudgetID = budgetID.String()

		forecastCache.Store(budgetID, &forecastCacheEntry{
			result:    &result,
			expiresAt: time.Now().Add(forecastCacheTTL),
		})

		results = append(results, result)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":  results,
		"count": len(results),
	})
}
