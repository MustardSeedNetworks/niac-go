/**
 * SessionSwitcher — the scenario indicator and switcher in the shell.
 *
 * The switcher is the only production caller of AppContext's session pin, so
 * these tests are what keeps the pin from going back to having no way to set
 * it (which is how it shipped in P1-8).
 */
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { SimulationStatus } from '../api/types';
import { AppProvider } from '../contexts/AppContext';
import '../i18n';
import { SessionSwitcher } from './SessionSwitcher';

const fetchSimulationStatus = vi.fn<() => Promise<SimulationStatus>>();
const fetchDevices = vi.fn<(sessionId: string) => Promise<unknown[]>>();

vi.mock('../api/client', () => ({
  fetchSimulationStatus: () => fetchSimulationStatus(),
  fetchDevices: (sessionId: string) => fetchDevices(sessionId),
  fetchStats: () => Promise.resolve({}),
  fetchNeighbors: () => Promise.resolve([]),
  fetchHistory: () => Promise.resolve([]),
  fetchVersion: () => Promise.resolve({}),
  fetchErrorTypes: () => Promise.resolve({}),
  fetchInterfaces: () => Promise.resolve({ interfaces: [] }),
}));

const session = (sessionId: string): SimulationStatus => ({
  running: true,
  sessionId,
  deviceCount: 1,
  uptimeSeconds: 1,
});

function renderSwitcher() {
  return render(
    <AppProvider>
      <SessionSwitcher />
    </AppProvider>,
  );
}

beforeEach(() => {
  fetchSimulationStatus.mockReset();
  fetchDevices.mockReset().mockResolvedValue([]);
});

describe('SessionSwitcher', () => {
  it('says so when no scenario is running', async () => {
    fetchSimulationStatus.mockResolvedValue({ running: false } as SimulationStatus);
    renderSwitcher();

    await waitFor(() =>
      expect(screen.getByTestId('session-switcher')).toHaveTextContent('No scenario running'),
    );
  });

  it('names the running scenario without offering a choice of one', async () => {
    fetchSimulationStatus.mockResolvedValue({
      ...session('hospital'),
      sessions: [session('hospital')],
    });
    renderSwitcher();

    await waitFor(() =>
      expect(screen.getByTestId('session-switcher')).toHaveTextContent('hospital'),
    );
    expect(screen.queryByTestId('session-switcher-select')).not.toBeInTheDocument();
  });

  it('switches the scenario runtime reads are scoped to', async () => {
    fetchSimulationStatus.mockResolvedValue({
      ...session('hospital'),
      sessions: [session('hospital'), session('warehouse')],
    });
    renderSwitcher();

    const select = await screen.findByTestId('session-switcher-select');
    await waitFor(() => expect(fetchDevices).toHaveBeenCalledWith('hospital'));

    fireEvent.change(select, { target: { value: 'warehouse' } });

    await waitFor(() => expect(fetchDevices).toHaveBeenCalledWith('warehouse'));
    expect(select).toHaveValue('warehouse');
  });
});
