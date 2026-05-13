import { apiClient } from './client';
import {
  BudgetDefinition,
  BudgetForecast,
  ListBudgetsResponse,
  PaginatedResponse,
  CreateBudgetRequest,
  UpdateBudgetRequest,
  UsageRecord,
  UsageHistoryResponse,
  ValidateCELResponse,
} from './types';

export const budgetsApi = {
  async list(
    page = 1,
    pageSize = 30,
    options?: { enabledOnly?: boolean }
  ): Promise<PaginatedResponse<BudgetDefinition>> {
    const params = new URLSearchParams();
    params.set('page', page.toString());
    params.set('page_size', pageSize.toString());
    if (options?.enabledOnly) {
      params.set('enabled_only', 'true');
    }
    const response = await apiClient.get<PaginatedResponse<BudgetDefinition> | ListBudgetsResponse>(
      `/budgets?${params.toString()}`
    );
    if ('data' in response && 'pagination' in response) {
      return response as PaginatedResponse<BudgetDefinition>;
    }
    const legacyResponse = response as ListBudgetsResponse;
    const budgets = legacyResponse.budgets || [];
    return {
      data: budgets,
      pagination: {
        page: 1,
        page_size: budgets.length,
        total_count: budgets.length,
        total_pages: 1,
      },
    };
  },

  async get(id: string): Promise<BudgetDefinition> {
    return apiClient.get<BudgetDefinition>(`/budgets/${id}`);
  },

  async create(data: CreateBudgetRequest): Promise<BudgetDefinition> {
    return apiClient.post<BudgetDefinition, CreateBudgetRequest>('/budgets', data);
  },

  async update(id: string, data: UpdateBudgetRequest): Promise<BudgetDefinition> {
    return apiClient.put<BudgetDefinition, UpdateBudgetRequest>(`/budgets/${id}`, data);
  },

  async delete(id: string, options?: { cascade?: boolean }): Promise<void> {
    const params = options?.cascade ? '?cascade=true' : '';
    return apiClient.delete(`/budgets/${id}${params}`);
  },

  async getUsage(id: string, since?: Date, limit?: number): Promise<UsageRecord[]> {
    const params = new URLSearchParams();
    if (since) {
      params.set('since', since.toISOString());
    }
    if (limit) {
      params.set('limit', limit.toString());
    }
    const query = params.toString();
    const path = `/budgets/${id}/usage${query ? `?${query}` : ''}`;
    const response = await apiClient.get<{ data: UsageRecord[] } | UsageHistoryResponse>(path);
    if ('data' in response) return response.data || [];
    return (response as UsageHistoryResponse).usage_records || [];
  },

  async reset(id: string): Promise<void> {
    await apiClient.post(`/budgets/${id}/reset`);
  },

  async validateCEL(expression: string): Promise<ValidateCELResponse> {
    return apiClient.post<ValidateCELResponse, { expression: string }>('/validate-cel', {
      expression,
    });
  },

  async listParentCandidates(): Promise<{ id: string; name: string }[]> {
    const response = await apiClient.get<{ id: string; name: string }[]>(
      '/budgets/parent-candidates'
    );
    return response || [];
  },

  async getChildren(id: string): Promise<BudgetDefinition[]> {
    const response = await apiClient.get<{ data: BudgetDefinition[] }>(`/budgets/${id}/children`);
    return response.data;
  },

  async getForecast(id: string): Promise<BudgetForecast> {
    return apiClient.get<BudgetForecast>(`/budgets/${id}/forecast`);
  },

  async listForecasts(): Promise<BudgetForecast[]> {
    const response = await apiClient.get<{ data: BudgetForecast[]; count: number }>(
      '/budgets/forecast'
    );
    return response.data || [];
  },
};
