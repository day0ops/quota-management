package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// AlertType represents the type of budget alert.
type AlertType string

const (
	AlertTypeThresholdWarning   AlertType = "threshold_warning"
	AlertTypeBudgetExhausted    AlertType = "budget_exhausted"
	AlertTypeForecastOverrun    AlertType = "forecast_overrun"
	AlertTypeForecastExhaustion AlertType = "forecast_exhaustion"
)

// AlertStatus represents the lifecycle state of an alert.
type AlertStatus string

const (
	AlertStatusActive    AlertStatus = "active"
	AlertStatusResolved  AlertStatus = "resolved"
	AlertStatusDismissed AlertStatus = "dismissed"
)

// BudgetAlert represents a single alert event for a budget.
type BudgetAlert struct {
	ID                 uuid.UUID       `json:"id"`
	BudgetID           uuid.UUID       `json:"budget_id"`
	AlertType          AlertType       `json:"alert_type"`
	Status             AlertStatus     `json:"status"`
	TriggeredUsageUSD  float64         `json:"triggered_usage_usd"`
	BudgetAmountUSD    float64         `json:"budget_amount_usd"`
	ThresholdPct       sql.NullInt32   `json:"threshold_pct"`
	ProjectedSpendUSD  sql.NullFloat64 `json:"projected_spend_usd"`
	DaysUntilExhausted sql.NullFloat64 `json:"days_until_exhausted"`
	DismissedByUserID  sql.NullString  `json:"dismissed_by_user_id"`
	DismissedAt        sql.NullTime    `json:"dismissed_at"`
	ResolvedAt         sql.NullTime    `json:"resolved_at"`
	PeriodStart        time.Time       `json:"period_start"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
	// Joined from budget_definitions — not stored in budget_alerts
	BudgetName string `json:"budget_name"`
	EntityType string `json:"entity_type"`
}
