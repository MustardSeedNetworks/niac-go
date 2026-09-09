import { useQuery } from '@tanstack/react-query';
import { useCallback, useMemo } from 'react';
import { useResourceError } from './useResourceError';

export interface ResourceOptions<T> {
  intervalMs?: number;
  transform?: (value: T) => T;
  errorToast?: boolean | { title?: string };
  enabled?: boolean;
}

export function useApiResource<T>(
  fetcher: (signal?: AbortSignal) => Promise<T>,
  queryKey: readonly [string, ...unknown[]],
  options: ResourceOptions<T> = {},
) {
  const { intervalMs, transform, errorToast, enabled = true } = options;
  const query = useQuery({
    queryKey,
    queryFn: ({ signal }) => fetcher(signal),
    // Internet connectivity does not determine whether a local daemon is reachable.
    networkMode: 'always',
    enabled,
    select: transform,
    refetchInterval: intervalMs,
    retry: false,
    refetchOnWindowFocus: false,
  });
  useResourceError(
    enabled && errorToast ? query.error : null,
    typeof errorToast === 'object' ? errorToast.title : undefined,
  );

  const data = enabled ? (query.data ?? null) : null;
  const error = enabled ? query.error : null;
  const loading = enabled && query.isLoading;
  const queryRefetch = query.refetch;
  const refetch = useCallback(async () => {
    await queryRefetch();
  }, [queryRefetch]);
  return useMemo(() => ({ data, loading, error, refetch }), [data, loading, error, refetch]);
}
