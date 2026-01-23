import {
  fetchConfig,
  fetchDevices,
  fetchNeighbors,
  fetchReplayStatus,
  fetchRuntimeStatus,
  fetchStats,
  fetchTopology,
} from '../api/client';

type PrefetchFn = () => Promise<unknown>;

const ROUTE_PREFETCH_MAP: Record<string, PrefetchFn[]> = {
  '/': [fetchStats],
  '/runtime': [fetchRuntimeStatus],
  '/devices': [fetchDevices],
  '/device-config': [fetchDevices, fetchConfig],
  '/topology': [fetchTopology, fetchNeighbors],
  '/analysis': [fetchReplayStatus],
};

const prefetched = new Set<string>();

/**
 * Prefetch API data for a route during browser idle time.
 * Each route is only prefetched once per session to avoid redundant requests.
 */
export function prefetchRoute(path: string): void {
  if (prefetched.has(path)) return;

  const fetchers = ROUTE_PREFETCH_MAP[path];
  if (!fetchers) return;

  prefetched.add(path);

  const run = () => {
    for (const fn of fetchers) {
      fn().catch(() => {
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
