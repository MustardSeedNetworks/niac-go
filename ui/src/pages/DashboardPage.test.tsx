/**
 * DashboardPage.test.tsx — locks the dashboard-honesty fixes (PR
 * "Dashboard honesty + shell sim-status"):
 *   - the Error Injection catalog deep-links to /traffic with the error
 *     type preselected via query param, instead of duplicating the
 *     injection form on the dashboard
 *   - the Automation timeline card (a verbatim duplicate of Recent Runs)
 *     is gone
 *   - "View all history" points at the real Recent Runs section on
 *     /traffic instead of the top of that page
 *   - the RX/TX and DNS/DHCP stat cards show both real counters as a
 *     labeled pair, not one smuggled into the other's helper text
 *   - a "Start a Simulation" quick action links to /runtime
 */
import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type {
  ErrorInjectionInfo,
  HistoryRecord,
  SimulationStatus,
  StackStatsResponse,
} from '../api/types';
import { AppProvider } from '../contexts/AppContext';
import '../i18n';
import { DashboardPage } from './DashboardPage';

const fetchSimulationStatus = vi.fn<() => Promise<SimulationStatus>>();
const fetchStats = vi.fn<() => Promise<StackStatsResponse>>();
const fetchDevices = vi.fn();
const fetchHistory = vi.fn<() => Promise<HistoryRecord[]>>();
const fetchNeighbors = vi.fn();
const fetchVersion = vi.fn();
const fetchErrorTypes = vi.fn<() => Promise<ErrorInjectionInfo>>();
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

const stats: StackStatsResponse = {
  timestamp: new Date().toISOString(),
  interface: 'eth0',
  version: '0.0.0',
  deviceCount: 5,
  stack: {
    packetsSent: 200,
    packetsReceived: 1000,
    arpRequests: 0,
    arpReplies: 0,
    icmpRequests: 0,
    icmpReplies: 0,
    dnsQueries: 40,
    dhcpRequests: 12,
    snmpQueries: 0,
    errors: 0,
  },
};

const history: HistoryRecord[] = [
  {
    id: 1,
    configName: 'demo.yaml',
    startedAt: new Date().toISOString(),
    duration: '30s',
    interface: 'eth0',
    deviceCount: 3,
    packetsReceived: 100,
    packetsSent: 50,
    errors: 0,
  },
];

const errorInfo: ErrorInjectionInfo = {
  availableTypes: [
    { type: 'drop', description: 'Drop packets' },
    { type: 'delay', description: 'Delay packets' },
  ],
  info: 'Inject errors on a device interface.',
};

function renderDashboard() {
  return render(
    <MemoryRouter>
      <AppProvider>
        <DashboardPage />
      </AppProvider>
    </MemoryRouter>,
  );
}

describe('DashboardPage', () => {
  beforeEach(() => {
    fetchStats.mockReset().mockResolvedValue(stats);
    fetchDevices.mockReset().mockResolvedValue([]);
    fetchHistory.mockReset().mockResolvedValue(history);
    fetchNeighbors.mockReset().mockResolvedValue([]);
    fetchVersion.mockReset().mockResolvedValue({ version: '0.0.0' });
    fetchErrorTypes.mockReset().mockResolvedValue(errorInfo);
    fetchInterfaces.mockReset().mockResolvedValue({ interfaces: [] });
    fetchSimulationStatus.mockReset().mockResolvedValue({
      running: true,
      deviceCount: 5,
      uptimeSeconds: 120,
    });
  });

  it('renders RX and TX as an honest labeled pair, not one hidden in helper text', async () => {
    renderDashboard();
    await screen.findByText('1,000'); // RX value renders once stats resolve
    expect(screen.getByText('1,000')).toBeInTheDocument();
    expect(screen.getByText('200')).toBeInTheDocument();
    expect(screen.getByText('RX')).toBeInTheDocument();
    expect(screen.getByText('TX')).toBeInTheDocument();
  });

  it('renders DNS and DHCP as an honest labeled pair, not one hidden in helper text', async () => {
    renderDashboard();
    await screen.findByText('40');
    expect(screen.getByText('40')).toBeInTheDocument();
    expect(screen.getByText('12')).toBeInTheDocument();
    expect(screen.getByText('DNS')).toBeInTheDocument();
    expect(screen.getByText('DHCP')).toBeInTheDocument();
  });

  it('does not render an Automation timeline card duplicating Recent Runs', async () => {
    renderDashboard();
    await screen.findByText('demo.yaml');
    expect(screen.queryByText('Automation timeline')).not.toBeInTheDocument();
  });

  it('"View all history" links to the real Recent Runs section on /traffic', async () => {
    renderDashboard();
    const link = await screen.findByText('View all history →');
    expect(link.closest('a')).toHaveAttribute('href', '/traffic#recent-runs');
  });

  it('has a "Start a Simulation" quick action linking to /runtime', async () => {
    renderDashboard();
    const link = await screen.findByText('Start a Simulation');
    expect(link.closest('a')).toHaveAttribute('href', '/runtime');
  });

  it('deep-links each catalog error type to /traffic with the type preselected, instead of an inline form', async () => {
    renderDashboard();
    const toggle = await screen.findByText('Error Injection');
    toggle.closest('button')?.click();

    const dropLink = await screen.findByText('drop');
    expect(dropLink.closest('a')).toHaveAttribute('href', '/traffic?errorType=drop');

    const delayLink = screen.getByText('delay');
    expect(delayLink.closest('a')).toHaveAttribute('href', '/traffic?errorType=delay');

    // No inline injection form (device/interface/value inputs) on the dashboard.
    expect(screen.queryByRole('slider')).not.toBeInTheDocument();
  });
});
