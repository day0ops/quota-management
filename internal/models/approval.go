package models

import (
	"time"

	"github.com/google/uuid"
)

// ApprovalStatus represents the approval state of a budget.
type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"
	ApprovalStatusClosed   ApprovalStatus = "closed"
)

// BudgetApproval represents a single approval action on a budget.
type BudgetApproval struct {
	ID            uuid.UUID `json:"id" db:"id"`
	BudgetID      uuid.UUID `json:"budget_id" db:"budget_id"`
	AttemptNumber int       `json:"attempt_number" db:"attempt_number"`
	Action        string    `json:"action" db:"action"`
	ActorUserID   string    `json:"actor_user_id" db:"actor_user_id"`
	ActorEmail    string    `json:"actor_email,omitempty" db:"actor_email"`
	Reason        string    `json:"reason,omitempty" db:"reason"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

// AuditLogEntry represents a single audit log entry.
type AuditLogEntry struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	EntityType  string                 `json:"entity_type" db:"entity_type"`
	EntityID    string                 `json:"entity_id" db:"entity_id"`
	Action      string                 `json:"action" db:"action"`
	ActorUserID string                 `json:"actor_user_id,omitempty" db:"actor_user_id"`
	ActorEmail  string                 `json:"actor_email,omitempty" db:"actor_email"`
	OrgID       string                 `json:"org_id,omitempty" db:"org_id"`
	TeamID      string                 `json:"team_id,omitempty" db:"team_id"`
	Metadata    map[string]interface{} `json:"metadata,omitempty" db:"metadata"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
}

// ApprovalWithBudget joins an approval with its budget details for list views.
type ApprovalWithBudget struct {
	BudgetApproval
	BudgetName     string  `json:"budget_name"`
	BudgetAmount   float64 `json:"budget_amount_usd"`
	BudgetPeriod   string  `json:"budget_period"`
	OwnerOrgID     string  `json:"owner_org_id,omitempty"`
	OwnerTeamID    string  `json:"owner_team_id,omitempty"`
	CreatedByEmail string  `json:"created_by_email,omitempty"`
	ApprovalStatus string  `json:"approval_status"`
	RejectionCount int     `json:"rejection_count"`
}
