import { createContext, type ReactNode, useContext, useMemo, useState } from 'react';
import {
  fetchDevices,
  fetchErrorTypes,
  fetchHistory,
  fetchInterfaces,
  fetchNeighbors,
  fetchSimulationStatus,
  fetchStats,
  fetchVersion,
} from '../api/client';
import type {
  DeviceSummary,
  ErrorInjectionInfo,
  HistoryRecord,
  InterfacesResponse,
  NeighborRecord,
  SimulationStatus,
  StackStatsResponse,
  VersionInfo,
} from '../api/types';
import { POLL_INTERVALS } from '../constants/polling';
import { useApiResource } from '../hooks/useApiResource';

/**
 * Centralized state management using React Context.
 *
 * Provides shared application state to avoid prop drilling and
 * duplicate API calls. Memoized to prevent unnecessary re-renders.
 */

type ApiResourceResult<T> = {
  data: T | null;
  loading: boolean;
  error: Error | null;
  refetch: () => void;
};

interface AppContextValue {
  stats: ApiResourceResult<StackStatsResponse>;
  devices: ApiResourceResult<DeviceSummary[]>;
  history: ApiResourceResult<HistoryRecord[]>;
  neighbors: ApiResourceResult<NeighborRecord[]>;
  version: ApiResourceResult<VersionInfo>;
  errorTypes: ApiResourceResult<ErrorInjectionInfo>;
  interfaces: ApiResourceResult<InterfacesResponse>;
  simStatus: ApiResourceResult<SimulationStatus>;
  /**
   * Which scenario this browser is looking at. Selection lives here rather
   * than on the daemon: with several scenarios running, a server-side
   * "current" session means one tab switching scenario silently repoints
   * every other tab. Null until a scenario is running.
   */
  sessionId: string | null;
  setSessionId: (sessionId: string | null) => void;
  pollIntervals: typeof POLL_INTERVALS;
}

const AppContext = createContext<AppContextValue | null>(null);

export function AppProvider({ children }: { children: ReactNode }) {
  // Which scenario this browser is reading. Adopts whichever session the
  // daemon reports until the operator picks one, then this browser keeps its
  // own choice regardless of what other tabs do.
  const [pinnedSessionId, setPinnedSessionId] = useState<string | null>(null);

  const simStatus = useApiResource(fetchSimulationStatus, [], {
    intervalMs: POLL_INTERVALS.fast,
  });
  const runningSessionIds = useMemo(
    () => (simStatus.data?.sessions ?? []).map((session) => session.sessionId).filter(Boolean),
    [simStatus.data],
  );
  // Drop a pin that no longer names a running scenario, so a stopped session
  // does not leave every read 404ing.
  const sessionId =
    pinnedSessionId && runningSessionIds.includes(pinnedSessionId)
      ? pinnedSessionId
      : (simStatus.data?.sessionId ?? null);

  // Runtime reads take the session as a dependency: switching scenario has to
  // refetch, not keep showing the previous one's devices. With no scenario
  // running there is nothing to read, so these resolve empty rather than
  // requesting a session that does not exist.
  const stats = useApiResource(() => fetchStats(sessionId ?? ''), [sessionId], {
    intervalMs: POLL_INTERVALS.medium,
    enabled: sessionId !== null,
  });
  const devices = useApiResource(() => fetchDevices(sessionId ?? ''), [sessionId], {
    intervalMs: POLL_INTERVALS.slow,
    enabled: sessionId !== null,
  });
  const history = useApiResource(fetchHistory, [], {
    intervalMs: POLL_INTERVALS.slow,
  });
  const neighbors = useApiResource(() => fetchNeighbors(sessionId ?? ''), [sessionId], {
    intervalMs: POLL_INTERVALS.medium,
    enabled: sessionId !== null,
  });
  const version = useApiResource(fetchVersion, [], {
    intervalMs: POLL_INTERVALS.verySlow,
  });
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
      simStatus,
      sessionId,
      setSessionId: setPinnedSessionId,
      pollIntervals: POLL_INTERVALS,
    }),
    [stats, devices, history, neighbors, version, errorTypes, interfaces, simStatus, sessionId],
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
export function useAppState<K extends keyof AppContextValue>(key: K): AppContextValue[K] {
  const context = useAppContext();
  return context[key];
}
