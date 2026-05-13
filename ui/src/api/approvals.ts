import { apiClient } from './client';
import type { PaginatedResponse, ApprovalWithBudget } from './types';

export const approvalsApi = {
  listPending: (page = 1, pageSize = 30) =>
    apiClient.get<PaginatedResponse<ApprovalWithBudget>>(
      `/approvals?page=${page}&page_size=${pageSize}`
    ),

  count: () => apiClient.get<{ count: number }>('/approvals/count'),

  approve: (budgetId: string) =>
    apiClient.post<{ message: string }>(`/approvals/${budgetId}/approve`, {}),

  reject: (budgetId: string, reason: string) =>
    apiClient.post<{ message: string }>(`/approvals/${budgetId}/reject`, { reason }),

  resubmit: (budgetId: string) =>
    apiClient.post<{ message: string }>(`/approvals/${budgetId}/resubmit`, {}),

  history: (page = 1, pageSize = 30) =>
    apiClient.get<PaginatedResponse<ApprovalWithBudget>>(
      `/approvals/history?page=${page}&page_size=${pageSize}`
    ),
};
