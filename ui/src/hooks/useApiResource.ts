import { useCallback, useEffect, useRef, useState } from 'react';

interface Options<T> {
  intervalMs?: number;
  transform?: (value: T) => T;
}

// FIX #306: Race condition fix with cancellation ref checked inside run()
export function useApiResource<T>(
  fetcher: () => Promise<T>,
  deps: unknown[] = [],
  options: Options<T> = {},
) {
  const { intervalMs, transform } = options;
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  const timerRef = useRef<number | null>(null);
  const fetcherRef = useRef<() => Promise<T>>(fetcher);
  const cancelledRef = useRef(false);

  // Update fetcher ref when it changes
  useEffect(() => {
    fetcherRef.current = fetcher;
  }, [fetcher]);

  const run = useCallback(async () => {
    try {
      const result = await fetcherRef.current();
      if (cancelledRef.current) return;
      setData(transform ? transform(result) : result);
      setError(null);
    } catch (err) {
      if (cancelledRef.current) return;
      setError(err as Error);
    } finally {
      if (!cancelledRef.current) {
        setLoading(false);
      }
    }
  }, [transform]);

  useEffect(() => {
    cancelledRef.current = false;

    void run();

    if (intervalMs) {
      timerRef.current = window.setInterval(() => {
        if (!cancelledRef.current) {
          void run();
        }
      }, intervalMs);
    }

    return () => {
      cancelledRef.current = true;
      if (timerRef.current) {
        clearInterval(timerRef.current);
        timerRef.current = null;
      }
    };
  }, [...deps, intervalMs, run]);

  return { data, loading, error, refetch: run } as const;
}
