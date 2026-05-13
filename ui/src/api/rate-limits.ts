import { apiClient } from './client';
import {
  RateLimitAllocation,
  ListRateLimitsResponse,
  PaginatedResponse,
  CreateRateLimitRequest,
  UpdateRateLimitRequest,
  RateLimitApprovalWithAllocation,
} from './types';

export const rateLimitsApi = {
  async list(
    page = 1,
    pageSize = 30,
    options?: { enabledOnly?: boolean }
  ): Promise<PaginatedResponse<RateLimitAllocation>> {
    const params = new URLSearchParams();
    params.set('page', page.toString());
    params.set('page_size', pageSize.toString());
    if (options?.enabledOnly) {
      params.set('enabled_only', 'true');
    }
    const response = await apiClient.get<
      PaginatedResponse<RateLimitAllocation> | ListRateLimitsResponse | RateLimitAllocation[]
    >(`/rate-limits?${params.toString()}`);
    // Handle paginated response
    if ('data' in response && 'pagination' in response) {
      return response as PaginatedResponse<RateLimitAllocation>;
    }
    // Handle plain array response from backend
    if (Array.isArray(response)) {
      return {
        data: response,
        pagination: {
          page: 1,
          page_size: response.length,
          total_count: response.length,
          total_pages: 1,
        },
      };
    }
    // Handle legacy { rate_limits: [...] } response
    const legacyResponse = response as ListRateLimitsResponse;
    const rateLimits = legacyResponse.rate_limits || [];
    return {
      data: rateLimits,
      pagination: {
        page: 1,
        page_size: rateLimits.length,
        total_count: rateLimits.length,
        total_pages: 1,
      },
    };
  },

  async get(id: string): Promise<RateLimitAllocation> {
    return apiClient.get<RateLimitAllocation>(`/rate-limits/${id}`);
  },

  async create(data: CreateRateLimitRequest): Promise<RateLimitAllocation> {
    return apiClient.post<RateLimitAllocation, CreateRateLimitRequest>('/rate-limits', data);
  },

  async update(id: string, data: UpdateRateLimitRequest): Promise<RateLimitAllocation> {
    return apiClient.put<RateLimitAllocation, UpdateRateLimitRequest>(`/rate-limits/${id}`, data);
  },

  async delete(id: string): Promise<void> {
    return apiClient.delete(`/rate-limits/${id}`);
  },

  async approve(id: string): Promise<RateLimitAllocation> {
    return apiClient.post<RateLimitAllocation>(`/rate-limits/${id}/approve`);
  },

  async reject(id: string, reason?: string): Promise<RateLimitAllocation> {
    return apiClient.post<RateLimitAllocation, { reason?: string }>(`/rate-limits/${id}/reject`, {
      reason,
    });
  },

  async listPending(): Promise<RateLimitAllocation[]> {
    const response = await apiClient.get<RateLimitAllocation[]>('/rate-limits/pending');
    return Array.isArray(response) ? response : [];
  },

  async count(): Promise<{ count: number }> {
    return apiClient.get<{ count: number }>('/rate-limits/pending/count');
  },

  async history(
    page = 1,
    pageSize = 30
  ): Promise<PaginatedResponse<RateLimitApprovalWithAllocation>> {
    return apiClient.get<PaginatedResponse<RateLimitApprovalWithAllocation>>(
      `/rate-limits/approvals/history?page=${page}&page_size=${pageSize}`
    );
  },

  async resubmit(id: string): Promise<RateLimitAllocation> {
    return apiClient.post<RateLimitAllocation>(`/rate-limits/${id}/resubmit`);
  },
};
