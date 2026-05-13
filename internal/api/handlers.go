package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/agentgateway/quota-management/internal/audit"
	"github.com/agentgateway/quota-management/internal/auth"
	"github.com/agentgateway/quota-management/internal/cel"
	"github.com/agentgateway/quota-management/internal/db"
	"github.com/agentgateway/quota-management/internal/metrics"
	"github.com/agentgateway/quota-management/internal/models"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"
)

// Handler provides HTTP handlers for the management API.
type Handler struct {
	repo            *db.Repository
	celEvaluator    *cel.Evaluator
	auditSvc        *audit.Service
	approvalHandler *ApprovalHandler
	auditHandler    *AuditHandler
}

// NewHandler creates a new API handler.
func NewHandler(repo *db.Repository, celEvaluator *cel.Evaluator, auditSvc *audit.Service) *Handler {
	return &Handler{
		repo:            repo,
		celEvaluator:    celEvaluator,
		auditSvc:        auditSvc,
		approvalHandler: NewApprovalHandler(repo, auditSvc),
		auditHandler:    NewAuditHandler(repo),
	}
}

// RegisterRoutes registers all API routes.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Identity endpoint (uses optional auth to return identity if JWT present)
	r.Handle("/api/v1/identity", auth.OptionalAuthMiddleware(http.HandlerFunc(h.GetIdentity))).Methods("GET")

	// Model costs (require org admin auth for mutations, authenticated for reads)
	r.Handle("/api/v1/model-costs", auth.OptionalAuthMiddleware(http.HandlerFunc(h.ListModelCosts))).Methods("GET")
	r.Handle("/api/v1/model-costs", auth.OptionalAuthMiddleware(http.HandlerFunc(h.CreateModelCost))).Methods("POST")
	r.Handle("/api/v1/model-costs/providers", auth.OptionalAuthMiddleware(http.HandlerFunc(h.GetModelCostProviders))).Methods("GET")
	r.Handle("/api/v1/model-costs/{model_id}", auth.OptionalAuthMiddleware(http.HandlerFunc(h.GetModelCost))).Methods("GET")
	r.Handle("/api/v1/model-costs/{model_id}", auth.OptionalAuthMiddleware(http.HandlerFunc(h.UpdateModelCost))).Methods("PUT")
	r.Handle("/api/v1/model-costs/{model_id}", auth.OptionalAuthMiddleware(http.HandlerFunc(h.DeleteModelCost))).Methods("DELETE")

	// Budgets (use optional auth to filter by identity when authenticated)
	r.Handle("/api/v1/budgets", auth.OptionalAuthMiddleware(http.HandlerFunc(h.ListBudgets))).Methods("GET")
	r.Handle("/api/v1/budgets/parent-candidates", auth.OptionalAuthMiddleware(http.HandlerFunc(h.ListParentCandidates))).Methods("GET")
	r.Handle("/api/v1/budgets/forecast", auth.OptionalAuthMiddleware(http.HandlerFunc(h.ListBudgetForecasts))).Methods("GET")
	r.Handle("/api/v1/budgets", auth.OptionalAuthMiddleware(http.HandlerFunc(h.CreateBudget))).Methods("POST")
	r.Handle("/api/v1/budgets/{id}", auth.OptionalAuthMiddleware(http.HandlerFunc(h.GetBudget))).Methods("GET")
	r.Handle("/api/v1/budgets/{id}", auth.OptionalAuthMiddleware(http.HandlerFunc(h.UpdateBudget))).Methods("PUT")
	r.Handle("/api/v1/budgets/{id}", auth.OptionalAuthMiddleware(http.HandlerFunc(h.DeleteBudget))).Methods("DELETE")
	r.Handle("/api/v1/budgets/{id}/usage", auth.OptionalAuthMiddleware(http.HandlerFunc(h.GetBudgetUsage))).Methods("GET")
	r.Handle("/api/v1/budgets/{id}/forecast", auth.OptionalAuthMiddleware(http.HandlerFunc(h.GetBudgetForecast))).Methods("GET")
	r.Handle("/api/v1/budgets/{id}/reset", auth.OptionalAuthMiddleware(http.HandlerFunc(h.ResetBudget))).Methods("POST")
	r.Handle("/api/v1/budgets/{id}/children", auth.OptionalAuthMiddleware(http.HandlerFunc(h.GetBudgetChildren))).Methods("GET")

	// Alerts
	// NOTE: dismiss-all must be registered before {id}/dismiss to prevent mux
	// from matching "dismiss-all" as an {id} value.
	r.Handle("/api/v1/alerts", auth.OptionalAuthMiddleware(http.HandlerFunc(h.ListAlerts))).Methods("GET")
	r.Handle("/api/v1/alerts/count", auth.OptionalAuthMiddleware(http.HandlerFunc(h.CountAlerts))).Methods("GET")
	r.Handle("/api/v1/alerts/dismiss-all", auth.OptionalAuthMiddleware(http.HandlerFunc(h.DismissAllAlerts))).Methods("POST")
	r.Handle("/api/v1/alerts/{id}/dismiss", auth.OptionalAuthMiddleware(http.HandlerFunc(h.DismissAlert))).Methods("PUT")

	// CEL validation
	r.HandleFunc("/api/v1/validate-cel", h.ValidateCEL).Methods("POST")

	// Approvals (require auth)
	r.Handle("/api/v1/approvals", auth.OptionalAuthMiddleware(http.HandlerFunc(h.approvalHandler.ListPendingApprovals))).Methods("GET")
	r.Handle("/api/v1/approvals/count", auth.OptionalAuthMiddleware(http.HandlerFunc(h.approvalHandler.CountPendingApprovals))).Methods("GET")
	r.Handle("/api/v1/approvals/history", auth.OptionalAuthMiddleware(http.HandlerFunc(h.approvalHandler.ListApprovalHistory))).Methods("GET")
	r.Handle("/api/v1/approvals/{budget_id}/approve", auth.OptionalAuthMiddleware(http.HandlerFunc(h.approvalHandler.ApproveBudget))).Methods("POST")
	r.Handle("/api/v1/approvals/{budget_id}/reject", auth.OptionalAuthMiddleware(http.HandlerFunc(h.approvalHandler.RejectBudget))).Methods("POST")
	r.Handle("/api/v1/approvals/{budget_id}/resubmit", auth.OptionalAuthMiddleware(http.HandlerFunc(h.approvalHandler.ResubmitBudget))).Methods("POST")

	// Audit
	r.Handle("/api/v1/audit", auth.OptionalAuthMiddleware(http.HandlerFunc(h.auditHandler.ListAuditLogs))).Methods("GET")

	// Health check
	r.HandleFunc("/health", h.Health).Methods("GET")
	r.HandleFunc("/ready", h.Ready).Methods("GET")
}

// Response helpers

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Error().Err(err).Msg("failed to encode JSON response")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]interface{}{
		"error": map[string]string{
			"message": message,
		},
	})
}

// requireOrgAdmin checks if the request is from an authenticated org admin.
// Returns the identity if valid, nil otherwise (and writes error response).
func requireOrgAdmin(w http.ResponseWriter, r *http.Request) *auth.Identity {
	identity := auth.GetIdentity(r.Context())
	if identity == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return nil
	}
	if !identity.IsOrg {
		writeError(w, http.StatusForbidden, "org admin access required")
		return nil
	}
	return identity
}

// requireAuthenticated checks if the request is from an authenticated user.
// Returns the identity if valid, nil otherwise (and writes error response).
func requireAuthenticated(w http.ResponseWriter, r *http.Request) *auth.Identity {
	identity := auth.GetIdentity(r.Context())
	if identity == nil {
		writeError(w, http.StatusUnauthorized, "authentication required")
		return nil
	}
	return identity
}

func entityTypeName(et models.EntityType) string {
	switch et {
	case models.EntityTypeOrg:
		return "organization"
	case models.EntityTypeTeam:
		return "team"
	case models.EntityTypeUser:
		return "user"
	default:
		return string(et)
	}
}

// isRegularUser returns true if the identity is a regular user (not org admin, not team admin).
func isRegularUser(identity *auth.Identity) bool {
	if identity == nil {
		return false
	}
	return !identity.IsOrg && !identity.IsTeamAdmin
}

// canEnableBudget checks if the identity can enable a disabled budget
func canEnableBudget(budget *models.BudgetDefinition, identity *auth.Identity) bool {
	if budget.Enabled {
		return true // Already enabled - no-op, allow
	}
	// If disabled by org-admin, only org-admin can re-enable
	if budget.DisabledByIsOrg && !identity.IsOrg {
		return false
	}
	return true
}

// modelCostToMap converts a ModelCost to a map with proper JSON serialization
// for sql.Null* types (which otherwise serialize as {"String":"...", "Valid": true}).
func modelCostToMap(c *models.ModelCost) map[string]interface{} {
	result := map[string]interface{}{
		"id":                      c.ID,
		"model_id":                c.ModelID,
		"provider":                c.Provider,
		"input_cost_per_million":  c.InputCostPerMillion,
		"output_cost_per_million": c.OutputCostPerMillion,
		"effective_date":          c.EffectiveDate,
		"created_at":              c.CreatedAt,
		"updated_at":              c.UpdatedAt,
	}
	if c.CreatedByUserID.Valid {
		result["created_by_user_id"] = c.CreatedByUserID.String
	}
	if c.CreatedByEmail.Valid {
		result["created_by_email"] = c.CreatedByEmail.String
	}

	// Handle nullable fields - only include if valid
	if c.CacheReadCostMillion.Valid {
		result["cache_read_cost_million"] = c.CacheReadCostMillion.Float64
	}
	if c.CacheWriteCostMillion.Valid {
		result["cache_write_cost_million"] = c.CacheWriteCostMillion.Float64
	}
	if c.ModelPattern.Valid {
		result["model_pattern"] = c.ModelPattern.String
	}

	return result
}

// budgetToMap converts a BudgetDefinition to a map with proper JSON serialization.
func budgetToMap(b *models.BudgetDefinition) map[string]interface{} {
	result := map[string]interface{}{
		"id":                    b.ID,
		"entity_type":           b.EntityType,
		"name":                  b.Name,
		"match_expression":      b.MatchExpression,
		"budget_amount_usd":     b.BudgetAmountUSD,
		"period":                b.Period,
		"warning_threshold_pct": b.WarningThresholdPct,
		"parent_id":             b.ParentID,
		"isolated":              b.Isolated,
		"allow_fallback":        b.AllowFallback,
		"enabled":               b.Enabled,
		"disabled_by_is_org":    b.DisabledByIsOrg,
		"current_period_start":  b.CurrentPeriodStart,
		"current_usage_usd":     b.CurrentUsageUSD,
		"pending_usage_usd":     b.PendingUsageUSD,
		"remaining_usd":         b.CalculateRemaining(),
		"version":               b.Version,
		"created_at":            b.CreatedAt,
		"updated_at":            b.UpdatedAt,
		"approval_status":       string(b.ApprovalStatus),
		"rejection_count":       b.RejectionCount,
	}
	if b.CreatedByUserID.Valid {
		result["created_by_user_id"] = b.CreatedByUserID.String
	}
	if b.CreatedByEmail.Valid {
		result["created_by_email"] = b.CreatedByEmail.String
	}

	if b.Description.Valid {
		result["description"] = b.Description.String
	}
	if b.CustomPeriodSeconds.Valid {
		result["custom_period_seconds"] = b.CustomPeriodSeconds.Int32
	}
	if b.OwnerOrgID.Valid {
		result["owner_org_id"] = b.OwnerOrgID.String
	}
	if b.OwnerTeamID.Valid {
		result["owner_team_id"] = b.OwnerTeamID.String
	}
	if b.OwnerUserID.Valid {
		result["owner_user_id"] = b.OwnerUserID.String
	}

	return result
}

// Model Cost handlers

// ListModelCosts lists model costs with pagination.
// Requires authentication.
func (h *Handler) ListModelCosts(w http.ResponseWriter, r *http.Request) {
	if requireAuthenticated(w, r) == nil {
		return
	}

	identity := auth.GetIdentity(r.Context())
	if identity != nil && isRegularUser(identity) {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	params := parsePagination(r)

	sortBy := r.URL.Query().Get("sort_by")
	if sortBy != "input_cost" && sortBy != "output_cost" && sortBy != "both" {
		sortBy = ""
	}
	sortDir := r.URL.Query().Get("sort_dir")
	if sortDir != "asc" && sortDir != "desc" {
		sortDir = "asc"
	}

	filter := db.ModelCostFilter{
		Provider: r.URL.Query().Get("provider"),
		SortBy:   sortBy,
		SortDir:  sortDir,
	}

	costs, totalCount, err := h.repo.ListModelCostsPaginated(r.Context(), filter, params.Offset(), params.PageSize)
	if err != nil {
		log.Error().Err(err).Msg("failed to list model costs")
		writeError(w, http.StatusInternalServerError, "failed to list model costs")
		return
	}

	result := make([]map[string]interface{}, len(costs))
	for i, c := range costs {
		result[i] = modelCostToMap(&c)
	}

	writePaginatedJSON(w, http.StatusOK, result, params, totalCount)
}

// GetModelCostProviders returns all distinct provider names.
func (h *Handler) GetModelCostProviders(w http.ResponseWriter, r *http.Request) {
	providers, err := h.repo.ListDistinctProviders(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to list providers")
		writeError(w, http.StatusInternalServerError, "failed to list providers")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"providers": providers,
	})
}

// GetModelCost gets a model cost by model ID.
// Requires authentication.
func (h *Handler) GetModelCost(w http.ResponseWriter, r *http.Request) {
	if requireAuthenticated(w, r) == nil {
		return
	}

	identity := auth.GetIdentity(r.Context())
	if identity != nil && isRegularUser(identity) {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	vars := mux.Vars(r)
	modelID := vars["model_id"]

	cost, err := h.repo.GetModelCostByID(r.Context(), modelID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "model cost not found")
			return
		}
		log.Error().Err(err).Str("model_id", modelID).Msg("failed to get model cost")
		writeError(w, http.StatusInternalServerError, "failed to get model cost")
		return
	}

	writeJSON(w, http.StatusOK, modelCostToMap(cost))
}

// CreateModelCostRequest represents a create model cost request.
type CreateModelCostRequest struct {
	ModelID               string   `json:"model_id"`
	Provider              string   `json:"provider"`
	InputCostPerMillion   float64  `json:"input_cost_per_million"`
	OutputCostPerMillion  float64  `json:"output_cost_per_million"`
	CacheReadCostMillion  *float64 `json:"cache_read_cost_million,omitempty"`
	CacheWriteCostMillion *float64 `json:"cache_write_cost_million,omitempty"`
	ModelPattern          *string  `json:"model_pattern,omitempty"`
}

// CreateModelCost creates a new model cost.
// Requires org admin authentication.
func (h *Handler) CreateModelCost(w http.ResponseWriter, r *http.Request) {
	identity := requireOrgAdmin(w, r)
	if identity == nil {
		return
	}

	if isRegularUser(identity) {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	var req CreateModelCostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.ModelID == "" {
		writeError(w, http.StatusBadRequest, "model_id is required")
		return
	}

	if req.Provider == "" {
		writeError(w, http.StatusBadRequest, "provider is required")
		return
	}

	mc := &models.ModelCost{
		ModelID:              req.ModelID,
		Provider:             req.Provider,
		InputCostPerMillion:  req.InputCostPerMillion,
		OutputCostPerMillion: req.OutputCostPerMillion,
		EffectiveDate:        time.Now(),
		CreatedByUserID:      sql.NullString{String: identity.Subject, Valid: identity.Subject != ""},
		CreatedByEmail:       sql.NullString{String: identity.Email, Valid: identity.Email != ""},
	}

	if err := h.repo.CreateModelCost(r.Context(), mc); err != nil {
		log.Error().Err(err).Msg("failed to create model cost")
		writeError(w, http.StatusInternalServerError, "failed to create model cost")
		return
	}

	h.auditSvc.LogAction(r.Context(), "model_cost", mc.ModelID, "created", auth.GetIdentity(r.Context()), map[string]interface{}{
		"model_id": mc.ModelID,
		"provider": mc.Provider,
	})

	writeJSON(w, http.StatusCreated, mc)
}

// UpdateModelCost updates a model cost.
// Requires org admin authentication.
func (h *Handler) UpdateModelCost(w http.ResponseWriter, r *http.Request) {
	identity := requireOrgAdmin(w, r)
	if identity == nil {
		return
	}

	if isRegularUser(identity) {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	vars := mux.Vars(r)
	modelID := vars["model_id"]

	var req CreateModelCostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	mc := &models.ModelCost{
		ModelID:              modelID,
		Provider:             req.Provider,
		InputCostPerMillion:  req.InputCostPerMillion,
		OutputCostPerMillion: req.OutputCostPerMillion,
	}

	if err := h.repo.UpdateModelCost(r.Context(), mc); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "model cost not found")
			return
		}
		log.Error().Err(err).Str("model_id", modelID).Msg("failed to update model cost")
		writeError(w, http.StatusInternalServerError, "failed to update model cost")
		return
	}

	h.auditSvc.LogAction(r.Context(), "model_cost", modelID, "updated", auth.GetIdentity(r.Context()), map[string]interface{}{
		"model_id": modelID,
		"provider": mc.Provider,
	})

	writeJSON(w, http.StatusOK, mc)
}

// DeleteModelCost deletes a model cost.
// Requires org admin authentication.
func (h *Handler) DeleteModelCost(w http.ResponseWriter, r *http.Request) {
	identity := requireOrgAdmin(w, r)
	if identity == nil {
		return
	}

	if isRegularUser(identity) {
		writeError(w, http.StatusForbidden, "access denied")
		return
	}

	vars := mux.Vars(r)
	modelID := vars["model_id"]

	if err := h.repo.DeleteModelCost(r.Context(), modelID); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "model cost not found")
			return
		}
		log.Error().Err(err).Str("model_id", modelID).Msg("failed to delete model cost")
		writeError(w, http.StatusInternalServerError, "failed to delete model cost")
		return
	}

	h.auditSvc.LogAction(r.Context(), "model_cost", modelID, "deleted", auth.GetIdentity(r.Context()), map[string]interface{}{
		"model_id": modelID,
	})

	w.WriteHeader(http.StatusNoContent)
}

// Budget handlers

// ListBudgets lists budgets with pagination and SQL-level RBAC filtering.
// Query params:
//   - enabled_only=true: Only return enabled budgets (for "At a Glance" counts)
func (h *Handler) ListBudgets(w http.ResponseWriter, r *http.Request) {
	params := parsePagination(r)

	var filter db.BudgetListFilter
	identity := auth.GetIdentity(r.Context())
	if identity != nil {
		filter.OrgID = identity.OrgID
		filter.TeamID = identity.TeamID
		filter.IsOrg = identity.IsOrg
		filter.IsTeamAdmin = identity.IsTeamAdmin
		if !identity.IsOrg && !identity.IsTeamAdmin {
			filter.UserID = identity.UserID
		}
	}
	if r.URL.Query().Get("enabled_only") == "true" {
		filter.EnabledOnly = true
	}

	budgets, totalCount, err := h.repo.ListBudgetsPaginated(r.Context(), filter, params.Offset(), params.PageSize)
	if err != nil {
		log.Error().Err(err).Msg("failed to list budgets")
		writeError(w, http.StatusInternalServerError, "failed to list budgets")
		return
	}

	result := make([]map[string]interface{}, len(budgets))
	for i, b := range budgets {
		result[i] = budgetToMap(&b)
	}

	writePaginatedJSON(w, http.StatusOK, result, params, totalCount)
}

// ListParentCandidates returns org-level budgets that can be selected as parent budgets.
// Returns minimal data (id, name, amount, period) for dropdown selection.
func (h *Handler) ListParentCandidates(w http.ResponseWriter, r *http.Request) {
	identity := auth.GetIdentity(r.Context())
	if identity == nil || identity.OrgID == "" {
		// Unauthenticated users get empty list
		writeJSON(w, http.StatusOK, []db.ParentBudgetCandidate{})
		return
	}

	candidates, err := h.repo.ListParentCandidates(r.Context(), identity.OrgID)
	if err != nil {
		log.Error().Err(err).Msg("failed to list parent candidates")
		writeError(w, http.StatusInternalServerError, "failed to list parent candidates")
		return
	}

	writeJSON(w, http.StatusOK, candidates)
}

// GetBudget gets a budget by ID.
func (h *Handler) GetBudget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid budget ID")
		return
	}

	budget, err := h.repo.GetBudgetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "budget not found")
			return
		}
		log.Error().Err(err).Str("id", id.String()).Msg("failed to get budget")
		writeError(w, http.StatusInternalServerError, "failed to get budget")
		return
	}

	result := map[string]interface{}{
		"id":                    budget.ID,
		"entity_type":           budget.EntityType,
		"name":                  budget.Name,
		"match_expression":      budget.MatchExpression,
		"budget_amount_usd":     budget.BudgetAmountUSD,
		"period":                budget.Period,
		"warning_threshold_pct": budget.WarningThresholdPct,
		"parent_id":             budget.ParentID,
		"isolated":              budget.Isolated,
		"allow_fallback":        budget.AllowFallback,
		"enabled":               budget.Enabled,
		"disabled_by_is_org":    budget.DisabledByIsOrg,
		"current_period_start":  budget.CurrentPeriodStart,
		"current_usage_usd":     budget.CurrentUsageUSD,
		"pending_usage_usd":     budget.PendingUsageUSD,
		"remaining_usd":         budget.CalculateRemaining(),
		"next_period_start":     budget.NextPeriodStart(),
		"description":           budget.Description.String,
		"version":               budget.Version,
		"created_at":            budget.CreatedAt,
		"updated_at":            budget.UpdatedAt,
	}
	if budget.OwnerOrgID.Valid {
		result["owner_org_id"] = budget.OwnerOrgID.String
	}
	if budget.OwnerTeamID.Valid {
		result["owner_team_id"] = budget.OwnerTeamID.String
	}
	if budget.OwnerUserID.Valid {
		result["owner_user_id"] = budget.OwnerUserID.String
	}

	writeJSON(w, http.StatusOK, result)
}

// CreateBudgetRequest represents a create budget request.
type CreateBudgetRequest struct {
	EntityType          string  `json:"entity_type"`
	Name                string  `json:"name"`
	MatchExpression     string  `json:"match_expression"`
	BudgetAmountUSD     float64 `json:"budget_amount_usd"`
	Period              string  `json:"period"`
	CustomPeriodSeconds *int32  `json:"custom_period_seconds,omitempty"`
	WarningThresholdPct *int    `json:"warning_threshold_pct,omitempty"`
	ParentID            *string `json:"parent_id,omitempty"`
	Isolated            *bool   `json:"isolated,omitempty"`
	AllowFallback       *bool   `json:"allow_fallback,omitempty"`
	Enabled             *bool   `json:"enabled,omitempty"`
	Description         *string `json:"description,omitempty"`
	Version             *int64  `json:"version,omitempty"`
	OwnerOrgID          *string `json:"owner_org_id,omitempty"`
	OwnerTeamID         *string `json:"owner_team_id,omitempty"`
	OwnerUserID         *string `json:"owner_user_id,omitempty"`
}

// CreateBudget creates a new budget.
func (h *Handler) CreateBudget(w http.ResponseWriter, r *http.Request) {
	var req CreateBudgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	identity := auth.GetIdentity(r.Context())
	if identity != nil && isRegularUser(identity) {
		if req.EntityType != string(models.EntityTypeUser) {
			writeError(w, http.StatusForbidden, "regular users can only create user-type budgets")
			return
		}
	}

	if req.EntityType == "" {
		writeError(w, http.StatusBadRequest, "entity_type is required")
		return
	}

	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	if req.MatchExpression == "" {
		writeError(w, http.StatusBadRequest, "match_expression is required")
		return
	}

	if req.Period == "" {
		writeError(w, http.StatusBadRequest, "period is required")
		return
	}

	if req.Period == "custom" && (req.CustomPeriodSeconds == nil || *req.CustomPeriodSeconds < 3600) {
		writeError(w, http.StatusBadRequest, "custom_period_seconds must be at least 3600 (1 hour)")
		return
	}

	budget := &models.BudgetDefinition{
		EntityType:      models.EntityType(req.EntityType),
		Name:            req.Name,
		MatchExpression: req.MatchExpression,
		BudgetAmountUSD: req.BudgetAmountUSD,
		Period:          models.BudgetPeriod(req.Period),
		Isolated:        false, // Default to non-isolated
		Enabled:         true,  // Default to enabled
	}

	if req.CustomPeriodSeconds != nil {
		budget.CustomPeriodSeconds.Valid = true
		budget.CustomPeriodSeconds.Int32 = *req.CustomPeriodSeconds
	}

	if req.WarningThresholdPct != nil {
		budget.WarningThresholdPct = *req.WarningThresholdPct
	} else {
		budget.WarningThresholdPct = 80
	}

	var parentBudget *models.BudgetDefinition
	if req.ParentID != nil {
		parentID, err := uuid.Parse(*req.ParentID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid parent_id")
			return
		}
		budget.ParentID = &parentID

		// Get parent budget for inheritance and validation
		parentBudget, err = h.repo.GetBudgetByID(r.Context(), parentID)
		if err == nil {
			// Team budgets inherit isolated and allow_fallback from parent org
			budget.Isolated = parentBudget.Isolated
			budget.AllowFallback = parentBudget.AllowFallback
		}
	}

	// User budgets under a team inherit owner_team_id from parent team budget
	if budget.EntityType == models.EntityTypeUser && parentBudget != nil && parentBudget.OwnerTeamID.Valid {
		budget.OwnerTeamID.Valid = true
		budget.OwnerTeamID.String = parentBudget.OwnerTeamID.String
	}

	// Isolated and AllowFallback can only be set by org-admins or for org entity types
	// Team users cannot set these - they inherit from parent
	if req.Isolated != nil {
		if budget.EntityType == models.EntityTypeOrg {
			// Orgs can always set isolation
			budget.Isolated = *req.Isolated
		} else if identity != nil && identity.IsOrg {
			// Org-admins can set isolation on team budgets
			budget.Isolated = *req.Isolated
		}
		// Team users setting isolated is ignored - they inherit from parent
	}

	if req.AllowFallback != nil {
		if budget.EntityType == models.EntityTypeOrg {
			// Orgs can always set allow_fallback
			budget.AllowFallback = *req.AllowFallback
		} else if identity != nil && identity.IsOrg {
			// Org-admins can set allow_fallback on team budgets
			budget.AllowFallback = *req.AllowFallback
		}
		// Team users setting allow_fallback is ignored - they inherit from parent
	}

	// Validate: non-isolated child budget cannot exceed parent budget
	if parentBudget != nil && !budget.Isolated {
		if budget.BudgetAmountUSD > parentBudget.BudgetAmountUSD {
			writeError(w, http.StatusBadRequest, fmt.Sprintf(
				"team budget ($%.6f) cannot exceed parent org budget ($%.6f) for non-isolated budgets",
				budget.BudgetAmountUSD, parentBudget.BudgetAmountUSD))
			return
		}
	}

	if req.Enabled != nil {
		budget.Enabled = *req.Enabled
	}

	if req.Description != nil {
		budget.Description.Valid = true
		budget.Description.String = *req.Description
	}

	// Set ownership from request or identity
	if req.OwnerOrgID != nil && *req.OwnerOrgID != "" {
		budget.OwnerOrgID.Valid = true
		budget.OwnerOrgID.String = *req.OwnerOrgID
	}
	if req.OwnerTeamID != nil && *req.OwnerTeamID != "" {
		budget.OwnerTeamID.Valid = true
		budget.OwnerTeamID.String = *req.OwnerTeamID
	}
	if req.OwnerUserID != nil && *req.OwnerUserID != "" {
		budget.OwnerUserID.Valid = true
		budget.OwnerUserID.String = *req.OwnerUserID
	}

	// If no ownership specified in request, use identity
	if identity != nil && !budget.OwnerOrgID.Valid && !budget.OwnerTeamID.Valid {
		orgID, teamID := auth.GetOwnershipFromIdentity(identity)
		// For standalone team budgets (no parent), only set team ownership
		// This ensures they don't appear in org-admin approval lists
		if budget.ParentID == nil && teamID != "" {
			// Standalone team budget - only set team, not org
			budget.OwnerTeamID.Valid = true
			budget.OwnerTeamID.String = teamID
		} else {
			// Child budget or org budget - set both
			if orgID != "" {
				budget.OwnerOrgID.Valid = true
				budget.OwnerOrgID.String = orgID
			}
			if teamID != "" {
				budget.OwnerTeamID.Valid = true
				budget.OwnerTeamID.String = teamID
			}
		}
	}

	// For user-type budgets, identity takes precedence over any request-provided owner_user_id
	// to prevent users from creating budgets impersonating other users.
	if budget.EntityType == models.EntityTypeUser && identity != nil && identity.Subject != "" {
		budget.OwnerUserID = sql.NullString{String: identity.Subject, Valid: true}
	}

	// Set creator info from identity
	if identity != nil {
		budget.CreatedByUserID = sql.NullString{String: identity.Subject, Valid: identity.Subject != ""}
		budget.CreatedByEmail = sql.NullString{String: identity.Email, Valid: identity.Email != ""}

		if identity.IsOrg {
			// Org users are always auto-approved
			budget.ApprovalStatus = models.ApprovalStatusApproved
		} else if budget.ParentID == nil {
			// Team users creating standalone budgets (no parent) are auto-approved
			budget.ApprovalStatus = models.ApprovalStatusApproved
		} else {
			// Team users creating child budgets need approval
			budget.ApprovalStatus = models.ApprovalStatusPending
		}
	} else {
		budget.ApprovalStatus = models.ApprovalStatusApproved
	}

	// Reject if name already taken for this entity type
	if _, err := h.repo.GetBudgetByEntity(r.Context(), budget.EntityType, budget.Name); err == nil {
		writeError(w, http.StatusConflict, fmt.Sprintf("a budget named %q already exists for entity type %q", budget.Name, entityTypeName(budget.EntityType)))
		return
	}

	// Reject if match expression already used for this entity type
	if _, err := h.repo.GetBudgetByMatchExpression(r.Context(), budget.EntityType, budget.MatchExpression); err == nil {
		writeError(w, http.StatusConflict, fmt.Sprintf("a budget with this match expression already exists for entity type %q", entityTypeName(budget.EntityType)))
		return
	}

	if err := h.repo.CreateBudget(r.Context(), budget); err != nil {
		log.Error().Err(err).Msg("failed to create budget")
		writeError(w, http.StatusInternalServerError, "failed to create budget")
		return
	}

	// Create approval record for pending budgets
	if budget.ApprovalStatus == models.ApprovalStatusPending {
		h.repo.CreateBudgetApproval(r.Context(), &models.BudgetApproval{
			BudgetID:      budget.ID,
			AttemptNumber: 1,
			Action:        "submitted",
			ActorUserID:   budget.CreatedByUserID.String,
			ActorEmail:    budget.CreatedByEmail.String,
		})
	}

	// Audit log
	h.auditSvc.LogAction(r.Context(), "budget", budget.ID.String(), "created", identity, map[string]interface{}{
		"budget_name":     budget.Name,
		"approval_status": string(budget.ApprovalStatus),
	})

	writeJSON(w, http.StatusCreated, budgetToMap(budget))
}

// UpdateBudget updates a budget.
func (h *Handler) UpdateBudget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid budget ID")
		return
	}

	// Get existing budget
	existing, err := h.repo.GetBudgetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "budget not found")
			return
		}
		log.Error().Err(err).Str("id", id.String()).Msg("failed to get budget")
		writeError(w, http.StatusInternalServerError, "failed to get budget")
		return
	}

	var req CreateBudgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Update fields
	if req.MatchExpression != "" {
		existing.MatchExpression = req.MatchExpression
	}
	if req.BudgetAmountUSD > 0 {
		existing.BudgetAmountUSD = req.BudgetAmountUSD
	}
	if req.Period != "" {
		existing.Period = models.BudgetPeriod(req.Period)
	}
	if req.CustomPeriodSeconds != nil {
		if *req.CustomPeriodSeconds < 3600 {
			writeError(w, http.StatusBadRequest, "custom_period_seconds must be at least 3600 (1 hour)")
			return
		}
		existing.CustomPeriodSeconds.Valid = true
		existing.CustomPeriodSeconds.Int32 = *req.CustomPeriodSeconds
	}
	// Validate effective period after applying changes
	if existing.Period == models.PeriodCustom && (!existing.CustomPeriodSeconds.Valid || existing.CustomPeriodSeconds.Int32 < 3600) {
		writeError(w, http.StatusBadRequest, "custom_period_seconds must be at least 3600 (1 hour)")
		return
	}
	if req.WarningThresholdPct != nil {
		existing.WarningThresholdPct = *req.WarningThresholdPct
	}
	if req.ParentID != nil {
		parentID, err := uuid.Parse(*req.ParentID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid parent_id")
			return
		}
		existing.ParentID = &parentID
	}

	// Get identity for permission checks
	identity := auth.GetIdentity(r.Context())

	// Isolation can only be modified by org-admins or for org entity types
	// Isolated and AllowFallback can only be modified by org-admins or for org entity types
	// Team users cannot change these - they inherit from parent
	if req.Isolated != nil {
		if existing.EntityType == models.EntityTypeOrg {
			// Orgs can always set isolation
			existing.Isolated = *req.Isolated
		} else if identity != nil && identity.IsOrg {
			// Org-admins can set isolation on team budgets
			existing.Isolated = *req.Isolated
		}
		// Team users setting isolated is silently ignored - they inherit from parent
	}

	if req.AllowFallback != nil {
		if existing.EntityType == models.EntityTypeOrg {
			// Orgs can always set allow_fallback
			existing.AllowFallback = *req.AllowFallback
		} else if identity != nil && identity.IsOrg {
			// Org-admins can set allow_fallback on team budgets
			existing.AllowFallback = *req.AllowFallback
		}
		// Team users setting allow_fallback is silently ignored - they inherit from parent
	}

	if req.Enabled != nil {
		if *req.Enabled {
			// Only check permissions when actually transitioning from disabled to enabled
			if !existing.Enabled {
				// Enabling - check permissions
				if identity != nil && !canEnableBudget(existing, identity) {
					writeError(w, http.StatusForbidden, "You don't have permission to update this budget")
					return
				}
				existing.Enabled = true
				existing.DisabledByUserID = sql.NullString{}
				existing.DisabledByEmail = sql.NullString{}
				existing.DisabledByIsOrg = false
				existing.DisabledAt = sql.NullTime{}

				// Audit log for enable
				if identity != nil {
					h.auditSvc.LogAction(r.Context(), "budget", existing.ID.String(), "budget_enabled", identity, map[string]interface{}{
						"previous_disabled_by_is_org": existing.DisabledByIsOrg,
					})
				}
			}
			// If already enabled, no action needed
		} else {
			// Disabling
			existing.Enabled = false
			if identity != nil {
				existing.DisabledByUserID = sql.NullString{String: identity.Subject, Valid: true}
				existing.DisabledByEmail = sql.NullString{String: identity.Email, Valid: true}
				existing.DisabledByIsOrg = identity.IsOrg
				existing.DisabledAt = sql.NullTime{Time: time.Now(), Valid: true}

				// Audit log for disable
				h.auditSvc.LogAction(r.Context(), "budget", existing.ID.String(), "budget_disabled", identity, map[string]interface{}{
					"disabled_by_is_org": identity.IsOrg,
					"cascaded":           false,
				})

				// Cascade disable to non-isolated children (org-admin only)
				if identity.IsOrg {
					if err := h.disableNonIsolatedChildren(r.Context(), existing.ID, identity); err != nil {
						log.Printf("Warning: failed to cascade disable: %v", err)
					}
				}
			}
		}
	}
	if req.Description != nil {
		existing.Description.Valid = true
		existing.Description.String = *req.Description
	}
	if req.Version != nil {
		existing.Version = *req.Version
	}
	if req.OwnerOrgID != nil {
		if *req.OwnerOrgID == "" {
			existing.OwnerOrgID.Valid = false
			existing.OwnerOrgID.String = ""
		} else {
			existing.OwnerOrgID.Valid = true
			existing.OwnerOrgID.String = *req.OwnerOrgID
		}
	}
	if req.OwnerTeamID != nil {
		if *req.OwnerTeamID == "" {
			existing.OwnerTeamID.Valid = false
			existing.OwnerTeamID.String = ""
		} else {
			existing.OwnerTeamID.Valid = true
			existing.OwnerTeamID.String = *req.OwnerTeamID
		}
	}
	if req.OwnerUserID != nil {
		if *req.OwnerUserID == "" {
			existing.OwnerUserID.Valid = false
			existing.OwnerUserID.String = ""
		} else {
			existing.OwnerUserID.Valid = true
			existing.OwnerUserID.String = *req.OwnerUserID
		}
	}

	// Validate: non-isolated child budget cannot exceed parent budget
	if existing.ParentID != nil && !existing.Isolated {
		parent, err := h.repo.GetBudgetByID(r.Context(), *existing.ParentID)
		if err == nil && existing.BudgetAmountUSD > parent.BudgetAmountUSD {
			writeError(w, http.StatusBadRequest, fmt.Sprintf(
				"team budget ($%.6f) cannot exceed parent org budget ($%.6f) for non-isolated budgets",
				existing.BudgetAmountUSD, parent.BudgetAmountUSD))
			return
		}
	}

	if err := h.repo.UpdateBudget(r.Context(), existing); err != nil {
		if errors.Is(err, db.ErrOptimisticLock) {
			writeError(w, http.StatusConflict, "budget was modified by another request")
			return
		}
		log.Error().Err(err).Str("id", id.String()).Msg("failed to update budget")
		writeError(w, http.StatusInternalServerError, "failed to update budget")
		return
	}

	h.auditSvc.LogAction(r.Context(), "budget", id.String(), "updated", auth.GetIdentity(r.Context()), map[string]interface{}{
		"budget_name": existing.Name,
	})

	writeJSON(w, http.StatusOK, budgetToMap(existing))
}

// disableNonIsolatedChildren disables all non-isolated child budgets
func (h *Handler) disableNonIsolatedChildren(ctx context.Context, parentID uuid.UUID, identity *auth.Identity) error {
	children, err := h.repo.GetChildBudgets(ctx, parentID)
	if err != nil {
		return err
	}

	for _, child := range children {
		if child.Isolated || !child.Enabled {
			continue // Skip isolated or already disabled
		}

		child.Enabled = false
		child.DisabledByUserID = sql.NullString{String: identity.Subject, Valid: true}
		child.DisabledByEmail = sql.NullString{String: identity.Email, Valid: true}
		child.DisabledByIsOrg = true
		child.DisabledAt = sql.NullTime{Time: time.Now(), Valid: true}

		if err := h.repo.UpdateBudget(ctx, &child); err != nil {
			log.Printf("Warning: failed to disable child budget %s: %v", child.ID, err)
			continue
		}

		// Audit log for cascaded disable
		h.auditSvc.LogAction(ctx, "budget", child.ID.String(), "budget_disabled", identity, map[string]interface{}{
			"disabled_by_is_org": true,
			"cascaded":           true,
			"cascaded_from":      parentID.String(),
		})
	}

	return nil
}

// DeleteBudget deletes a budget.
// Query params:
//   - cascade=true: Delete all descendant budgets as well (children, grandchildren, etc.)
func (h *Handler) DeleteBudget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid budget ID")
		return
	}

	cascade := r.URL.Query().Get("cascade") == "true"

	// Fetch budget first to get info for metrics cleanup
	budget, err := h.repo.GetBudgetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "budget not found")
			return
		}
		log.Error().Err(err).Str("id", id.String()).Msg("failed to fetch budget for deletion")
		writeError(w, http.StatusInternalServerError, "failed to delete budget")
		return
	}

	// Check if budget has children
	children, err := h.repo.GetChildBudgets(r.Context(), id)
	if err != nil {
		log.Error().Err(err).Str("id", id.String()).Msg("failed to check for child budgets")
		writeError(w, http.StatusInternalServerError, "failed to delete budget")
		return
	}

	if len(children) > 0 && !cascade {
		writeError(w, http.StatusConflict, fmt.Sprintf("cannot delete budget: %d child budget(s) exist. Delete children first or use cascade=true.", len(children)))
		return
	}

	var deletedCount int
	if cascade && len(children) > 0 {
		// Cascade delete: delete all descendants + parent in a transaction
		deletedCount, err = h.repo.DeleteBudgetCascade(r.Context(), id)
		if err != nil {
			if errors.Is(err, db.ErrNotFound) {
				writeError(w, http.StatusNotFound, "budget not found")
				return
			}
			log.Error().Err(err).Str("id", id.String()).Msg("failed to cascade delete budget")
			writeError(w, http.StatusInternalServerError, "failed to delete budget")
			return
		}
	} else {
		// Simple delete (no children or no cascade)
		if err := h.repo.DeleteBudget(r.Context(), id); err != nil {
			if errors.Is(err, db.ErrNotFound) {
				writeError(w, http.StatusNotFound, "budget not found")
				return
			}
			log.Error().Err(err).Str("id", id.String()).Msg("failed to delete budget")
			writeError(w, http.StatusInternalServerError, "failed to delete budget")
			return
		}
		deletedCount = 1
	}

	// Clean up Prometheus metrics for the deleted budget
	metrics.DeleteBudgetMetrics(string(budget.EntityType), budget.Name, string(budget.Period))

	h.auditSvc.LogAction(r.Context(), "budget", id.String(), "deleted", auth.GetIdentity(r.Context()), map[string]interface{}{
		"budget_name":   budget.Name,
		"cascade":       cascade,
		"deleted_count": deletedCount,
	})

	w.WriteHeader(http.StatusNoContent)
}

// GetBudgetUsage gets usage history for a budget with pagination.
func (h *Handler) GetBudgetUsage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid budget ID")
		return
	}

	params := parsePagination(r)

	records, totalCount, err := h.repo.GetUsageByBudgetIDPaginated(r.Context(), id, params.Offset(), params.PageSize)
	if err != nil {
		log.Error().Err(err).Str("id", id.String()).Msg("failed to get usage records")
		writeError(w, http.StatusInternalServerError, "failed to get usage records")
		return
	}

	writePaginatedJSON(w, http.StatusOK, records, params, totalCount)
}

// ResetBudget resets the usage for a budget.
func (h *Handler) ResetBudget(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid budget ID")
		return
	}

	// Get budget details for audit log
	budget, err := h.repo.GetBudgetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "budget not found")
			return
		}
		log.Error().Err(err).Str("id", id.String()).Msg("failed to get budget")
		writeError(w, http.StatusInternalServerError, "failed to get budget")
		return
	}

	previousUsage := budget.CurrentUsageUSD

	if err := h.repo.ResetBudgetUsage(r.Context(), id); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeError(w, http.StatusNotFound, "budget not found")
			return
		}
		log.Error().Err(err).Str("id", id.String()).Msg("failed to reset budget")
		writeError(w, http.StatusInternalServerError, "failed to reset budget")
		return
	}

	h.auditSvc.LogAction(r.Context(), "budget", id.String(), "budget_reset", auth.GetIdentity(r.Context()), map[string]interface{}{
		"budget_name":    budget.Name,
		"previous_usage": previousUsage,
	})

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "budget reset successfully",
	})
}

// GetBudgetChildren returns all child budgets
func (h *Handler) GetBudgetChildren(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := uuid.Parse(vars["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid budget ID")
		return
	}

	children, err := h.repo.GetChildBudgets(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to get children")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": children,
	})
}

// GetIdentity returns the authenticated user's identity.
func (h *Handler) GetIdentity(w http.ResponseWriter, r *http.Request) {
	identity := auth.GetIdentity(r.Context())
	if identity == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"authenticated": false,
		})
		return
	}

	// Determine role string
	role := "user"
	if identity.IsOrg {
		role = "org_admin"
	} else if identity.IsTeamAdmin {
		role = "team_admin"
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"authenticated": true,
		"subject":       identity.Subject,
		"email":         identity.Email,
		"org_id":        identity.OrgID,
		"team_id":       identity.TeamID,
		"is_org":        identity.IsOrg,
		"is_team_admin": identity.IsTeamAdmin,
		"role":          role,
	})
}

// Health returns the health status.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
	})
}

// Ready returns the readiness status.
func (h *Handler) Ready(w http.ResponseWriter, r *http.Request) {
	// Simple database connectivity check
	if err := h.repo.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "not ready",
			"error":  err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ready",
	})
}

// ValidateCELRequest represents a CEL validation request.
type ValidateCELRequest struct {
	Expression string `json:"expression"`
}

// ValidateCEL validates a CEL expression.
func (h *Handler) ValidateCEL(w http.ResponseWriter, r *http.Request) {
	var req ValidateCELRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Expression == "" {
		writeError(w, http.StatusBadRequest, "expression is required")
		return
	}

	err := h.celEvaluator.ValidateExpression(req.Expression)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"valid": true,
	})
}
