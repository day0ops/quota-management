package api

import (
	"net/http"
	"strconv"

	"github.com/agentgateway/quota-management/internal/auth"
	"github.com/agentgateway/quota-management/internal/db"
	"github.com/agentgateway/quota-management/internal/models"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

func alertToMap(a models.BudgetAlert) map[string]any {
	m := map[string]any{
		"id":                  a.ID.String(),
		"budget_id":           a.BudgetID.String(),
		"budget_name":         a.BudgetName,
		"entity_type":         a.EntityType,
		"alert_type":          string(a.AlertType),
		"status":              string(a.Status),
		"triggered_usage_usd": a.TriggeredUsageUSD,
		"budget_amount_usd":   a.BudgetAmountUSD,
		"period_start":        a.PeriodStart,
		"created_at":          a.CreatedAt,
	}
	if a.ThresholdPct.Valid {
		m["threshold_pct"] = a.ThresholdPct.Int32
	} else {
		m["threshold_pct"] = nil
	}
	if a.ProjectedSpendUSD.Valid {
		m["projected_spend_usd"] = a.ProjectedSpendUSD.Float64
	} else {
		m["projected_spend_usd"] = nil
	}
	if a.DaysUntilExhausted.Valid {
		m["days_until_exhausted"] = a.DaysUntilExhausted.Float64
	} else {
		m["days_until_exhausted"] = nil
	}
	return m
}

// ListAlerts handles GET /api/v1/alerts
func (h *Handler) ListAlerts(w http.ResponseWriter, r *http.Request) {
	identity := auth.GetIdentity(r.Context())

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "active"
	}

	var budgetID *uuid.UUID
	if idStr := r.URL.Query().Get("budget_id"); idStr != "" {
		id, err := uuid.Parse(idStr)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid budget_id")
			return
		}
		budgetID = &id
	}

	isOrgAdmin := identity != nil && identity.IsOrg
	orgID := ""
	teamID := ""
	if identity != nil {
		orgID = identity.OrgID
		teamID = identity.TeamID
	}

	filter := db.AlertListFilter{
		Status:       status,
		AlertType:    r.URL.Query().Get("alert_type"),
		BudgetID:     budgetID,
		IsOrgAdmin:   isOrgAdmin,
		CallerOrgID:  orgID,
		CallerTeamID: teamID,
		Page:         page,
		PageSize:     pageSize,
	}

	alerts, total, err := h.repo.ListAlerts(r.Context(), filter)
	if err != nil {
		log.Error().Err(err).Msg("list alerts")
		writeError(w, http.StatusInternalServerError, "failed to list alerts")
		return
	}

	rows := make([]map[string]any, len(alerts))
	for i, a := range alerts {
		rows[i] = alertToMap(a)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"alerts": rows,
		"pagination": map[string]any{
			"total_count": total,
			"page":        page,
			"page_size":   pageSize,
		},
	})
}

// CountAlerts handles GET /api/v1/alerts/count
func (h *Handler) CountAlerts(w http.ResponseWriter, r *http.Request) {
	identity := auth.GetIdentity(r.Context())

	isOrgAdmin := identity != nil && identity.IsOrg
	orgID := ""
	teamID := ""
	if identity != nil {
		orgID = identity.OrgID
		teamID = identity.TeamID
	}

	count, err := h.repo.CountActiveAlerts(r.Context(), isOrgAdmin, orgID, teamID)
	if err != nil {
		log.Error().Err(err).Msg("count alerts")
		writeError(w, http.StatusInternalServerError, "failed to count alerts")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"active": count})
}

// DismissAlert handles PUT /api/v1/alerts/:id/dismiss
func (h *Handler) DismissAlert(w http.ResponseWriter, r *http.Request) {
	identity := auth.GetIdentity(r.Context())

	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid alert id")
		return
	}

	userID := ""
	if identity != nil {
		userID = identity.UserID
		if userID == "" {
			userID = identity.Email
		}
	}

	if err := h.repo.DismissAlert(r.Context(), id, userID); err != nil {
		if err == db.ErrNotFound {
			writeError(w, http.StatusNotFound, "alert not found or already resolved")
			return
		}
		log.Error().Err(err).Msg("dismiss alert")
		writeError(w, http.StatusInternalServerError, "failed to dismiss alert")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// DismissAllAlerts handles POST /api/v1/alerts/dismiss-all
func (h *Handler) DismissAllAlerts(w http.ResponseWriter, r *http.Request) {
	identity := auth.GetIdentity(r.Context())

	isOrgAdmin := identity != nil && identity.IsOrg
	orgID, teamID, userID := "", "", ""
	if identity != nil {
		orgID = identity.OrgID
		teamID = identity.TeamID
		userID = identity.UserID
		if userID == "" {
			userID = identity.Email
		}
	}

	count, err := h.repo.DismissAllAlerts(r.Context(), isOrgAdmin, orgID, teamID, userID)
	if err != nil {
		log.Error().Err(err).Msg("dismiss all alerts")
		writeError(w, http.StatusInternalServerError, "failed to dismiss alerts")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"dismissed": count})
}
