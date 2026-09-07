import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useUIStore } from '../stores/ui-store';
import { getErrorMessage } from '../utils/format';

interface Options<T> {
  intervalMs?: number;
  transform?: (value: T) => T;
  /**
   * Surface fetch/refetch failures as a global error toast (see
   * `ui/ToastContainer`). Opt-in per call site — most callers keep this
   * unset: background pollers that expect to be offline sometimes (e.g.
   * the daemon isn't running yet) should rely on their own page-level
   * indicator instead of toasting on every poll tick.
   *
   * Only fires once per distinct error message (dedup via a ref), and
   * re-fires if the resource recovers and then fails again with a new
   * message.
   */
  errorToast?: boolean | { title?: string };
  /**
   * Skip fetching entirely while false, leaving `data` null and `loading`
   * false. For a resource that needs something the caller does not have yet —
   * a session-scoped read before any scenario is running. Requesting anyway
   * would ask for a session that does not exist, and substituting a
   * placeholder would report numbers that were never measured.
   */
  enabled?: boolean;
}

export function useApiResource<T>(
  fetcher: (signal?: AbortSignal) => Promise<T>,
  deps: unknown[] = [],
  options: Options<T> = {},
) {
  const { intervalMs, transform, errorToast, enabled = true } = options;
  const [data, setData] = useState<T | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);
  const timerRef = useRef<number | null>(null);
  const fetcherRef = useRef<(signal?: AbortSignal) => Promise<T>>(fetcher);
  // transform and errorToast are held the same way the fetcher is, and for a
  // sharper reason than tidiness. Both are declared inline at their call
  // sites -- an errorToast whose title is a translated string is a fresh
  // object on every render -- so naming them as dependencies of `run`
  // made `run` a new
  // function on every render, which made the effect below re-run on every
  // render, which fetched, which set state, which rendered. Five call sites
  // sat in an unbounded fetch loop against the daemon (Alerts, Devices,
  // Topology, Packet Inspector and its capture starter); the page-story
  // harness measured 39,191 requests in ten seconds on Alerts. Holding them
  // in refs keeps `run` stable while still using the latest value.
  const transformRef = useRef(transform);
  const errorToastRef = useRef(errorToast);
  const lastToastedMessageRef = useRef<string | null>(null);
  const addNotification = useUIStore((s) => s.addNotification);
  const { t } = useTranslation('common');

  // Update fetcher ref when it changes
  useEffect(() => {
    fetcherRef.current = fetcher;
  }, [fetcher]);

  useEffect(() => {
    transformRef.current = transform;
    errorToastRef.current = errorToast;
  }, [transform, errorToast]);

  const run = useCallback(
    async (signal?: AbortSignal) => {
      try {
        const result = await fetcherRef.current(signal);
        // FIX #179: Don't update state if aborted
        if (signal?.aborted) return;
        const apply = transformRef.current;
        setData(apply ? apply(result) : result);
        setError(null);
        lastToastedMessageRef.current = null;
      } catch (err) {
        // FIX #179: Suppress AbortError from state updates
        if (err instanceof DOMException && err.name === 'AbortError') return;
        if (signal?.aborted) return;
        setError(err as Error);
        const toast = errorToastRef.current;
        if (toast) {
          const message = getErrorMessage(err);
          if (lastToastedMessageRef.current !== message) {
            lastToastedMessageRef.current = message;
            const title =
              (typeof toast === 'object' ? toast.title : undefined) ??
              t('toast.requestFailedTitle');
            addNotification({ type: 'error', title, message });
          }
        }
      } finally {
        if (!signal?.aborted) {
          setLoading(false);
        }
      }
    },
    [addNotification, t],
  );

  useEffect(() => {
    if (!enabled) {
      // Nothing to fetch yet. Leave data null rather than showing a stale or
      // invented value, and stop reporting a load that will never happen.
      setLoading(false);
      return;
    }
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
  }, [...deps, intervalMs, run, enabled]);

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
