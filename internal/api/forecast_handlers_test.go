package api

import (
	"testing"
	"time"

	"github.com/agentgateway/quota-management/internal/db"
	"github.com/stretchr/testify/assert"
)

func TestComputeForecast_BasicProjection(t *testing.T) {
	now := time.Date(2026, 4, 15, 12, 0, 0, 0, time.UTC)
	periodStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	dailyUsage := []db.DailyUsage{
		{Day: periodStart, TotalUSD: 3.0},
		{Day: periodStart.AddDate(0, 0, 1), TotalUSD: 3.0},
		{Day: periodStart.AddDate(0, 0, 13), TotalUSD: 3.0},
	}
	burnRate := 42.0 / 14.5 // ≈ 2.9/day

	result := computeForecast(100.0, 42.0, periodStart, periodEnd, dailyUsage, burnRate, now, periodStart, nil, 0, dateFormatDaily)

	assert.InDelta(t, burnRate, result.BurnRateUSDPerDay, 0.01)
	assert.Greater(t, result.ProjectedSpendUSD, 42.0)
	assert.Greater(t, result.DaysUntilExhausted, 0.0)
	assert.InDelta(t, 14.5/30.0, result.Confidence, 0.1)
}

func TestComputeForecast_ZeroUsage(t *testing.T) {
	now := time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC)
	periodStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)

	result := computeForecast(100.0, 0.0, periodStart, periodEnd, nil, 0.0, now, periodStart, nil, 0, dateFormatDaily)

	assert.Equal(t, 0.0, result.BurnRateUSDPerDay)
	assert.Equal(t, 0.0, result.ProjectedSpendUSD)
	assert.Equal(t, -1.0, result.DaysUntilExhausted)
}

func TestComputeForecast_TimeSeriesHasActualAndProjected(t *testing.T) {
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	periodStart := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := time.Date(2026, 4, 6, 0, 0, 0, 0, time.UTC)

	dailyUsage := []db.DailyUsage{
		{Day: time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), TotalUSD: 2.0},
		{Day: time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC), TotalUSD: 4.0},
	}
	burnRate := 3.0 // (2+4)/2 = 3/day

	result := computeForecast(50.0, 6.0, periodStart, periodEnd, dailyUsage, burnRate, now, periodStart, nil, 0, dateFormatDaily)

	actualsCount := 0
	projectedCount := 0
	for _, pt := range result.TimeSeries {
		if pt.ActualUSD != nil {
			actualsCount++
		}
		if pt.ProjectedUSD != nil {
			projectedCount++
		}
	}
	assert.Equal(t, 2, actualsCount)
	assert.Greater(t, projectedCount, 0)
}
