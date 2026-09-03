import {
  fetchConfig,
  fetchDevices,
  fetchNeighbors,
  fetchReplayStatus,
  fetchStats,
  fetchTopology,
} from '../api/client';

type PrefetchFn = () => Promise<unknown>;
/** Runtime reads need a session, so they can only be warmed once one is running. */
type SessionPrefetchFn = (sessionId: string) => Promise<unknown>;

const ROUTE_PREFETCH_MAP: Record<string, PrefetchFn[]> = {
  '/device-config': [fetchConfig],
  '/analysis': [fetchReplayStatus],
};

const SESSION_PREFETCH_MAP: Record<string, SessionPrefetchFn[]> = {
  '/': [fetchStats],
  '/devices': [fetchDevices],
  '/device-config': [fetchDevices],
  '/topology': [fetchTopology, fetchNeighbors],
};

const prefetched = new Set<string>();

/**
 * Prefetch API data for a route during browser idle time.
 * Each route is only prefetched once per session to avoid redundant requests.
 */
export function prefetchRoute(path: string, sessionId?: string | null): void {
  if (prefetched.has(path)) return;

  const fetchers = ROUTE_PREFETCH_MAP[path] ?? [];
  const sessionFetchers = sessionId ? (SESSION_PREFETCH_MAP[path] ?? []) : [];
  if (fetchers.length === 0 && sessionFetchers.length === 0) return;

  prefetched.add(path);

  const run = () => {
    for (const fn of fetchers) {
      fn().catch(() => {
        // Silently ignore prefetch failures
      });
    }
    for (const fn of sessionFetchers) {
      fn(sessionId as string).catch(() => {
        // Silently ignore prefetch failures
      });
    }
  };

  if ('requestIdleCallback' in window) {
    window.requestIdleCallback(run, { timeout: 2000 });
  } else {
    setTimeout(run, 100);
  }
}
