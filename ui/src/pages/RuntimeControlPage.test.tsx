/**
 * RuntimeControlPage.test.tsx
 *
 * Regression test for PR "3b — confirm-modal consolidation": the
 * stop-simulation guard used to be a bare `window.confirm(...)`, which
 * can't be styled, localized, or tested through Testing Library the same
 * way as the rest of the app's confirmations. This pins the migrated
 * behavior onto the shared `ConfirmModal`: clicking Stop opens the modal
 * without stopping anything, Cancel dismisses it without calling
 * `stopSimulation`, and confirming calls it exactly once.
 */
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { SimulationStatus } from '../api/types';
import { AppProvider } from '../contexts/AppContext';
import '../i18n';
import { RuntimeControlPage } from './RuntimeControlPage';

const stopSimulation = vi.fn<() => Promise<void>>();

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    fetchSimulationStatus: vi.fn(),
    fetchStats: vi.fn(),
    fetchDevices: vi.fn(),
    fetchHistory: vi.fn(),
    fetchNeighbors: vi.fn(),
    fetchVersion: vi.fn(),
    fetchErrorTypes: vi.fn(),
    fetchInterfaces: vi.fn(),
    fetchUsableInterfaces: vi.fn(),
    fetchUserConfigContent: vi.fn(),
    startSimulation: vi.fn(),
    stopSimulation: () => stopSimulation(),
  };
});

const running: SimulationStatus = {
  running: true,
  interface: 'eth0',
  deviceCount: 3,
  uptimeSeconds: 42,
};

const wrapper = ({ children }: { children: ReactNode }) => (
  <MemoryRouter>
    <AppProvider>{children}</AppProvider>
  </MemoryRouter>
);

function renderPage() {
  return render(<RuntimeControlPage />, { wrapper });
}

describe('RuntimeControlPage — stop-simulation confirmation', () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    const client = await import('../api/client');
    vi.mocked(client.fetchSimulationStatus).mockResolvedValue(running);
    vi.mocked(client.fetchDevices).mockResolvedValue([]);
    vi.mocked(client.fetchHistory).mockResolvedValue([]);
    vi.mocked(client.fetchNeighbors).mockResolvedValue([]);
    vi.mocked(client.fetchVersion).mockResolvedValue({ version: '0.0.0' });
    vi.mocked(client.fetchErrorTypes).mockResolvedValue({ availableTypes: [], info: '' });
    vi.mocked(client.fetchInterfaces).mockResolvedValue({ interfaces: [], currentInterface: '' });
    vi.mocked(client.fetchUsableInterfaces).mockResolvedValue({
      interfaces: [],
      currentInterface: '',
    });
    stopSimulation.mockResolvedValue(undefined);
  });

  it('does not call stopSimulation until the confirm modal is accepted', async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText(/stop simulation/i)).toBeInTheDocument());

    fireEvent.click(screen.getByRole('button', { name: /stop simulation/i }));

    expect(await screen.findByText(/interrupt the current run/i)).toBeInTheDocument();
    expect(stopSimulation).not.toHaveBeenCalled();
  });

  it('cancel dismisses the modal without stopping the simulation', async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText(/stop simulation/i)).toBeInTheDocument());

    fireEvent.click(screen.getByRole('button', { name: /stop simulation/i }));
    await screen.findByText(/interrupt the current run/i);

    fireEvent.click(screen.getByRole('button', { name: /^cancel$/i }));

    await waitFor(() =>
      expect(screen.queryByText(/interrupt the current run/i)).not.toBeInTheDocument(),
    );
    expect(stopSimulation).not.toHaveBeenCalled();
  });

  it('confirming the modal calls stopSimulation exactly once', async () => {
    renderPage();
    await waitFor(() => expect(screen.getByText(/stop simulation/i)).toBeInTheDocument());

    fireEvent.click(screen.getByRole('button', { name: /stop simulation/i }));
    await screen.findByText(/interrupt the current run/i);

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: /^stop$/i }));
    });

    await waitFor(() => expect(stopSimulation).toHaveBeenCalledTimes(1));
  });
});
