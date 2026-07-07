/**
 * HeaderBar.test.tsx — locks the shell sim-status chip (PR "Dashboard
 * honesty + shell sim-status"). The chip sits next to ConnectionStatus
 * and reflects the shared useSimulationStatus() poll — it must not spin
 * up its own independent fetch against /api/v1/simulation.
 */
import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { SimulationStatus } from '../api/types';
import { AppProvider } from '../contexts/AppContext';
import '../i18n';
import { HeaderBar } from './HeaderBar';

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

beforeEach(() => {
  fetchStats.mockReset().mockResolvedValue(null);
  fetchDevices.mockReset().mockResolvedValue([]);
  fetchHistory.mockReset().mockResolvedValue([]);
  fetchNeighbors.mockReset().mockResolvedValue([]);
  fetchVersion.mockReset().mockResolvedValue({ version: '0.0.0' });
  fetchErrorTypes.mockReset().mockResolvedValue({ availableTypes: [], info: '' });
  fetchInterfaces.mockReset().mockResolvedValue({ interfaces: [] });
  fetchSimulationStatus.mockReset();
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true } as Response));
});

describe('HeaderBar sim-status chip', () => {
  it('shows running when the shared simulation status poll reports running', async () => {
    fetchSimulationStatus.mockResolvedValue({
      running: true,
      deviceCount: 2,
      uptimeSeconds: 10,
    });

    render(
      <AppProvider>
        <HeaderBar />
      </AppProvider>,
    );

    const chip = await screen.findByTestId('simulation-status-chip');
    await waitFor(() => expect(chip).toHaveTextContent('Running'));
  });

  it('shows stopped when the shared simulation status poll reports stopped', async () => {
    fetchSimulationStatus.mockResolvedValue({
      running: false,
      deviceCount: 0,
      uptimeSeconds: 0,
    });

    render(
      <AppProvider>
        <HeaderBar />
      </AppProvider>,
    );

    const chip = await screen.findByTestId('simulation-status-chip');
    await waitFor(() => expect(chip).toHaveTextContent('Stopped'));
  });
});
