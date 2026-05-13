package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// BudgetPeriod represents the budget period type.
type BudgetPeriod string

const (
	PeriodHourly  BudgetPeriod = "hourly"
	PeriodDaily   BudgetPeriod = "daily"
	PeriodWeekly  BudgetPeriod = "weekly"
	PeriodMonthly BudgetPeriod = "monthly"
	PeriodCustom  BudgetPeriod = "custom"
)

// EntityType represents the entity type for budget.
type EntityType string

const (
	EntityTypeOrg  EntityType = "org"  // Budget per organization (from jwt.claims.org)
	EntityTypeTeam EntityType = "team" // Budget per team (from jwt.claims.team)
	EntityTypeUser EntityType = "user" // Budget per user (from jwt.sub)
)

// ModelCost represents the cost configuration for an LLM model.
type ModelCost struct {
	ID                    uuid.UUID       `json:"id" db:"id"`
	ModelID               string          `json:"model_id" db:"model_id"`
	Provider              string          `json:"provider" db:"provider"`
	InputCostPerMillion   float64         `json:"input_cost_per_million" db:"input_cost_per_million"`
	OutputCostPerMillion  float64         `json:"output_cost_per_million" db:"output_cost_per_million"`
	CacheReadCostMillion  sql.NullFloat64 `json:"cache_read_cost_million,omitempty" db:"cache_read_cost_million"`
	CacheWriteCostMillion sql.NullFloat64 `json:"cache_write_cost_million,omitempty" db:"cache_write_cost_million"`
	ModelPattern          sql.NullString  `json:"model_pattern,omitempty" db:"model_pattern"`
	EffectiveDate         time.Time       `json:"effective_date" db:"effective_date"`
	CreatedByUserID       sql.NullString  `json:"created_by_user_id,omitempty" db:"created_by_user_id"`
	CreatedByEmail        sql.NullString  `json:"created_by_email,omitempty" db:"created_by_email"`
	CreatedAt             time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at" db:"updated_at"`
}

// BudgetDefinition represents a budget configuration.
type BudgetDefinition struct {
	ID                  uuid.UUID      `json:"id" db:"id"`
	EntityType          EntityType     `json:"entity_type" db:"entity_type"`
	Name                string         `json:"name" db:"name"`
	MatchExpression     string         `json:"match_expression" db:"match_expression"`
	BudgetAmountUSD     float64        `json:"budget_amount_usd" db:"budget_amount_usd"`
	Period              BudgetPeriod   `json:"period" db:"period"`
	CustomPeriodSeconds sql.NullInt32  `json:"custom_period_seconds,omitempty" db:"custom_period_seconds"`
	WarningThresholdPct int            `json:"warning_threshold_pct" db:"warning_threshold_pct"`
	ParentID            *uuid.UUID     `json:"parent_id,omitempty" db:"parent_id"`
	Isolated            bool           `json:"isolated" db:"isolated"`
	AllowFallback       bool           `json:"allow_fallback" db:"allow_fallback"`
	Enabled             bool           `json:"enabled" db:"enabled"`
	DisabledByUserID    sql.NullString `json:"disabled_by_user_id,omitempty" db:"disabled_by_user_id"`
	DisabledByEmail     sql.NullString `json:"disabled_by_email,omitempty" db:"disabled_by_email"`
	DisabledByIsOrg     bool           `json:"disabled_by_is_org" db:"disabled_by_is_org"`
	DisabledAt          sql.NullTime   `json:"disabled_at,omitempty" db:"disabled_at"`
	CurrentPeriodStart  time.Time      `json:"current_period_start" db:"current_period_start"`
	CurrentUsageUSD     float64        `json:"current_usage_usd" db:"current_usage_usd"`
	PendingUsageUSD     float64        `json:"pending_usage_usd" db:"pending_usage_usd"`
	Description         sql.NullString `json:"description,omitempty" db:"description"`
	OwnerOrgID          sql.NullString `json:"owner_org_id,omitempty" db:"owner_org_id"`
	OwnerTeamID         sql.NullString `json:"owner_team_id,omitempty" db:"owner_team_id"`
	OwnerUserID         sql.NullString `json:"owner_user_id,omitempty" db:"owner_user_id"`
	ApprovalStatus      ApprovalStatus `json:"approval_status" db:"approval_status"`
	CreatedByUserID     sql.NullString `json:"created_by_user_id,omitempty" db:"created_by_user_id"`
	CreatedByEmail      sql.NullString `json:"created_by_email,omitempty" db:"created_by_email"`
	RejectionCount      int            `json:"rejection_count" db:"rejection_count"`
	Version             int64          `json:"version" db:"version"`
	CreatedAt           time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at" db:"updated_at"`
}

// UsageRecord represents a single usage record.
type UsageRecord struct {
	ID            uuid.UUID `json:"id" db:"id"`
	BudgetID      uuid.UUID `json:"budget_id" db:"budget_id"`
	RequestID     string    `json:"request_id" db:"request_id"`
	ModelID       string    `json:"model_id" db:"model_id"`
	InputTokens   int64     `json:"input_tokens" db:"input_tokens"`
	OutputTokens  int64     `json:"output_tokens" db:"output_tokens"`
	CostUSD       float64   `json:"cost_usd" db:"cost_usd"`
	ParentCharged bool      `json:"parent_charged" db:"parent_charged"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// RequestReservation represents a pending budget reservation.
type RequestReservation struct {
	ID               uuid.UUID `json:"id" db:"id"`
	BudgetID         uuid.UUID `json:"budget_id" db:"budget_id"`
	RequestID        string    `json:"request_id" db:"request_id"`
	EstimatedCostUSD float64   `json:"estimated_cost_usd" db:"estimated_cost_usd"`
	ExpiresAt        time.Time `json:"expires_at" db:"expires_at"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
}

// BudgetWithRemaining extends BudgetDefinition with calculated remaining budget.
type BudgetWithRemaining struct {
	BudgetDefinition
	RemainingUSD float64 `json:"remaining_usd"`
}

// CalculateRemaining calculates the remaining budget.
func (b *BudgetDefinition) CalculateRemaining() float64 {
	return b.BudgetAmountUSD - b.CurrentUsageUSD - b.PendingUsageUSD
}

// NextPeriodStart calculates the next period start time.
func (b *BudgetDefinition) NextPeriodStart() time.Time {
	switch b.Period {
	case PeriodHourly:
		return b.CurrentPeriodStart.Add(time.Hour)
	case PeriodDaily:
		return b.CurrentPeriodStart.AddDate(0, 0, 1)
	case PeriodWeekly:
		return b.CurrentPeriodStart.AddDate(0, 0, 7)
	case PeriodMonthly:
		return b.CurrentPeriodStart.AddDate(0, 1, 0)
	case PeriodCustom:
		if b.CustomPeriodSeconds.Valid {
			return b.CurrentPeriodStart.Add(time.Duration(b.CustomPeriodSeconds.Int32) * time.Second)
		}
	}
	// Default to daily
	return b.CurrentPeriodStart.AddDate(0, 0, 1)
}

// ShouldReset checks if the budget period should be reset.
func (b *BudgetDefinition) ShouldReset() bool {
	return time.Now().After(b.NextPeriodStart())
}
