package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// TimeUnit represents the rate limit time unit.
type TimeUnit string

const (
	TimeUnitMinute TimeUnit = "MINUTE"
	TimeUnitHour   TimeUnit = "HOUR"
	TimeUnitDay    TimeUnit = "DAY"
)

// Enforcement represents the enforcement mode.
type Enforcement string

const (
	EnforcementEnforced   Enforcement = "enforced"
	EnforcementMonitoring Enforcement = "monitoring"
)

// RateLimitAllocation represents a team's rate limit allocation for a model.
type RateLimitAllocation struct {
	ID               uuid.UUID      `json:"id" db:"id"`
	OrgID            string         `json:"org_id" db:"org_id"`
	TeamID           string         `json:"team_id" db:"team_id"`
	ModelPattern     string         `json:"model_pattern" db:"model_pattern"`
	TokenLimit       sql.NullInt64  `json:"token_limit,omitempty" db:"token_limit"`
	TokenUnit        sql.NullString `json:"token_unit,omitempty" db:"token_unit"`
	RequestLimit     sql.NullInt64  `json:"request_limit,omitempty" db:"request_limit"`
	RequestUnit      sql.NullString `json:"request_unit,omitempty" db:"request_unit"`
	BurstPercentage  int            `json:"burst_percentage" db:"burst_percentage"`
	Enforcement      Enforcement    `json:"enforcement" db:"enforcement"`
	Enabled          bool           `json:"enabled" db:"enabled"`
	DisabledByUserID sql.NullString `json:"disabled_by_user_id,omitempty" db:"disabled_by_user_id"`
	DisabledByEmail  sql.NullString `json:"disabled_by_email,omitempty" db:"disabled_by_email"`
	DisabledByIsOrg  bool           `json:"disabled_by_is_org" db:"disabled_by_is_org"`
	DisabledAt       sql.NullTime   `json:"disabled_at,omitempty" db:"disabled_at"`
	ApprovalStatus   ApprovalStatus `json:"approval_status" db:"approval_status"`
	ApprovedBy       sql.NullString `json:"approved_by,omitempty" db:"approved_by"`
	ApprovedAt       sql.NullTime   `json:"approved_at,omitempty" db:"approved_at"`
	CreatedByUserID  sql.NullString `json:"created_by_user_id,omitempty" db:"created_by_user_id"`
	CreatedByEmail   sql.NullString `json:"created_by_email,omitempty" db:"created_by_email"`
	Description      sql.NullString `json:"description,omitempty" db:"description"`
	RejectionCount   int            `json:"rejection_count" db:"rejection_count"`
	Version          int64          `json:"version" db:"version"`
	CreatedAt        time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at" db:"updated_at"`
}

// RateLimitApproval represents an approval action on a rate limit allocation.
type RateLimitApproval struct {
	ID            uuid.UUID      `json:"id" db:"id"`
	AllocationID  uuid.UUID      `json:"allocation_id" db:"allocation_id"`
	AttemptNumber int            `json:"attempt_number" db:"attempt_number"`
	Action        string         `json:"action" db:"action"`
	ActorUserID   sql.NullString `json:"actor_user_id,omitempty" db:"actor_user_id"`
	ActorEmail    sql.NullString `json:"actor_email,omitempty" db:"actor_email"`
	Reason        sql.NullString `json:"reason,omitempty" db:"reason"`
	CreatedAt     time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at" db:"updated_at"`
}

// RateLimitApprovalWithAllocation combines approval with allocation details.
type RateLimitApprovalWithAllocation struct {
	RateLimitApproval
	TeamID       string `json:"team_id"`
	ModelPattern string `json:"model_pattern"`
	OrgID        string `json:"org_id"`
}

// HasTokenLimit returns true if token limit is set.
func (r *RateLimitAllocation) HasTokenLimit() bool {
	return r.TokenLimit.Valid && r.TokenLimit.Int64 > 0
}

// HasRequestLimit returns true if request limit is set.
func (r *RateLimitAllocation) HasRequestLimit() bool {
	return r.RequestLimit.Valid && r.RequestLimit.Int64 > 0
}

// TokenUnitValue returns the token unit as TimeUnit or empty.
func (r *RateLimitAllocation) TokenUnitValue() TimeUnit {
	if r.TokenUnit.Valid {
		return TimeUnit(r.TokenUnit.String)
	}
	return ""
}

// RequestUnitValue returns the request unit as TimeUnit or empty.
func (r *RateLimitAllocation) RequestUnitValue() TimeUnit {
	if r.RequestUnit.Valid {
		return TimeUnit(r.RequestUnit.String)
	}
	return ""
}
