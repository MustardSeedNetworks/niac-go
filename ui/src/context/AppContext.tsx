import { createContext, useContext, useMemo, type ReactNode } from 'react';
import { useApiResource } from '../hooks/useApiResource';
import {
  fetchStats,
  fetchDevices,
  fetchHistory,
  fetchNeighbors,
  fetchVersion,
  fetchErrorTypes,
  fetchInterfaces,
} from '../api/client';

/**
 * FEATURE #133: Centralized state management using React Context
 *
 * Provides shared application state to avoid prop drilling and
 * duplicate API calls. Memoized to prevent unnecessary re-renders.
 */

// Polling intervals (in milliseconds)
const POLL_INTERVALS = {
  FAST: 2000,      // 2s - Real-time simulation status
  MEDIUM: 5000,    // 5s - Live stats
  SLOW: 15000,     // 15s - Historical data
  VERY_SLOW: 60000, // 1m - Static data like version
} as const;

interface AppContextValue {
  stats: ReturnType<typeof useApiResource>;
  devices: ReturnType<typeof useApiResource>;
  history: ReturnType<typeof useApiResource>;
  neighbors: ReturnType<typeof useApiResource>;
  version: ReturnType<typeof useApiResource>;
  errorTypes: ReturnType<typeof useApiResource>;
  interfaces: ReturnType<typeof useApiResource>;
  pollIntervals: typeof POLL_INTERVALS;
}

const AppContext = createContext<AppContextValue | null>(null);

export function AppProvider({ children }: { children: ReactNode }) {
  // Fetch shared data at the top level
  const stats = useApiResource(fetchStats, [], { intervalMs: POLL_INTERVALS.MEDIUM });
  const devices = useApiResource(fetchDevices, [], { intervalMs: POLL_INTERVALS.SLOW });
  const history = useApiResource(fetchHistory, [], { intervalMs: POLL_INTERVALS.SLOW });
  const neighbors = useApiResource(fetchNeighbors, [], { intervalMs: POLL_INTERVALS.MEDIUM });
  const version = useApiResource(fetchVersion, [], { intervalMs: POLL_INTERVALS.VERY_SLOW });
  const errorTypes = useApiResource(fetchErrorTypes, []);
  const interfaces = useApiResource(fetchInterfaces, []);

  // Memoize context value to prevent unnecessary re-renders
  const value = useMemo(
    () => ({
      stats,
      devices,
      history,
      neighbors,
      version,
      errorTypes,
      interfaces,
      pollIntervals: POLL_INTERVALS,
    }),
    [stats, devices, history, neighbors, version, errorTypes, interfaces]
  );

  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
}

/**
 * Hook to access shared application state
 *
 * @throws Error if used outside AppProvider
 */
export function useAppContext() {
  const context = useContext(AppContext);
  if (!context) {
    throw new Error('useAppContext must be used within AppProvider');
  }
  return context;
}

/**
 * Hook to access specific slice of state
 *
 * Prevents components from re-rendering when unrelated state changes.
 */
export function useAppState<K extends keyof AppContextValue>(
  key: K
): AppContextValue[K] {
  const context = useAppContext();
  return context[key];
}
