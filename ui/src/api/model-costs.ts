import { apiClient } from './client';
import {
  ModelCost,
  ListModelCostsResponse,
  PaginatedResponse,
  CreateModelCostRequest,
  UpdateModelCostRequest,
} from './types';

export const modelCostsApi = {
  async list(
    page = 1,
    pageSize = 30,
    provider = '',
    sortBy = '',
    sortDir = 'asc'
  ): Promise<PaginatedResponse<ModelCost>> {
    const params = new URLSearchParams({ page: String(page), page_size: String(pageSize) });
    if (provider) params.set('provider', provider);
    if (sortBy) {
      params.set('sort_by', sortBy);
      params.set('sort_dir', sortDir);
    }
    const response = await apiClient.get<PaginatedResponse<ModelCost> | ListModelCostsResponse>(
      `/model-costs?${params}`
    );
    if ('data' in response && 'pagination' in response) {
      return response as PaginatedResponse<ModelCost>;
    }
    const legacyResponse = response as ListModelCostsResponse;
    const costs = legacyResponse.model_costs || [];
    return {
      data: costs,
      pagination: {
        page: 1,
        page_size: costs.length,
        total_count: costs.length,
        total_pages: 1,
      },
    };
  },

  async providers(): Promise<string[]> {
    const response = await apiClient.get<{ providers: string[] }>('/model-costs/providers');
    return response.providers ?? [];
  },

  async get(modelId: string): Promise<ModelCost> {
    return apiClient.get<ModelCost>(`/model-costs/${encodeURIComponent(modelId)}`);
  },

  async create(data: CreateModelCostRequest): Promise<ModelCost> {
    return apiClient.post<ModelCost, CreateModelCostRequest>('/model-costs', data);
  },

  async update(modelId: string, data: UpdateModelCostRequest): Promise<ModelCost> {
    return apiClient.put<ModelCost, UpdateModelCostRequest>(
      `/model-costs/${encodeURIComponent(modelId)}`,
      data
    );
  },

  async delete(modelId: string): Promise<void> {
    return apiClient.delete(`/model-costs/${encodeURIComponent(modelId)}`);
  },
};
