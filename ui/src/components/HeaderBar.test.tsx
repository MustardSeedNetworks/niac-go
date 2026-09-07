/**
 * HeaderBar.test.tsx — locks the shell's scenario indicator. It sits next to
 * ConnectionStatus and reads the shared AppContext poll, so adding it must not
 * spin up a second fetch against /api/v1/simulation. What it shows is
 * SessionSwitcher's own test; this one covers the header wiring.
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

describe('HeaderBar scenario indicator', () => {
  it('names the running scenario from the shared simulation status poll', async () => {
    fetchSimulationStatus.mockResolvedValue({
      running: true,
      sessionId: 'hospital',
      deviceCount: 2,
      uptimeSeconds: 10,
      sessions: [{ running: true, sessionId: 'hospital', deviceCount: 2, uptimeSeconds: 10 }],
    });

    render(
      <AppProvider>
        <HeaderBar />
      </AppProvider>,
    );

    const switcher = await screen.findByTestId('session-switcher');
    await waitFor(() => expect(switcher).toHaveTextContent('hospital'));
    expect(fetchSimulationStatus).toHaveBeenCalledTimes(1);
  });

  it('stays on screen with no scenario running', async () => {
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

    const switcher = await screen.findByTestId('session-switcher');
    await waitFor(() => expect(switcher).toHaveTextContent('No scenario running'));
  });
});
