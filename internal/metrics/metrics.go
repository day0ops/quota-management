package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// Request-level metrics (one increment per request)
	BudgetRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "budget_management_requests_total",
			Help: "Total number of requests processed by budget limiter",
		},
		[]string{"result"}, // "allowed" or "denied"
	)

	// Budget check metrics (per-budget granularity)
	BudgetChecksTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "budget_management_checks_total",
			Help: "Total number of budget checks performed (per budget)",
		},
		[]string{"entity_type", "name", "result"},
	)

	BudgetCheckDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "budget_management_check_duration_seconds",
			Help:    "Duration of budget check operations",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
		},
		[]string{"entity_type"},
	)

	// Cost tracking metrics (per entity)
	CostChargedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "budget_management_cost_charged_usd_total",
			Help: "Total cost charged in USD per entity",
		},
		[]string{"entity_type", "name", "model"},
	)

	TokensProcessedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "budget_management_tokens_total",
			Help: "Total tokens processed per entity",
		},
		[]string{"entity_type", "name", "model", "direction"},
	)

	// Budget usage metrics (gauges for current state)
	BudgetUsageUSD = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "budget_management_usage_usd",
			Help: "Current budget usage in USD",
		},
		[]string{"entity_type", "name", "period"},
	)

	BudgetRemainingUSD = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "budget_management_remaining_usd",
			Help: "Remaining budget in USD",
		},
		[]string{"entity_type", "name", "period"},
	)

	BudgetUtilizationPct = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "budget_management_utilization_pct",
			Help: "Budget utilization percentage",
		},
		[]string{"entity_type", "name", "period"},
	)

	// Rate limiting metrics
	RequestsRateLimitedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "budget_management_requests_rate_limited_total",
			Help: "Total number of requests rate limited due to budget exhaustion",
		},
		[]string{"entity_type", "name"},
	)

	// Budget fallback metrics
	BudgetFallbacksTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "budget_management_fallbacks_total",
			Help: "Total number of requests that fell back to parent budget",
		},
		[]string{"child_entity_type", "child_name", "parent_entity_type", "parent_name"},
	)

	// Reservation metrics
	ActiveReservations = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "budget_management_active_reservations",
			Help: "Number of active budget reservations",
		},
	)

	ReservationsCreatedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "budget_management_reservations_created_total",
			Help: "Total number of reservations created",
		},
	)

	ReservationsExpiredTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "budget_management_reservations_expired_total",
			Help: "Total number of reservations that expired",
		},
	)

	// Period reset metrics
	BudgetPeriodsResetTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "budget_management_periods_reset_total",
			Help: "Total number of budget periods reset",
		},
		[]string{"period"},
	)

	// Quota status metrics (budget + rate limit approval tracking)
	QuotaStatusTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "quota_status_total",
			Help: "Total count of quota items by org, type, and status",
		},
		[]string{"org", "type", "status"},
	)

	// ext-proc metrics
	ExtProcRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "budget_management_extproc_requests_total",
			Help: "Total number of ext-proc requests processed",
		},
		[]string{"phase", "status"},
	)

	ExtProcDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "budget_management_extproc_duration_seconds",
			Help:    "Duration of ext-proc request processing",
			Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1},
		},
		[]string{"phase"},
	)
)

// RecordBudgetRequest records a request-level result (once per request)
func RecordBudgetRequest(allowed bool) {
	result := "allowed"
	if !allowed {
		result = "denied"
	}
	BudgetRequestsTotal.WithLabelValues(result).Inc()
}

// RecordBudgetCheck records a budget check result (per-budget granularity)
func RecordBudgetCheck(entityType, name string, allowed bool, duration time.Duration) {
	result := "allowed"
	if !allowed {
		result = "denied"
	}
	BudgetChecksTotal.WithLabelValues(entityType, name, result).Inc()
	BudgetCheckDuration.WithLabelValues(entityType).Observe(duration.Seconds())
}

// RecordCostCharged records cost charged to a specific entity
func RecordCostCharged(entityType, name, model string, costUSD float64) {
	CostChargedTotal.WithLabelValues(entityType, name, model).Add(costUSD)
}

// RecordTokens records tokens processed for a specific entity
func RecordTokens(entityType, name, model string, inputTokens, outputTokens int64) {
	TokensProcessedTotal.WithLabelValues(entityType, name, model, "input").Add(float64(inputTokens))
	TokensProcessedTotal.WithLabelValues(entityType, name, model, "output").Add(float64(outputTokens))
}

// UpdateBudgetUsage updates the current budget usage metrics
func UpdateBudgetUsage(entityType, name, period string, usageUSD, remainingUSD, budgetAmountUSD float64) {
	BudgetUsageUSD.WithLabelValues(entityType, name, period).Set(usageUSD)
	BudgetRemainingUSD.WithLabelValues(entityType, name, period).Set(remainingUSD)
	if budgetAmountUSD > 0 {
		utilization := (usageUSD / budgetAmountUSD) * 100
		BudgetUtilizationPct.WithLabelValues(entityType, name, period).Set(utilization)
	}
}

// RecordRateLimited records a rate limited request
func RecordRateLimited(entityType, name string) {
	RequestsRateLimitedTotal.WithLabelValues(entityType, name).Inc()
}

// RecordBudgetFallback records when a request falls back to a parent budget
func RecordBudgetFallback(childEntityType, childName, parentEntityType, parentName string) {
	BudgetFallbacksTotal.WithLabelValues(childEntityType, childName, parentEntityType, parentName).Inc()
}

// RecordExtProc records ext-proc processing
func RecordExtProc(phase, status string, duration time.Duration) {
	ExtProcRequestsTotal.WithLabelValues(phase, status).Inc()
	ExtProcDuration.WithLabelValues(phase).Observe(duration.Seconds())
}

// RecordQuotaStatus records a quota status change (budget or rate limit).
func RecordQuotaStatus(org, quotaType, status string) {
	QuotaStatusTotal.WithLabelValues(org, quotaType, status).Inc()
}

// DeleteBudgetMetrics removes all gauge metrics for a deleted budget
func DeleteBudgetMetrics(entityType, name, period string) {
	BudgetUsageUSD.DeleteLabelValues(entityType, name, period)
	BudgetRemainingUSD.DeleteLabelValues(entityType, name, period)
	BudgetUtilizationPct.DeleteLabelValues(entityType, name, period)
}

// Handler returns the prometheus HTTP handler
func Handler() http.Handler {
	return promhttp.Handler()
}
