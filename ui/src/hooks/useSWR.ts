import useSWR, { SWRConfiguration, mutate as globalMutate } from 'swr';
import { useState, useCallback } from 'react';

// Default SWR config for the app
export const swrConfig: SWRConfiguration = {
  revalidateOnFocus: true,
  revalidateOnReconnect: true,
  dedupingInterval: 2000,
  errorRetryCount: 3,
};

// Hook for data fetching with SWR
interface UseSWRApiOptions extends SWRConfiguration {
  // Don't fetch on mount
  skip?: boolean;
}

export function useSWRApi<T>(
  key: string | null,
  fetcher: () => Promise<T>,
  options: UseSWRApiOptions = {}
) {
  const { skip, ...swrOptions } = options;

  const { data, error, isLoading, isValidating, mutate } = useSWR<T>(skip ? null : key, fetcher, {
    ...swrConfig,
    ...swrOptions,
  });

  const refresh = useCallback(() => {
    return mutate();
  }, [mutate]);

  return {
    data: data ?? null,
    loading: isLoading,
    isRefetching: isValidating && !isLoading,
    error: error ?? null,
    refresh,
    mutate,
  };
}

// Hook for paginated data with SWR
interface UseSWRPaginatedOptions extends UseSWRApiOptions {
  initialPage?: number;
  pageSize?: number;
}

export function useSWRPaginated<T>(
  keyPrefix: string,
  fetcher: (
    page: number,
    pageSize: number
  ) => Promise<{ data: T[]; pagination: { total_count: number; total_pages: number } }>,
  options: UseSWRPaginatedOptions = {}
) {
  const { initialPage = 1, pageSize = 30, skip, ...swrOptions } = options;
  const [page, setPage] = useState(initialPage);

  const key = skip ? null : `${keyPrefix}?page=${page}&pageSize=${pageSize}`;

  const { data, error, isLoading, isValidating, mutate } = useSWR(
    key,
    () => fetcher(page, pageSize),
    {
      ...swrConfig,
      ...swrOptions,
    }
  );

  const refresh = useCallback(() => {
    return mutate();
  }, [mutate]);

  const goToPage = useCallback((newPage: number) => {
    setPage(newPage);
  }, []);

  return {
    data: data?.data ?? null,
    pagination: data?.pagination ?? null,
    loading: isLoading,
    isRefetching: isValidating && !isLoading,
    error: error ?? null,
    page,
    setPage: goToPage,
    refresh,
    mutate,
  };
}

// Mutation hook (unchanged - mutations don't need SWR)
export function useMutation<T, A extends unknown[]>(mutator: (...args: A) => Promise<T>) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const execute = useCallback(
    async (...args: A): Promise<T> => {
      setLoading(true);
      setError(null);
      try {
        const result = await mutator(...args);
        return result;
      } catch (err) {
        const error = err instanceof Error ? err : new Error(String(err));
        setError(error);
        throw error;
      } finally {
        setLoading(false);
      }
    },
    [mutator]
  );

  return {
    execute,
    loading,
    error,
  };
}

// Helper to invalidate/refetch specific keys
export function invalidateKey(key: string) {
  return globalMutate(key);
}

// Helper to invalidate all keys matching a prefix
export function invalidatePrefix(prefix: string) {
  return globalMutate(key => typeof key === 'string' && key.startsWith(prefix), undefined, {
    revalidate: true,
  });
}

// Cache keys for the app
export const CacheKeys = {
  budgets: 'budgets',
  budgetDetail: (id: string) => `budget:${id}`,
  budgetUsage: (id: string) => `budget:${id}:usage`,
  rateLimits: 'rate-limits',
  rateLimitDetail: (id: string) => `rate-limit:${id}`,
  approvals: 'approvals',
  approvalCount: 'approvals:count',
  rateLimitApprovalCount: 'rate-limits:pending:count',
  auditLogs: 'audit-logs',
  modelCosts: 'model-costs',
  parentCandidates: 'budgets:parent-candidates',
  sidebarStats: 'sidebar:stats',
  budgetForecast: (id: string) => `budget:${id}:forecast`,
  budgetForecasts: 'budgets:forecasts',
  alerts: 'alerts',
  alertCount: 'alerts:count',
} as const;
