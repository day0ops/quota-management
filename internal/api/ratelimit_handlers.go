package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"time"

	"github.com/agentgateway/quota-management/internal/audit"
	"github.com/agentgateway/quota-management/internal/auth"
	"github.com/agentgateway/quota-management/internal/db"
	"github.com/agentgateway/quota-management/internal/models"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// RateLimitHandler provides HTTP handlers for rate limit allocations.
type RateLimitHandler struct {
	repo     *db.RateLimitRepository
	auditSvc *audit.Service
}

// NewRateLimitHandler creates new rate limit handlers.
func NewRateLimitHandler(repo *db.RateLimitRepository, auditSvc *audit.Service) *RateLimitHandler {
	return &RateLimitHandler{repo: repo, auditSvc: auditSvc}
}

// canEnableRateLimit checks if the identity can enable a disabled rate limit
func canEnableRateLimit(alloc *models.RateLimitAllocation, identity *auth.Identity) bool {
	if alloc.Enabled {
		return true // Already enabled - no-op, allow
	}
	// If disabled by org-admin, only org-admin can re-enable
	if alloc.DisabledByIsOrg && !identity.IsOrg {
		return false
	}
	return true
}

// RegisterRoutes registers rate limit routes on a mux.Router.
func (h *RateLimitHandler) RegisterRoutes(r *mux.Router) {
	r.Handle("/api/v1/rate-limits", auth.OptionalAuthMiddleware(http.HandlerFunc(h.ListAllocations))).Methods("GET")
	r.Handle("/api/v1/rate-limits", auth.OptionalAuthMiddleware(http.HandlerFunc(h.CreateAllocation))).Methods("POST")
	r.Handle("/api/v1/rate-limits/pending", auth.OptionalAuthMiddleware(http.HandlerFunc(h.ListPendingAllocations))).Methods("GET")
	r.Handle("/api/v1/rate-limits/pending/count", auth.OptionalAuthMiddleware(http.HandlerFunc(h.CountPendingAllocations))).Methods("GET")
	r.Handle("/api/v1/rate-limits/approvals/history", auth.OptionalAuthMiddleware(http.HandlerFunc(h.ListApprovalHistory))).Methods("GET")
	r.Handle("/api/v1/rate-limits/{id}", auth.OptionalAuthMiddleware(http.HandlerFunc(h.GetAllocation))).Methods("GET")
	r.Handle("/api/v1/rate-limits/{id}", auth.OptionalAuthMiddleware(http.HandlerFunc(h.UpdateAllocation))).Methods("PUT")
	r.Handle("/api/v1/rate-limits/{id}", auth.OptionalAuthMiddleware(http.HandlerFunc(h.DeleteAllocation))).Methods("DELETE")
	r.Handle("/api/v1/rate-limits/{id}/approve", auth.OptionalAuthMiddleware(http.HandlerFunc(h.ApproveAllocation))).Methods("POST")
	r.Handle("/api/v1/rate-limits/{id}/reject", auth.OptionalAuthMiddleware(http.HandlerFunc(h.RejectAllocation))).Methods("POST")
	r.Handle("/api/v1/rate-limits/{id}/resubmit", auth.OptionalAuthMiddleware(http.HandlerFunc(h.ResubmitAllocation))).Methods("POST")
}

// CreateAllocationRequest is the request body for creating an allocation.
type CreateAllocationRequest struct {
	OrgID           string  `json:"org_id"`
	TeamID          string  `json:"team_id"`
	ModelPattern    string  `json:"model_pattern"`
	TokenLimit      *int64  `json:"token_limit,omitempty"`
	TokenUnit       *string `json:"token_unit,omitempty"`
	RequestLimit    *int64  `json:"request_limit,omitempty"`
	RequestUnit     *string `json:"request_unit,omitempty"`
	BurstPercentage int     `json:"burst_percentage"`
	Description     string  `json:"description,omitempty"`
}

// ApproveRequest is the request body for approving an allocation.
type ApproveRequest struct {
	Enforcement string `json:"enforcement"`
}

// RejectRequest is the request body for rejecting an allocation.
type RejectRequest struct {
	Reason string `json:"reason"`
}

// UpdateAllocationRequest is the request body for updating an allocation.
type UpdateAllocationRequest struct {
	ModelPattern    string  `json:"model_pattern,omitempty"`
	TokenLimit      *int64  `json:"token_limit,omitempty"`
	TokenUnit       *string `json:"token_unit,omitempty"`
	RequestLimit    *int64  `json:"request_limit,omitempty"`
	RequestUnit     *string `json:"request_unit,omitempty"`
	BurstPercentage *int    `json:"burst_percentage,omitempty"`
	Enforcement     string  `json:"enforcement,omitempty"`
	Enabled         *bool   `json:"enabled,omitempty"`
	Description     string  `json:"description,omitempty"`
	Version         int64   `json:"version"`
}

// ListAllocations handles GET /api/v1/rate-limits
// Uses identity from JWT for RBAC filtering - ignores user-supplied org_id for security.
// Query params:
//   - enabled_only=true: Only return enabled and approved allocations (for "At a Glance" counts)
func (h *RateLimitHandler) ListAllocations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	identity := auth.GetIdentity(ctx)
	if identity == nil || identity.OrgID == "" {
		// Unauthenticated users get empty list
		writeJSON(w, http.StatusOK, []map[string]interface{}{})
		return
	}

	// Build filter from query params
	filter := db.RateLimitListFilter{
		EnabledOnly: r.URL.Query().Get("enabled_only") == "true",
	}

	// Use org_id from JWT, not from query params
	allocations, err := h.repo.ListAllocationsByOrgFiltered(ctx, identity.OrgID, filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, h.allocationsToMaps(allocations))
}

// ListPendingAllocations handles GET /api/v1/rate-limits/pending
func (h *RateLimitHandler) ListPendingAllocations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	identity := auth.GetIdentity(ctx)
	if identity == nil || identity.OrgID == "" {
		// Unauthenticated users get empty list
		writeJSON(w, http.StatusOK, []map[string]interface{}{})
		return
	}

	allocations, err := h.repo.ListPendingAllocations(ctx, identity.OrgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, h.allocationsToMaps(allocations))
}

// CountPendingAllocations handles GET /api/v1/rate-limits/pending/count
func (h *RateLimitHandler) CountPendingAllocations(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	identity := auth.GetIdentity(ctx)
	if identity == nil || identity.OrgID == "" {
		// Unauthenticated users get zero count
		writeJSON(w, http.StatusOK, map[string]int{"count": 0})
		return
	}

	count, err := h.repo.CountPendingAllocations(ctx, identity.OrgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"count": count})
}

// GetAllocation handles GET /api/v1/rate-limits/{id}
func (h *RateLimitHandler) GetAllocation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	allocation, err := h.repo.GetAllocationByID(ctx, id)
	if err != nil {
		if err == db.ErrNotFound {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, h.allocationToMap(allocation))
}

// CreateAllocation handles POST /api/v1/rate-limits
func (h *RateLimitHandler) CreateAllocation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateAllocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.OrgID == "" || req.TeamID == "" || req.ModelPattern == "" {
		writeError(w, http.StatusBadRequest, "org_id, team_id, and model_pattern are required")
		return
	}

	if req.TokenLimit == nil && req.RequestLimit == nil {
		writeError(w, http.StatusBadRequest, "at least one of token_limit or request_limit is required")
		return
	}

	identity := auth.GetIdentity(ctx)

	allocation := &models.RateLimitAllocation{
		OrgID:           req.OrgID,
		TeamID:          req.TeamID,
		ModelPattern:    req.ModelPattern,
		BurstPercentage: req.BurstPercentage,
		Enabled:         true,
		Enforcement:     models.EnforcementEnforced,
	}

	// Auto-approve if created by org admin, otherwise pending
	var actorUserID, actorEmail string
	if identity != nil {
		allocation.CreatedByUserID = sql.NullString{String: identity.Subject, Valid: true}
		allocation.CreatedByEmail = sql.NullString{String: identity.Email, Valid: true}
		actorUserID = identity.Subject
		actorEmail = identity.Email

		if identity.IsOrg {
			allocation.ApprovalStatus = models.ApprovalStatusApproved
			allocation.ApprovedBy = sql.NullString{String: identity.Email, Valid: identity.Email != ""}
		} else {
			allocation.ApprovalStatus = models.ApprovalStatusPending
		}
	} else {
		allocation.ApprovalStatus = models.ApprovalStatusApproved
	}

	if req.TokenLimit != nil {
		allocation.TokenLimit = sql.NullInt64{Int64: *req.TokenLimit, Valid: true}
	}
	if req.TokenUnit != nil {
		allocation.TokenUnit = sql.NullString{String: *req.TokenUnit, Valid: true}
	}
	if req.RequestLimit != nil {
		allocation.RequestLimit = sql.NullInt64{Int64: *req.RequestLimit, Valid: true}
	}
	if req.RequestUnit != nil {
		allocation.RequestUnit = sql.NullString{String: *req.RequestUnit, Valid: true}
	}
	if req.Description != "" {
		allocation.Description = sql.NullString{String: req.Description, Valid: true}
	}

	if err := h.repo.CreateAllocation(ctx, allocation); err != nil {
		if err == db.ErrDuplicateEntity {
			writeError(w, http.StatusConflict, "a rate limit for this team and model pattern already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Create approval record only for pending allocations (team users)
	if allocation.ApprovalStatus == models.ApprovalStatusPending {
		approval := &models.RateLimitApproval{
			AllocationID:  allocation.ID,
			AttemptNumber: 1,
			Action:        "submitted",
			ActorUserID:   sql.NullString{String: actorUserID, Valid: actorUserID != ""},
			ActorEmail:    sql.NullString{String: actorEmail, Valid: actorEmail != ""},
		}
		_ = h.repo.CreateApproval(ctx, approval)
	}

	// Audit log
	h.auditSvc.LogAction(ctx, "rate_limit", allocation.ID.String(), "created", identity, map[string]interface{}{
		"team_id":       allocation.TeamID,
		"model_pattern": allocation.ModelPattern,
		"auto_approved": allocation.ApprovalStatus == models.ApprovalStatusApproved,
	})

	writeJSON(w, http.StatusCreated, h.allocationToMap(allocation))
}

// UpdateAllocation handles PUT /api/v1/rate-limits/{id}
func (h *RateLimitHandler) UpdateAllocation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	existing, err := h.repo.GetAllocationByID(ctx, id)
	if err != nil {
		if err == db.ErrNotFound {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var req UpdateAllocationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Check optimistic lock version
	if req.Version > 0 && req.Version != existing.Version {
		writeError(w, http.StatusConflict, "concurrent modification")
		return
	}

	if req.ModelPattern != "" {
		existing.ModelPattern = req.ModelPattern
	}
	if req.TokenLimit != nil {
		existing.TokenLimit = sql.NullInt64{Int64: *req.TokenLimit, Valid: true}
	}
	if req.TokenUnit != nil {
		existing.TokenUnit = sql.NullString{String: *req.TokenUnit, Valid: true}
	}
	if req.RequestLimit != nil {
		existing.RequestLimit = sql.NullInt64{Int64: *req.RequestLimit, Valid: true}
	}
	if req.RequestUnit != nil {
		existing.RequestUnit = sql.NullString{String: *req.RequestUnit, Valid: true}
	}
	if req.BurstPercentage != nil {
		existing.BurstPercentage = *req.BurstPercentage
	}
	if req.Enforcement != "" {
		existing.Enforcement = models.Enforcement(req.Enforcement)
	}
	if req.Description != "" {
		existing.Description = sql.NullString{String: req.Description, Valid: true}
	}

	// Handle enable/disable with permission checks
	identity := auth.GetIdentity(ctx)
	if req.Enabled != nil {
		if *req.Enabled {
			// Enabling - check permissions
			if identity != nil && !canEnableRateLimit(existing, identity) {
				writeError(w, http.StatusForbidden, "You don't have permission to update this rate limit")
				return
			}
			existing.Enabled = true
			existing.DisabledByUserID = sql.NullString{}
			existing.DisabledByEmail = sql.NullString{}
			existing.DisabledByIsOrg = false
			existing.DisabledAt = sql.NullTime{}

			// Audit log for enable
			if identity != nil {
				h.auditSvc.LogAction(ctx, "rate_limit", existing.ID.String(), "rate_limit_enabled", identity, map[string]interface{}{
					"previous_disabled_by_is_org": existing.DisabledByIsOrg,
				})
			}
		} else {
			// Disabling
			existing.Enabled = false
			if identity != nil {
				existing.DisabledByUserID = sql.NullString{String: identity.Subject, Valid: true}
				existing.DisabledByEmail = sql.NullString{String: identity.Email, Valid: true}
				existing.DisabledByIsOrg = identity.IsOrg
				existing.DisabledAt = sql.NullTime{Time: time.Now(), Valid: true}

				// Audit log for disable
				h.auditSvc.LogAction(ctx, "rate_limit", existing.ID.String(), "rate_limit_disabled", identity, map[string]interface{}{
					"disabled_by_is_org": identity.IsOrg,
				})
			}
		}
	}

	if err := h.repo.UpdateAllocation(ctx, existing); err != nil {
		if err == db.ErrOptimisticLock {
			writeError(w, http.StatusConflict, "concurrent modification")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Audit log
	h.auditSvc.LogAction(ctx, "rate_limit", id.String(), "updated", auth.GetIdentity(ctx), map[string]interface{}{
		"team_id":       existing.TeamID,
		"model_pattern": existing.ModelPattern,
	})

	writeJSON(w, http.StatusOK, h.allocationToMap(existing))
}

// DeleteAllocation handles DELETE /api/v1/rate-limits/{id}
func (h *RateLimitHandler) DeleteAllocation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Get allocation first for audit log
	allocation, _ := h.repo.GetAllocationByID(ctx, id)

	if err := h.repo.DeleteAllocation(ctx, id); err != nil {
		if err == db.ErrNotFound {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Audit log
	metadata := map[string]interface{}{}
	if allocation != nil {
		metadata["team_id"] = allocation.TeamID
		metadata["model_pattern"] = allocation.ModelPattern
	}
	h.auditSvc.LogAction(ctx, "rate_limit", id.String(), "deleted", auth.GetIdentity(ctx), metadata)

	w.WriteHeader(http.StatusNoContent)
}

// ApproveAllocation handles POST /api/v1/rate-limits/{id}/approve
func (h *RateLimitHandler) ApproveAllocation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req ApproveRequest
	// Decode body if present, but don't fail on empty body
	_ = json.NewDecoder(r.Body).Decode(&req)

	enforcement := models.EnforcementEnforced
	if req.Enforcement == "monitoring" {
		enforcement = models.EnforcementMonitoring
	}

	identity := auth.GetIdentity(ctx)
	approvedBy := "admin"
	var actorUserID, actorEmail string
	if identity != nil {
		approvedBy = identity.Email
		if approvedBy == "" {
			approvedBy = identity.Subject
		}
		actorUserID = identity.Subject
		actorEmail = identity.Email
	}

	// Get the latest attempt number before approving
	attemptNumber, _ := h.repo.GetLatestAttemptNumber(ctx, id)
	if attemptNumber == 0 {
		attemptNumber = 1
	}

	if err := h.repo.ApproveAllocation(ctx, id, approvedBy, enforcement); err != nil {
		if err == db.ErrNotFound {
			writeError(w, http.StatusNotFound, "not found or already processed")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Create approval record
	approval := &models.RateLimitApproval{
		AllocationID:  id,
		AttemptNumber: attemptNumber,
		Action:        "approved",
		ActorUserID:   sql.NullString{String: actorUserID, Valid: actorUserID != ""},
		ActorEmail:    sql.NullString{String: actorEmail, Valid: actorEmail != ""},
	}
	_ = h.repo.CreateApproval(ctx, approval)

	allocation, err := h.repo.GetAllocationByID(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Audit log
	h.auditSvc.LogAction(ctx, "rate_limit", id.String(), "approved", identity, map[string]interface{}{
		"team_id":       allocation.TeamID,
		"model_pattern": allocation.ModelPattern,
		"enforcement":   enforcement,
	})

	writeJSON(w, http.StatusOK, h.allocationToMap(allocation))
}

// RejectAllocation handles POST /api/v1/rate-limits/{id}/reject
func (h *RateLimitHandler) RejectAllocation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	var req RejectRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	identity := auth.GetIdentity(ctx)
	var actorUserID, actorEmail string
	if identity != nil {
		actorUserID = identity.Subject
		actorEmail = identity.Email
	}

	// Get the latest attempt number before rejecting
	attemptNumber, _ := h.repo.GetLatestAttemptNumber(ctx, id)
	if attemptNumber == 0 {
		attemptNumber = 1
	}

	if err := h.repo.RejectAllocation(ctx, id); err != nil {
		if err == db.ErrNotFound {
			writeError(w, http.StatusNotFound, "not found or already processed")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Create rejection record
	approval := &models.RateLimitApproval{
		AllocationID:  id,
		AttemptNumber: attemptNumber,
		Action:        "rejected",
		ActorUserID:   sql.NullString{String: actorUserID, Valid: actorUserID != ""},
		ActorEmail:    sql.NullString{String: actorEmail, Valid: actorEmail != ""},
		Reason:        sql.NullString{String: req.Reason, Valid: req.Reason != ""},
	}
	_ = h.repo.CreateApproval(ctx, approval)

	allocation, err := h.repo.GetAllocationByID(ctx, id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Audit log
	h.auditSvc.LogAction(ctx, "rate_limit", id.String(), "rejected", identity, map[string]interface{}{
		"team_id":       allocation.TeamID,
		"model_pattern": allocation.ModelPattern,
		"reason":        req.Reason,
	})

	writeJSON(w, http.StatusOK, h.allocationToMap(allocation))
}

// allocationToMap converts a RateLimitAllocation to a map for JSON serialization.
func (h *RateLimitHandler) allocationToMap(a *models.RateLimitAllocation) map[string]interface{} {
	if a == nil {
		return nil
	}
	result := map[string]interface{}{
		"id":               a.ID,
		"org_id":           a.OrgID,
		"team_id":          a.TeamID,
		"model_pattern":    a.ModelPattern,
		"burst_percentage": a.BurstPercentage,
		"enforcement":      a.Enforcement,
		"enabled":          a.Enabled,
		"approval_status":  a.ApprovalStatus,
		"rejection_count":  a.RejectionCount,
		"version":          a.Version,
		"created_at":       a.CreatedAt,
		"updated_at":       a.UpdatedAt,
	}

	if a.TokenLimit.Valid {
		result["token_limit"] = a.TokenLimit.Int64
	}
	if a.TokenUnit.Valid {
		result["token_unit"] = a.TokenUnit.String
	}
	if a.RequestLimit.Valid {
		result["request_limit"] = a.RequestLimit.Int64
	}
	if a.RequestUnit.Valid {
		result["request_unit"] = a.RequestUnit.String
	}
	if a.ApprovedBy.Valid {
		result["approved_by"] = a.ApprovedBy.String
	}
	if a.ApprovedAt.Valid {
		result["approved_at"] = a.ApprovedAt.Time
	}
	if a.CreatedByUserID.Valid {
		result["created_by_user_id"] = a.CreatedByUserID.String
	}
	if a.CreatedByEmail.Valid {
		result["created_by_email"] = a.CreatedByEmail.String
	}
	if a.Description.Valid {
		result["description"] = a.Description.String
	}

	return result
}

// allocationsToMaps converts a slice of allocations to maps.
func (h *RateLimitHandler) allocationsToMaps(allocations []models.RateLimitAllocation) []map[string]interface{} {
	result := make([]map[string]interface{}, len(allocations))
	for i := range allocations {
		result[i] = h.allocationToMap(&allocations[i])
	}
	return result
}

// ListApprovalHistory handles GET /api/v1/rate-limits/approvals/history
// Org admins see all approval history in their org; team members see their team's history only.
func (h *RateLimitHandler) ListApprovalHistory(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	identity := auth.GetIdentity(ctx)

	if identity == nil || identity.OrgID == "" {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"data":       []map[string]interface{}{},
			"pagination": map[string]interface{}{"page": 1, "page_size": 30, "total_count": 0, "total_pages": 1},
		})
		return
	}

	var filter db.RateLimitApprovalHistoryFilter
	if identity.IsOrg {
		filter.OrgID = identity.OrgID
	} else if identity.TeamID != "" {
		filter.TeamID = identity.TeamID
	}

	pagination := parsePagination(r)

	approvals, total, err := h.repo.ListApprovalHistory(ctx, filter, pagination.Page, pagination.PageSize)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	totalPages := (total + pagination.PageSize - 1) / pagination.PageSize
	if totalPages == 0 {
		totalPages = 1
	}

	// Convert to maps for JSON
	data := make([]map[string]interface{}, len(approvals))
	for i, a := range approvals {
		item := map[string]interface{}{
			"id":             a.ID,
			"allocation_id":  a.AllocationID,
			"attempt_number": a.AttemptNumber,
			"action":         a.Action,
			"team_id":        a.TeamID,
			"model_pattern":  a.ModelPattern,
			"org_id":         a.OrgID,
			"created_at":     a.CreatedAt,
		}
		if a.ActorUserID.Valid {
			item["actor_user_id"] = a.ActorUserID.String
		}
		if a.ActorEmail.Valid {
			item["actor_email"] = a.ActorEmail.String
		}
		if a.Reason.Valid {
			item["reason"] = a.Reason.String
		}
		data[i] = item
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": data,
		"pagination": map[string]interface{}{
			"page":        pagination.Page,
			"page_size":   pagination.PageSize,
			"total_count": total,
			"total_pages": totalPages,
		},
	})
}

// ResubmitAllocation handles POST /api/v1/rate-limits/{id}/resubmit
func (h *RateLimitHandler) ResubmitAllocation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid id")
		return
	}

	// Get the current allocation to verify it's rejected
	allocation, err := h.repo.GetAllocationByID(ctx, id)
	if err != nil {
		if err == db.ErrNotFound {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if allocation.ApprovalStatus != models.ApprovalStatusRejected {
		writeError(w, http.StatusBadRequest, "can only resubmit rejected allocations")
		return
	}

	identity := auth.GetIdentity(ctx)
	var actorUserID, actorEmail string
	if identity != nil {
		actorUserID = identity.Subject
		actorEmail = identity.Email
	}

	// Get the latest attempt number and increment
	attemptNumber, _ := h.repo.GetLatestAttemptNumber(ctx, id)
	attemptNumber++

	// Reset the allocation status to pending
	if err := h.repo.ResetAllocationForResubmit(ctx, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Create resubmitted record
	approval := &models.RateLimitApproval{
		AllocationID:  id,
		AttemptNumber: attemptNumber,
		Action:        "resubmitted",
		ActorUserID:   sql.NullString{String: actorUserID, Valid: actorUserID != ""},
		ActorEmail:    sql.NullString{String: actorEmail, Valid: actorEmail != ""},
	}
	_ = h.repo.CreateApproval(ctx, approval)

	// Refresh allocation
	allocation, _ = h.repo.GetAllocationByID(ctx, id)

	// Audit log
	h.auditSvc.LogAction(ctx, "rate_limit", id.String(), "resubmitted", identity, map[string]interface{}{
		"team_id":        allocation.TeamID,
		"model_pattern":  allocation.ModelPattern,
		"attempt_number": attemptNumber,
	})

	writeJSON(w, http.StatusOK, h.allocationToMap(allocation))
}
