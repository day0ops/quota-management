import { apiClient } from './client';
import type { PaginatedResponse, AuditLogEntry } from './types';

export interface AuditFilters {
  entity_type?: string;
  action?: string;
  actor?: string;
  from?: string;
  to?: string;
}

export const auditApi = {
  list: (page = 1, pageSize = 30, filters: AuditFilters = {}) => {
    const params = new URLSearchParams({ page: String(page), page_size: String(pageSize) });
    Object.entries(filters).forEach(([key, value]) => {
      if (value) params.set(key, value);
    });
    return apiClient.get<PaginatedResponse<AuditLogEntry>>(`/audit?${params}`);
  },
};
