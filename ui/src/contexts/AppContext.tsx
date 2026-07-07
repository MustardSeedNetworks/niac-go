import { createContext, type ReactNode, useContext, useMemo } from 'react';
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
 * FEATURE #133: Centralized state management using React Context
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
  pollIntervals: typeof POLL_INTERVALS;
}

const AppContext = createContext<AppContextValue | null>(null);

export function AppProvider({ children }: { children: ReactNode }) {
  // Fetch shared data at the top level
  const stats = useApiResource(fetchStats, [], {
    intervalMs: POLL_INTERVALS.medium,
  });
  const devices = useApiResource(fetchDevices, [], {
    intervalMs: POLL_INTERVALS.slow,
  });
  const history = useApiResource(fetchHistory, [], {
    intervalMs: POLL_INTERVALS.slow,
  });
  const neighbors = useApiResource(fetchNeighbors, [], {
    intervalMs: POLL_INTERVALS.medium,
  });
  const version = useApiResource(fetchVersion, [], {
    intervalMs: POLL_INTERVALS.verySlow,
  });
  const errorTypes = useApiResource(fetchErrorTypes, []);
  const interfaces = useApiResource(fetchInterfaces, []);
  // Single poller for simulation run/stop state, shared by HeaderBar (status
  // chip), DashboardPage (status banner + stat cards), RuntimeControlPage
  // (start/stop controls), and PacketInspectorPage (active-interface
  // detection) via the useSimulationStatus() hook — see hooks/useSimulationStatus.ts.
  // Previously each of those pages ran its own independent
  // useApiResource(fetchSimulationStatus, …) poll against the same endpoint.
  const simStatus = useApiResource(fetchSimulationStatus, [], {
    intervalMs: POLL_INTERVALS.fast,
  });

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
      pollIntervals: POLL_INTERVALS,
    }),
    [stats, devices, history, neighbors, version, errorTypes, interfaces, simStatus],
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
