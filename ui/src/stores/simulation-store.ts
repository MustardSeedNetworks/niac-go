import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import { immer } from 'zustand/middleware/immer';
import type { SimulationStatus, StackStatsResponse } from '../types';

/**
 * Simulation Store State
 *
 * Manages simulation runtime state and statistics.
 */
export interface SimulationStoreState {
  // State
  status: SimulationStatus | null;
  stats: StackStatsResponse | null;
  isConnected: boolean;
  lastError: string | null;

  // Computed
  isRunning: () => boolean;
  uptime: () => number;

  // Actions
  setStatus: (status: SimulationStatus | null) => void;
  setStats: (stats: StackStatsResponse | null) => void;
  setConnected: (connected: boolean) => void;
  setError: (error: string | null) => void;
  reset: () => void;
}

export const useSimulationStore = create<SimulationStoreState>()(
  devtools(
    immer((set, get) => ({
      // Initial state
      status: null,
      stats: null,
      isConnected: false,
      lastError: null,

      // Computed
      isRunning: () => Boolean(get().status?.running),

      uptime: () => get().status?.uptimeSeconds ?? 0,

      // Actions
      setStatus: (status) =>
        set((state) => {
          state.status = status;
          if (status) {
            state.lastError = null;
          }
        }),

      setStats: (stats) =>
        set((state) => {
          state.stats = stats;
        }),

      setConnected: (connected) =>
        set((state) => {
          state.isConnected = connected;
        }),

      setError: (error) =>
        set((state) => {
          state.lastError = error;
        }),

      reset: () =>
        set((state) => {
          state.status = null;
          state.stats = null;
          state.isConnected = false;
          state.lastError = null;
        }),
    })),
    { name: 'SimulationStore' },
  ),
);
