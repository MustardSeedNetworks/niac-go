import { useEffect, useRef, useState } from 'react';

interface Options<T> {
  intervalMs?: number;
  transform?: (value: T) => T;
}

export function useApiResource<T>(fetcher: () => Promise<T>, deps: unknown[] = [], options: Options<T> = {}) {
  const { intervalMs, transform } = options;
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  const timerRef = useRef<number | null>(null);
  const fetcherRef = useRef(fetcher);

  // Update fetcher ref when it changes
  useEffect(() => {
    fetcherRef.current = fetcher;
  }, [fetcher]);

  const run = async () => {
    try {
      const result = await fetcherRef.current();
      setData(transform ? transform(result) : result);
      setError(null);
    } catch (err) {
      setError(err as Error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    let cancelled = false;

    const runWithCancellation = async () => {
      if (cancelled) return;
      await run();
    };

    runWithCancellation();

    if (intervalMs) {
      timerRef.current = window.setInterval(run, intervalMs);
    }

    return () => {
      cancelled = true;
      if (timerRef.current) {
        clearInterval(timerRef.current);
        timerRef.current = null;
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);

  return { data, loading, error, refetch: run } as const;
}
