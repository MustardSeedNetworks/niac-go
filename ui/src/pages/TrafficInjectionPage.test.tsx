/**
 * TrafficInjectionPage.test.tsx — Fault Injection says why it is idle
 * (UI-NIAC-4, niac-go#2188).
 *
 * With no simulation running there is nothing to inject into, so every
 * control on the page was disabled with nothing to explain it and no way to
 * get to a simulation from here. The page now answers the same three things
 * the Dashboard's rollup answers — what state this is, why, and the one
 * action that changes it — and only renders the panels once there is a
 * simulation for them to act on.
 */

import enPages from '@locales/en/pages.json';
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { SimulationStatus } from '../api/types';
import { AppProvider } from '../contexts/AppContext';
import '../i18n';
import { TrafficInjectionPage } from './TrafficInjectionPage';

const fetchSimulationStatus = vi.fn<() => Promise<SimulationStatus>>();
const fetchDevices = vi.fn();
const fetchErrorTypes = vi.fn();
const fetchInterfaces = vi.fn();
const fetchStats = vi.fn();
const fetchHistory = vi.fn();
const fetchNeighbors = vi.fn();
const fetchVersion = vi.fn();

vi.mock('../api/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../api/client')>()),
  fetchSimulationStatus: () => fetchSimulationStatus(),
  fetchDevices: () => fetchDevices(),
  fetchErrorTypes: () => fetchErrorTypes(),
  fetchInterfaces: () => fetchInterfaces(),
  fetchStats: () => fetchStats(),
  fetchHistory: () => fetchHistory(),
  fetchNeighbors: () => fetchNeighbors(),
  fetchVersion: () => fetchVersion(),
}));

const idle: SimulationStatus = { running: false, deviceCount: 0, uptimeSeconds: 0 };
const running: SimulationStatus = {
  running: true,
  sessionId: 'default',
  deviceCount: 3,
  uptimeSeconds: 60,
  interface: 'eth0',
};

function renderPage() {
  return render(
    <MemoryRouter>
      <AppProvider>
        <TrafficInjectionPage />
      </AppProvider>
    </MemoryRouter>,
  );
}

describe('TrafficInjectionPage while nothing is running', () => {
  beforeEach(() => {
    fetchDevices.mockReset().mockResolvedValue([]);
    fetchErrorTypes.mockReset().mockResolvedValue({ availableTypes: [], info: '' });
    fetchInterfaces.mockReset().mockResolvedValue({ interfaces: [] });
    fetchStats.mockReset().mockResolvedValue(null);
    fetchHistory.mockReset().mockResolvedValue([]);
    fetchNeighbors.mockReset().mockResolvedValue([]);
    fetchVersion.mockReset().mockResolvedValue({ version: '0.0.0' });
    fetchSimulationStatus.mockReset().mockResolvedValue(idle);
  });

  it('says what state the page is in and why', async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByText(enPages.traffic.idle.headline)).toBeInTheDocument(),
    );
    expect(screen.getByText(enPages.traffic.idle.body)).toBeInTheDocument();
  });

  it('offers the one action that changes it', async () => {
    renderPage();
    const action = await screen.findByRole('link', {
      name: enPages.traffic.idle.startAction,
    });
    expect(action).toHaveAttribute('href', '/runtime');
  });

  it('does not render controls there is nothing to act on', async () => {
    renderPage();
    await waitFor(() =>
      expect(screen.getByText(enPages.traffic.idle.headline)).toBeInTheDocument(),
    );
    // The disabled panels were the defect: a wall of dead controls with no
    // explanation. They are absent, not merely greyed out.
    expect(screen.queryByText(enPages.traffic.page.errorInjectionTitle)).toBeNull();
    expect(screen.queryByText(enPages.traffic.page.pcapReplayTitle)).toBeNull();
  });

  it('renders the panels once a simulation is running', async () => {
    fetchSimulationStatus.mockResolvedValue(running);
    renderPage();
    await waitFor(() =>
      expect(screen.getByText(enPages.traffic.page.errorInjectionTitle)).toBeInTheDocument(),
    );
    expect(screen.getByText(enPages.traffic.page.pcapReplayTitle)).toBeInTheDocument();
    expect(screen.queryByText(enPages.traffic.idle.headline)).toBeNull();
  });
});
