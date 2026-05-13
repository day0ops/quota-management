// Budget periods
export type BudgetPeriod = 'hourly' | 'daily' | 'weekly' | 'monthly' | 'custom';

// Entity types
export type EntityType = 'org' | 'team' | 'user';

// Model Cost
export interface ModelCost {
  id: string;
  model_id: string;
  provider: string;
  input_cost_per_million: number;
  output_cost_per_million: number;
  cache_read_cost_million?: number;
  cache_write_cost_million?: number;
  model_pattern?: string;
  effective_date: string;
  created_at: string;
  updated_at: string;
  created_by_user_id?: string;
  created_by_email?: string;
}

export interface CreateModelCostRequest {
  model_id: string;
  provider: string;
  input_cost_per_million: number;
  output_cost_per_million: number;
  cache_read_cost_million?: number;
  cache_write_cost_million?: number;
  model_pattern?: string;
}

export interface UpdateModelCostRequest {
  provider?: string;
  input_cost_per_million?: number;
  output_cost_per_million?: number;
  cache_read_cost_million?: number;
  cache_write_cost_million?: number;
  model_pattern?: string;
}

// Budget Definition
export interface BudgetDefinition {
  id: string;
  entity_type: EntityType;
  name: string;
  match_expression: string;
  budget_amount_usd: number;
  period: BudgetPeriod;
  custom_period_seconds?: number;
  warning_threshold_pct: number;
  parent_id?: string;
  isolated: boolean;
  allow_fallback: boolean;
  enabled: boolean;
  disabled_by_user_id?: string;
  disabled_by_email?: string;
  disabled_by_is_org: boolean;
  disabled_at?: string;
  can_enable?: boolean;
  current_period_start: string;
  current_usage_usd: number;
  pending_usage_usd: number;
  remaining_usd: number;
  next_period_start?: string;
  description?: string;
  owner_org_id?: string;
  owner_team_id?: string;
  version?: number;
  created_at: string;
  updated_at: string;
  approval_status?: ApprovalStatus;
  created_by_user_id?: string;
  created_by_email?: string;
  rejection_count?: number;
}

export interface CreateBudgetRequest {
  entity_type: EntityType;
  name: string;
  match_expression: string;
  budget_amount_usd: number;
  period: BudgetPeriod;
  custom_period_seconds?: number;
  warning_threshold_pct?: number;
  parent_id?: string;
  isolated?: boolean;
  allow_fallback?: boolean;
  enabled?: boolean;
  description?: string;
  owner_org_id?: string;
  owner_team_id?: string;
}

export interface UpdateBudgetRequest {
  match_expression?: string;
  budget_amount_usd?: number;
  period?: BudgetPeriod;
  custom_period_seconds?: number;
  warning_threshold_pct?: number;
  parent_id?: string;
  isolated?: boolean;
  allow_fallback?: boolean;
  enabled?: boolean;
  description?: string;
  version?: number;
}

// Forecast
export interface TimeSeriesPoint {
  date: string;
  actual_usd?: number | null;
  projected_usd?: number | null;
}

export interface BudgetForecast {
  budget_id: string;
  period_start: string;
  period_end: string;
  budget_amount_usd: number;
  current_usage_usd: number;
  projected_spend_usd: number;
  days_until_exhausted: number; // -1 = on track this period
  burn_rate_usd_per_day: number;
  confidence: number; // 0.0-1.0
  time_series: TimeSeriesPoint[];
  period_boundaries?: string[]; // UTC date strings where historical periods reset
}

// Usage Record
export interface UsageRecord {
  id: string;
  budget_id: string;
  request_id: string;
  model_id: string;
  input_tokens: number;
  output_tokens: number;
  cost_usd: number;
  parent_charged: boolean;
  created_at: string;
}

// API Responses
export interface ListModelCostsResponse {
  model_costs: ModelCost[];
}

export interface ListBudgetsResponse {
  budgets: BudgetDefinition[];
}

export interface UsageHistoryResponse {
  usage_records: UsageRecord[];
}

export interface ApiError {
  error: {
    message: string;
  };
}

// CEL Validation
export interface ValidateCELRequest {
  expression: string;
}

export interface ValidateCELResponse {
  valid: boolean;
  error?: string;
}

// Identity / Auth
export interface Identity {
  authenticated: boolean;
  subject?: string;
  email?: string;
  org_id?: string;
  team_id?: string;
  is_org?: boolean;
}

// Approval Status
export type ApprovalStatus = 'pending' | 'approved' | 'rejected' | 'closed';

// Budget Approval
export interface BudgetApproval {
  id: string;
  budget_id: string;
  attempt_number: number;
  action: string;
  actor_user_id: string;
  actor_email?: string;
  reason?: string;
  created_at: string;
  updated_at: string;
}

// Approval with budget details (for list views)
export interface ApprovalWithBudget extends BudgetApproval {
  budget_name: string;
  budget_amount_usd: number;
  budget_period: string;
  owner_org_id?: string;
  owner_team_id?: string;
  created_by_email?: string;
  approval_status: ApprovalStatus;
  rejection_count: number;
}

// Audit Log Entry
export interface AuditLogEntry {
  id: string;
  entity_type: string;
  entity_id: string;
  action: string;
  actor_user_id?: string;
  actor_email?: string;
  org_id?: string;
  team_id?: string;
  metadata?: Record<string, unknown>;
  created_at: string;
}

// Pagination
export interface PaginationMeta {
  page: number;
  page_size: number;
  total_count: number;
  total_pages: number;
}

export interface PaginatedResponse<T> {
  data: T[];
  pagination: PaginationMeta;
}

// Rate Limit Types
export type TimeUnit = 'SECOND' | 'MINUTE' | 'HOUR' | 'DAY';
export type Enforcement = 'enforced' | 'monitoring';

export interface RateLimitAllocation {
  id: string;
  org_id: string;
  team_id: string;
  model_pattern: string;
  token_limit?: number;
  token_unit?: TimeUnit;
  request_limit?: number;
  request_unit?: TimeUnit;
  burst_percentage: number;
  enforcement: Enforcement;
  enabled: boolean;
  disabled_by_user_id?: string;
  disabled_by_email?: string;
  disabled_by_is_org: boolean;
  disabled_at?: string;
  can_enable?: boolean;
  approval_status: ApprovalStatus;
  approved_by?: string;
  approved_at?: string;
  created_by_user_id?: string;
  created_by_email?: string;
  description?: string;
  rejection_count?: number;
  version: number;
  created_at: string;
  updated_at: string;
}

export interface RateLimitApprovalWithAllocation {
  id: string;
  allocation_id: string;
  attempt_number: number;
  action: string;
  actor_user_id?: string;
  actor_email?: string;
  reason?: string;
  team_id: string;
  model_pattern: string;
  org_id: string;
  created_at: string;
}

export interface CreateRateLimitRequest {
  org_id?: string;
  team_id: string;
  model_pattern: string;
  token_limit?: number;
  token_unit?: TimeUnit;
  request_limit?: number;
  request_unit?: TimeUnit;
  burst_percentage?: number;
  enforcement?: Enforcement;
  enabled?: boolean;
  description?: string;
}

export interface UpdateRateLimitRequest {
  model_pattern?: string;
  token_limit?: number;
  token_unit?: TimeUnit;
  request_limit?: number;
  request_unit?: TimeUnit;
  burst_percentage?: number;
  enforcement?: Enforcement;
  enabled?: boolean;
  description?: string;
  version?: number;
}

export interface ListRateLimitsResponse {
  rate_limits: RateLimitAllocation[];
}

export type AlertType =
  | 'threshold_warning'
  | 'budget_exhausted'
  | 'forecast_overrun'
  | 'forecast_exhaustion';

export type AlertStatus = 'active' | 'resolved' | 'dismissed';

export interface BudgetAlert {
  id: string;
  budget_id: string;
  budget_name: string;
  entity_type: string;
  alert_type: AlertType;
  status: AlertStatus;
  triggered_usage_usd: number;
  budget_amount_usd: number;
  threshold_pct: number | null;
  projected_spend_usd: number | null;
  days_until_exhausted: number | null;
  period_start: string;
  created_at: string;
}

export interface AlertsResponse {
  alerts: BudgetAlert[];
  pagination: {
    total_count: number;
    page: number;
    page_size: number;
  };
}

export interface AlertCountResponse {
  active: number;
}
