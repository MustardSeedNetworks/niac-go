import { useCallback, useEffect, useMemo, useRef, useState } from 'react';

interface Options<T> {
  intervalMs?: number;
  transform?: (value: T) => T;
}

export function useApiResource<T>(
  fetcher: (signal?: AbortSignal) => Promise<T>,
  deps: unknown[] = [],
  options: Options<T> = {},
) {
  const { intervalMs, transform } = options;
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  const timerRef = useRef<number | null>(null);
  const fetcherRef = useRef<(signal?: AbortSignal) => Promise<T>>(fetcher);

  // Update fetcher ref when it changes
  useEffect(() => {
    fetcherRef.current = fetcher;
  }, [fetcher]);

  const run = useCallback(
    async (signal?: AbortSignal) => {
      try {
        const result = await fetcherRef.current(signal);
        // FIX #179: Don't update state if aborted
        if (signal?.aborted) return;
        setData(transform ? transform(result) : result);
        setError(null);
      } catch (err) {
        // FIX #179: Suppress AbortError from state updates
        if (err instanceof DOMException && err.name === 'AbortError') return;
        if (signal?.aborted) return;
        setError(err as Error);
      } finally {
        if (!signal?.aborted) {
          setLoading(false);
        }
      }
    },
    [transform],
  );

  useEffect(() => {
    // FIX #179: AbortController for cleanup on unmount/dependency change
    const controller = new AbortController();

    void run(controller.signal);

    if (intervalMs) {
      timerRef.current = window.setInterval(() => {
        void run(controller.signal);
      }, intervalMs);
    }

    return () => {
      controller.abort();
      if (timerRef.current) {
        clearInterval(timerRef.current);
        timerRef.current = null;
      }
    };
  }, [...deps, intervalMs, run]);

  const refetch = useCallback(() => run(), [run]);

  // Stable identity across renders so callers that pass this object as a
  // useMemo / useEffect dependency don't invalidate every render. Without
  // the wrapper a fresh `{ data, loading, error, refetch }` literal would
  // be allocated each render even when the underlying values were equal.
  return useMemo(
    () => ({ data, loading, error, refetch }) as const,
    [data, loading, error, refetch],
  );
}
