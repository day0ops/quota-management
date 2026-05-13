package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/agentgateway/quota-management/internal/audit"
	"github.com/agentgateway/quota-management/internal/auth"
	"github.com/agentgateway/quota-management/internal/db"
	"github.com/agentgateway/quota-management/internal/models"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// ApprovalHandler provides HTTP handlers for budget approval workflows.
type ApprovalHandler struct {
	repo     *db.Repository
	auditSvc *audit.Service
}

// NewApprovalHandler creates a new ApprovalHandler.
func NewApprovalHandler(repo *db.Repository, auditSvc *audit.Service) *ApprovalHandler {
	return &ApprovalHandler{repo: repo, auditSvc: auditSvc}
}

// ListPendingApprovals returns a paginated list of budgets pending approval,
// scoped to the caller's org if authenticated.
func (h *ApprovalHandler) ListPendingApprovals(w http.ResponseWriter, r *http.Request) {
	params := parsePagination(r)

	identity := auth.GetIdentity(r.Context())
	if identity != nil && !identity.IsOrg && !identity.IsTeamAdmin {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	if identity == nil || identity.OrgID == "" {
		writePaginatedJSON(w, http.StatusOK, []models.ApprovalWithBudget{}, params, 0)
		return
	}

	approvals, totalCount, err := h.repo.ListPendingApprovals(r.Context(), identity.OrgID, params.Offset(), params.PageSize)
	if err != nil {
		log.Error().Err(err).Msg("failed to list pending approvals")
		writeError(w, http.StatusInternalServerError, "failed to list pending approvals")
		return
	}

	writePaginatedJSON(w, http.StatusOK, approvals, params, totalCount)
}

// CountPendingApprovals returns the count of pending approvals for nav badge display.
// Shows count to all authenticated users so they can see their pending requests.
func (h *ApprovalHandler) CountPendingApprovals(w http.ResponseWriter, r *http.Request) {
	identity := auth.GetIdentity(r.Context())
	if identity != nil && !identity.IsOrg && !identity.IsTeamAdmin {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	if identity == nil || identity.OrgID == "" {
		writeJSON(w, http.StatusOK, map[string]int{"count": 0})
		return
	}

	count, err := h.repo.CountPendingApprovals(r.Context(), identity.OrgID)
	if err != nil {
		log.Error().Err(err).Msg("failed to count pending approvals")
		writeError(w, http.StatusInternalServerError, "failed to count pending approvals")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"count": count})
}

// ApproveBudget approves a pending budget. Requires org admin identity.
func (h *ApprovalHandler) ApproveBudget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	budgetID, err := uuid.Parse(vars["budget_id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid budget_id")
		return
	}

	identity := auth.GetIdentity(r.Context())
	if identity == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	if !identity.IsOrg && !identity.IsTeamAdmin {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	budget, err := h.repo.GetBudgetByID(r.Context(), budgetID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "budget not found")
			return
		}
		log.Error().Err(err).Str("budget_id", budgetID.String()).Msg("failed to get budget for approval")
		writeError(w, http.StatusInternalServerError, "failed to get budget")
		return
	}

	if !budget.OwnerOrgID.Valid || !auth.CanApproveOrReject(identity, budget.OwnerOrgID.String, budget.OwnerTeamID.String) {
		writeError(w, http.StatusForbidden, "forbidden: org admin role required")
		return
	}

	if budget.ApprovalStatus != models.ApprovalStatusPending {
		writeError(w, http.StatusConflict, "budget is not in pending status")
		return
	}

	if err := h.repo.UpdateBudgetApprovalStatus(r.Context(), budgetID, models.ApprovalStatusApproved, false); err != nil {
		log.Error().Err(err).Str("budget_id", budgetID.String()).Msg("failed to approve budget")
		writeError(w, http.StatusInternalServerError, "failed to approve budget")
		return
	}

	h.repo.CreateBudgetApproval(r.Context(), &models.BudgetApproval{
		BudgetID:      budgetID,
		AttemptNumber: budget.RejectionCount + 1,
		Action:        "approved",
		ActorUserID:   identity.Subject,
		ActorEmail:    identity.Email,
	})

	h.auditSvc.LogAction(r.Context(), "budget", budgetID.String(), "approved", identity, map[string]interface{}{
		"budget_name": budget.Name,
	})

	writeJSON(w, http.StatusOK, map[string]string{"message": "budget approved"})
}

// RejectBudget rejects a pending budget. Requires org admin identity. Reason is required.
// Auto-closes the budget after 3 rejections.
func (h *ApprovalHandler) RejectBudget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	budgetID, err := uuid.Parse(vars["budget_id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid budget_id")
		return
	}

	identity := auth.GetIdentity(r.Context())
	if identity == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	if !identity.IsOrg && !identity.IsTeamAdmin {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Reason == "" {
		writeError(w, http.StatusBadRequest, "reason is required")
		return
	}

	budget, err := h.repo.GetBudgetByID(r.Context(), budgetID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "budget not found")
			return
		}
		log.Error().Err(err).Str("budget_id", budgetID.String()).Msg("failed to get budget for rejection")
		writeError(w, http.StatusInternalServerError, "failed to get budget")
		return
	}

	if !budget.OwnerOrgID.Valid || !auth.CanApproveOrReject(identity, budget.OwnerOrgID.String, budget.OwnerTeamID.String) {
		writeError(w, http.StatusForbidden, "forbidden: org admin role required")
		return
	}

	if budget.ApprovalStatus != models.ApprovalStatusPending {
		writeError(w, http.StatusConflict, "budget is not in pending status")
		return
	}

	newRejectionCount := budget.RejectionCount + 1
	newStatus := models.ApprovalStatusRejected
	if newRejectionCount >= 3 {
		newStatus = models.ApprovalStatusClosed
	}

	if err := h.repo.UpdateBudgetApprovalStatus(r.Context(), budgetID, newStatus, true); err != nil {
		log.Error().Err(err).Str("budget_id", budgetID.String()).Msg("failed to reject budget")
		writeError(w, http.StatusInternalServerError, "failed to reject budget")
		return
	}

	h.repo.CreateBudgetApproval(r.Context(), &models.BudgetApproval{
		BudgetID:      budgetID,
		AttemptNumber: newRejectionCount,
		Action:        "rejected",
		ActorUserID:   identity.Subject,
		ActorEmail:    identity.Email,
		Reason:        req.Reason,
	})

	h.auditSvc.LogAction(r.Context(), "budget", budgetID.String(), "rejected", identity, map[string]interface{}{
		"budget_name":     budget.Name,
		"reason":          req.Reason,
		"rejection_count": newRejectionCount,
		"final_status":    string(newStatus),
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":         "budget rejected",
		"rejection_count": newRejectionCount,
		"status":          string(newStatus),
	})
}

// ResubmitBudget allows the original creator to resubmit a rejected budget.
// Only allowed if rejection_count < 3.
func (h *ApprovalHandler) ResubmitBudget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	budgetID, err := uuid.Parse(vars["budget_id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid budget_id")
		return
	}

	identity := auth.GetIdentity(r.Context())
	if identity == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return
	}

	budget, err := h.repo.GetBudgetByID(r.Context(), budgetID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "budget not found")
			return
		}
		log.Error().Err(err).Str("budget_id", budgetID.String()).Msg("failed to get budget for resubmission")
		writeError(w, http.StatusInternalServerError, "failed to get budget")
		return
	}

	if budget.ApprovalStatus != models.ApprovalStatusRejected {
		writeError(w, http.StatusConflict, "budget is not in rejected status")
		return
	}

	if budget.RejectionCount >= 3 {
		writeError(w, http.StatusConflict, "budget has been rejected too many times and cannot be resubmitted")
		return
	}

	if !auth.CanResubmit(identity, budget.CreatedByUserID.String) {
		writeError(w, http.StatusForbidden, "forbidden: only the original creator can resubmit")
		return
	}

	if err := h.repo.UpdateBudgetApprovalStatus(r.Context(), budgetID, models.ApprovalStatusPending, false); err != nil {
		log.Error().Err(err).Str("budget_id", budgetID.String()).Msg("failed to resubmit budget")
		writeError(w, http.StatusInternalServerError, "failed to resubmit budget")
		return
	}

	h.repo.CreateBudgetApproval(r.Context(), &models.BudgetApproval{
		BudgetID:      budgetID,
		AttemptNumber: budget.RejectionCount + 1,
		Action:        "resubmitted",
		ActorUserID:   identity.Subject,
		ActorEmail:    identity.Email,
	})

	h.auditSvc.LogAction(r.Context(), "budget", budgetID.String(), "resubmitted", identity, map[string]interface{}{
		"budget_name":     budget.Name,
		"rejection_count": budget.RejectionCount,
	})

	writeJSON(w, http.StatusOK, map[string]string{"message": "budget resubmitted for approval"})
}

// ListApprovalHistory returns the paginated approval action history.
// If budget_id is provided, returns history for that specific budget.
// Otherwise returns global history, scoped to the caller's org if authenticated.
func (h *ApprovalHandler) ListApprovalHistory(w http.ResponseWriter, r *http.Request) {
	identity := auth.GetIdentity(r.Context())
	if identity != nil && !identity.IsOrg && !identity.IsTeamAdmin {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	params := parsePagination(r)
	budgetIDStr := r.URL.Query().Get("budget_id")

	if budgetIDStr != "" {
		budgetID, err := uuid.Parse(budgetIDStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid budget_id")
			return
		}

		approvals, err := h.repo.ListApprovalHistory(r.Context(), budgetID)
		if err != nil {
			log.Error().Err(err).Str("budget_id", budgetIDStr).Msg("failed to list approval history")
			writeError(w, http.StatusInternalServerError, "failed to list approval history")
			return
		}

		totalCount := len(approvals)
		start := params.Offset()
		end := start + params.PageSize
		if start > totalCount {
			start = totalCount
		}
		if end > totalCount {
			end = totalCount
		}

		writePaginatedJSON(w, http.StatusOK, approvals[start:end], params, totalCount)
		return
	}

	var filter db.ApprovalHistoryFilter
	if identity != nil {
		if identity.IsOrg {
			filter.OrgID = identity.OrgID
		} else if identity.TeamID != "" {
			filter.TeamID = identity.TeamID
		}
	}

	history, totalCount, err := h.repo.ListAllApprovalHistory(r.Context(), filter, params.Offset(), params.PageSize)
	if err != nil {
		log.Error().Err(err).Msg("failed to list approval history")
		writeError(w, http.StatusInternalServerError, "failed to list approval history")
		return
	}

	writePaginatedJSON(w, http.StatusOK, history, params, totalCount)
}
