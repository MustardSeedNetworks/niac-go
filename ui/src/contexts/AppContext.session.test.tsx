/**
 * AppContext session scoping.
 *
 * NIAC can run several scenarios at once. Which one this browser reads is a
 * client-side choice: a server-side "current" session would mean one tab
 * switching scenario silently repoints every other tab.
 */
import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { SimulationStatus } from '../api/types';
import { AppProvider, useAppContext } from './AppContext';

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

function SessionProbe() {
  const { sessionId, setSessionId } = useAppContext();
  return (
    <div>
      <span data-testid="session">{sessionId ?? 'none'}</span>
      <button type="button" onClick={() => setSessionId('warehouse')}>
        pick warehouse
      </button>
    </div>
  );
}

function renderProbe() {
  return render(
    <AppProvider>
      <SessionProbe />
    </AppProvider>,
  );
}

describe('AppContext session scoping', () => {
  beforeEach(() => {
    fetchSimulationStatus.mockReset();
    fetchDevices.mockReset().mockResolvedValue([]);
  });

  it('reads no session while nothing is running', async () => {
    fetchSimulationStatus.mockResolvedValue({ running: false } as SimulationStatus);
    renderProbe();

    await waitFor(() => expect(screen.getByTestId('session')).toHaveTextContent('none'));
    // Requesting a session that does not exist would 404 on every poll.
    expect(fetchDevices).not.toHaveBeenCalled();
  });

  it('adopts the running session and scopes runtime reads to it', async () => {
    fetchSimulationStatus.mockResolvedValue({
      running: true,
      sessionId: 'hospital',
      sessions: [{ running: true, sessionId: 'hospital', deviceCount: 2, uptimeSeconds: 1 }],
    } as SimulationStatus);
    renderProbe();

    await waitFor(() => expect(screen.getByTestId('session')).toHaveTextContent('hospital'));
    await waitFor(() => expect(fetchDevices).toHaveBeenCalledWith('hospital'));
  });

  it('keeps this browser on the session it picked', async () => {
    // The daemon reports hospital as its own selection; this browser chose
    // warehouse and must stay there.
    fetchSimulationStatus.mockResolvedValue({
      running: true,
      sessionId: 'hospital',
      sessions: [
        { running: true, sessionId: 'hospital', deviceCount: 2, uptimeSeconds: 1 },
        { running: true, sessionId: 'warehouse', deviceCount: 1, uptimeSeconds: 1 },
      ],
    } as SimulationStatus);
    renderProbe();

    await waitFor(() => expect(screen.getByTestId('session')).toHaveTextContent('hospital'));
    screen.getByRole('button', { name: /pick warehouse/i }).click();

    await waitFor(() => expect(screen.getByTestId('session')).toHaveTextContent('warehouse'));
    // Switching scenario refetches rather than leaving the previous one's devices on screen.
    await waitFor(() => expect(fetchDevices).toHaveBeenCalledWith('warehouse'));
  });

  it('falls back when the picked session stops running', async () => {
    fetchSimulationStatus.mockResolvedValue({
      running: true,
      sessionId: 'hospital',
      sessions: [{ running: true, sessionId: 'hospital', deviceCount: 2, uptimeSeconds: 1 }],
    } as SimulationStatus);
    renderProbe();

    await waitFor(() => expect(screen.getByTestId('session')).toHaveTextContent('hospital'));
    // warehouse is not in the running set, so the pin is ignored rather than
    // leaving every read 404ing against a stopped session.
    screen.getByRole('button', { name: /pick warehouse/i }).click();

    await waitFor(() => expect(screen.getByTestId('session')).toHaveTextContent('hospital'));
  });
});
