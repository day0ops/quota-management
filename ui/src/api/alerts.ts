import { useSWRApi, CacheKeys } from '../hooks/useSWR';
import { AlertsResponse, AlertCountResponse } from './types';

const BASE = '/api/v1/alerts';

export interface AlertListParams {
  status?: 'active' | 'resolved' | 'dismissed' | 'all';
  alert_type?: string;
  budget_id?: string;
  page?: number;
  page_size?: number;
}

export const alertsApi = {
  async list(params: AlertListParams = {}): Promise<AlertsResponse> {
    const qs = new URLSearchParams();
    if (params.status) qs.set('status', params.status);
    if (params.alert_type) qs.set('alert_type', params.alert_type);
    if (params.budget_id) qs.set('budget_id', params.budget_id);
    if (params.page) qs.set('page', String(params.page));
    if (params.page_size) qs.set('page_size', String(params.page_size));
    const res = await fetch(`${BASE}?${qs}`);
    if (!res.ok) throw new Error('Failed to fetch alerts');
    return res.json();
  },

  async count(): Promise<AlertCountResponse> {
    const res = await fetch(`${BASE}/count`);
    if (!res.ok) throw new Error('Failed to fetch alert count');
    return res.json();
  },

  async dismiss(id: string): Promise<void> {
    const res = await fetch(`${BASE}/${id}/dismiss`, { method: 'PUT' });
    if (!res.ok) throw new Error('Failed to dismiss alert');
  },

  async dismissAll(): Promise<{ dismissed: number }> {
    const res = await fetch(`${BASE}/dismiss-all`, { method: 'POST' });
    if (!res.ok) throw new Error('Failed to dismiss all alerts');
    return res.json();
  },
};

export function useAlerts(params: AlertListParams = {}) {
  const statusKey = params.status ?? 'active';
  return useSWRApi<AlertsResponse>(
    `${CacheKeys.alerts}:${statusKey}`,
    () => alertsApi.list(params),
    { refreshInterval: 15000 }
  );
}

export function useAlertCount() {
  return useSWRApi<AlertCountResponse>(CacheKeys.alertCount, () => alertsApi.count(), {
    refreshInterval: 15000,
  });
}
