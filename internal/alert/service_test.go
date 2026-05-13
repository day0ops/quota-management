// extras/quota-management/internal/alert/service_test.go
package alert

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEvalThresholdWarning(t *testing.T) {
	assert.True(t, evalThresholdWarning(80.0, 100.0, 80))  // exactly at threshold
	assert.True(t, evalThresholdWarning(85.0, 100.0, 80))  // above threshold
	assert.False(t, evalThresholdWarning(79.9, 100.0, 80)) // below threshold
	assert.False(t, evalThresholdWarning(0.0, 100.0, 80))  // zero usage
}

func TestEvalThresholdWarning_ZeroBudget(t *testing.T) {
	assert.False(t, evalThresholdWarning(0.0, 0.0, 80)) // division by zero guard
}

func TestEvalBudgetExhausted(t *testing.T) {
	assert.True(t, evalBudgetExhausted(100.0, 100.0))  // exactly at limit
	assert.True(t, evalBudgetExhausted(100.01, 100.0)) // over limit
	assert.False(t, evalBudgetExhausted(99.9, 100.0))  // under limit
	assert.False(t, evalBudgetExhausted(0.0, 100.0))   // no usage
}

func TestEvalBudgetExhausted_ZeroBudget(t *testing.T) {
	assert.False(t, evalBudgetExhausted(0.0, 0.0)) // zero budget guard
}

func TestComputeAlertForecast_OnTrack(t *testing.T) {
	// 14.5 days elapsed, spent $20 at ~$1.38/day, projected ~$42 < $100
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	projected, days := computeAlertForecast(20.0, 100.0, start, end, 20.0, now)

	assert.Less(t, projected, 100.0)
	assert.Equal(t, -1.0, days)
}

func TestComputeAlertForecast_WillOverrun(t *testing.T) {
	// 14.5 days elapsed, spent $80 at ~$5.5/day, projected well over $100
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	projected, days := computeAlertForecast(80.0, 100.0, start, end, 80.0, now)

	assert.Greater(t, projected, 100.0)
	assert.GreaterOrEqual(t, days, 0.0)
	assert.Less(t, days, 5.0)
}

func TestComputeAlertForecast_NoBurnRate(t *testing.T) {
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	start := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	projected, days := computeAlertForecast(0.0, 100.0, start, end, 0.0, now)

	assert.Equal(t, 0.0, projected)
	assert.Equal(t, -1.0, days)
}

func TestEvalForecastExhaustion(t *testing.T) {
	assert.True(t, evalForecastExhaustion(2.9, 3.0))   // under threshold
	assert.True(t, evalForecastExhaustion(0.0, 3.0))   // already exhausted
	assert.False(t, evalForecastExhaustion(-1.0, 3.0)) // on track (won't exhaust)
	assert.False(t, evalForecastExhaustion(3.1, 3.0))  // above threshold
}
