/**
 * useSimulationStatus.test.tsx — locks the shared-poller contract (PR
 * "Dashboard honesty + shell sim-status").
 *
 * Before this hook, DashboardPage, RuntimeControlPage, and
 * PacketInspectorPage each ran their own independent
 * `useApiResource(fetchSimulationStatus, …)` poll. useSimulationStatus()
 * is a thin alias over the single AppProvider-owned poll, so multiple
 * consumers mounted under one AppProvider must share ONE underlying
 * `GET /api/v1/simulation` call per tick, not one per consumer.
 */
import { render, renderHook, screen, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { SimulationStatus } from '../api/types';
import { AppProvider } from '../contexts/AppContext';
import { useSimulationStatus } from './useSimulationStatus';

const fetchSimulationStatus = vi.fn<() => Promise<SimulationStatus>>();
const fetchStats = vi.fn();
const fetchDevices = vi.fn();
const fetchHistory = vi.fn();
const fetchNeighbors = vi.fn();
const fetchVersion = vi.fn();
const fetchErrorTypes = vi.fn();
const fetchInterfaces = vi.fn();

vi.mock('../api/client', () => ({
  fetchSimulationStatus: () => fetchSimulationStatus(),
  fetchStats: () => fetchStats(),
  fetchDevices: () => fetchDevices(),
  fetchHistory: () => fetchHistory(),
  fetchNeighbors: () => fetchNeighbors(),
  fetchVersion: () => fetchVersion(),
  fetchErrorTypes: () => fetchErrorTypes(),
  fetchInterfaces: () => fetchInterfaces(),
}));

const running: SimulationStatus = {
  running: true,
  interface: 'eth0',
  deviceCount: 3,
  uptimeSeconds: 42,
};

const wrapper = ({ children }: { children: ReactNode }) => <AppProvider>{children}</AppProvider>;

describe('useSimulationStatus', () => {
  beforeEach(() => {
    fetchSimulationStatus.mockReset().mockResolvedValue(running);
    fetchStats.mockReset().mockResolvedValue(null);
    fetchDevices.mockReset().mockResolvedValue([]);
    fetchHistory.mockReset().mockResolvedValue([]);
    fetchNeighbors.mockReset().mockResolvedValue([]);
    fetchVersion.mockReset().mockResolvedValue({ version: '0.0.0' });
    fetchErrorTypes.mockReset().mockResolvedValue({ availableTypes: [], info: '' });
    fetchInterfaces.mockReset().mockResolvedValue({ interfaces: [] });
  });

  it('exposes the AppProvider-fetched simulation status', async () => {
    const { result } = renderHook(() => useSimulationStatus(), { wrapper });
    await waitFor(() => expect(result.current.data).toEqual(running));
  });

  it('one AppProvider serves multiple useSimulationStatus() consumers with a single fetch', async () => {
    function ConsumerA() {
      const { data } = useSimulationStatus();
      return <span data-testid="a">{data?.running ? 'running' : 'stopped'}</span>;
    }
    function ConsumerB() {
      const { data } = useSimulationStatus();
      return <span data-testid="b">{data?.deviceCount ?? '—'}</span>;
    }
    function ConsumerC() {
      const { data } = useSimulationStatus();
      return <span data-testid="c">{data?.interface ?? '—'}</span>;
    }

    render(
      <AppProvider>
        <ConsumerA />
        <ConsumerB />
        <ConsumerC />
      </AppProvider>,
    );

    await waitFor(() => expect(screen.getByTestId('a')).toHaveTextContent('running'));
    expect(screen.getByTestId('b')).toHaveTextContent('3');
    expect(screen.getByTestId('c')).toHaveTextContent('eth0');

    // The single AppProvider poll serves all three consumers — one fetch,
    // not three.
    expect(fetchSimulationStatus).toHaveBeenCalledTimes(1);
  });
});
