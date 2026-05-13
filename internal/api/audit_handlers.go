package api

import (
	"net/http"
	"time"

	"github.com/agentgateway/quota-management/internal/auth"
	"github.com/agentgateway/quota-management/internal/db"
	"github.com/rs/zerolog/log"
)

// AuditHandler provides HTTP handlers for audit log access.
type AuditHandler struct {
	repo *db.Repository
}

// NewAuditHandler creates a new AuditHandler.
func NewAuditHandler(repo *db.Repository) *AuditHandler {
	return &AuditHandler{repo: repo}
}

// ListAuditLogs returns a paginated list of audit log entries.
// Org admins see all entries for their org; team members see their team's entries only.
// Supports query param filters: entity_type, action, actor, from, to (RFC3339).
func (h *AuditHandler) ListAuditLogs(w http.ResponseWriter, r *http.Request) {
	identity := auth.GetIdentity(r.Context())
	if identity != nil && !identity.IsOrg && !identity.IsTeamAdmin {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	params := parsePagination(r)

	filter := db.AuditFilter{}

	if identity != nil {
		if identity.IsOrg {
			filter.OrgID = identity.OrgID
		} else if identity.TeamID != "" {
			filter.TeamID = identity.TeamID
		}
	}

	if v := r.URL.Query().Get("entity_type"); v != "" {
		filter.EntityType = v
	}

	if v := r.URL.Query().Get("action"); v != "" {
		filter.Action = v
	}

	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	entries, totalCount, err := h.repo.ListAuditLogs(r.Context(), filter, params.Offset(), params.PageSize)
	if err != nil {
		log.Error().Err(err).Msg("failed to list audit logs")
		writeError(w, http.StatusInternalServerError, "failed to list audit logs")
		return
	}

	// Apply actor and date-range filters in-memory since they are not supported in AuditFilter SQL.
	actor := r.URL.Query().Get("actor")
	var from, to time.Time
	if fromStr != "" {
		from, _ = time.Parse(time.RFC3339, fromStr)
	}
	if toStr != "" {
		to, _ = time.Parse(time.RFC3339, toStr)
	}

	if actor != "" || !from.IsZero() || !to.IsZero() {
		filtered := entries[:0]
		for _, e := range entries {
			if actor != "" && e.ActorEmail != actor && e.ActorUserID != actor {
				continue
			}
			if !from.IsZero() && e.CreatedAt.Before(from) {
				continue
			}
			if !to.IsZero() && e.CreatedAt.After(to) {
				continue
			}
			filtered = append(filtered, e)
		}
		entries = filtered
		totalCount = len(entries)
	}

	writePaginatedJSON(w, http.StatusOK, entries, params, totalCount)
}
