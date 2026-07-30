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
import userEvent from '@testing-library/user-event';
import type { ReactNode } from 'react';
import { MemoryRouter } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { SimulationPreflightReport, SimulationPreflightRequest } from '../api/fabric-types';
import type { SimulationRequest, SimulationStatus, Template } from '../api/types';
import { AppProvider } from '../contexts/AppContext';
import '../i18n';
import { useUIStore } from '../stores/ui-store';
import { RuntimeControlPage } from './RuntimeControlPage';

const stopSimulation = vi.fn<() => Promise<void>>();
const preflightSimulation =
  vi.fn<(request: SimulationPreflightRequest) => Promise<SimulationPreflightReport>>();
const startSimulation = vi.fn<(request: SimulationRequest) => Promise<SimulationStatus>>();

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
    fetchTemplates: vi.fn(),
    preflightSimulation: (request: SimulationPreflightRequest) => preflightSimulation(request),
    startSimulation: (request: SimulationRequest) => startSimulation(request),
    stopSimulation: () => stopSimulation(),
  };
});

vi.mock('../api/library-client', () => ({
  fetchLibraryNetworks: vi.fn().mockResolvedValue([]),
  fetchLibraryNetworkContent: vi.fn(),
}));

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

    act(() => fireEvent.click(screen.getByRole('button', { name: /^stop$/i })));

    await waitFor(() => expect(stopSimulation).toHaveBeenCalledTimes(1));
  });
});

describe('RuntimeControlPage — routed start preflight', () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    useUIStore.getState().reset();
    useUIStore.getState().setSimulationSettings({
      selectedInterface: 'eth0',
      configSource: 'template',
      configName: '',
    });
    const client = await import('../api/client');
    vi.mocked(client.fetchSimulationStatus).mockResolvedValue({
      running: false,
      deviceCount: 0,
      uptimeSeconds: 0,
    });
    vi.mocked(client.fetchDevices).mockResolvedValue([]);
    vi.mocked(client.fetchHistory).mockResolvedValue([]);
    vi.mocked(client.fetchNeighbors).mockResolvedValue([]);
    vi.mocked(client.fetchVersion).mockResolvedValue({ version: '0.0.0' });
    vi.mocked(client.fetchErrorTypes).mockResolvedValue({ availableTypes: [], info: '' });
    vi.mocked(client.fetchInterfaces).mockResolvedValue({ interfaces: [], currentInterface: '' });
    vi.mocked(client.fetchUsableInterfaces).mockResolvedValue({
      interfaces: [{ name: 'eth0', description: 'Ethernet', addresses: [], current: true }],
      currentInterface: 'eth0',
    });
    vi.mocked(client.fetchTemplates).mockResolvedValue([
      {
        name: 'labs/routed.yaml',
        description: 'Routed acceptance fixture',
        deviceCount: 3,
        type: 'router',
      } satisfies Template,
    ]);
    preflightSimulation.mockResolvedValue({
      safe: true,
      topology: {
        binding: {
          attachment: 'tester',
          interface: 'eth0',
          mode: 'access',
          physicalVlan: 200,
          network: 'lab-access',
          wireTagged: false,
        },
        networks: [{ name: 'lab-access', prefix: '10.10.200.0/24' }],
        interfaces: [],
        routes: [],
        dhcpScopes: [],
      },
      diagnostics: [],
    });
    startSimulation.mockResolvedValue({
      running: true,
      interface: 'eth0',
      deviceCount: 3,
      uptimeSeconds: 0,
    });
  });

  it('preflights the selected routed template and starts with the approved binding', async () => {
    const user = userEvent.setup();
    renderPage();

    await screen.findByText('labs/routed.yaml');
    await user.click(screen.getByRole('button', { name: /^select$/i }));
    await user.click(await screen.findByRole('button', { name: /review connection/i }));
    await user.click(await screen.findByTestId('wizard-preflight-check'));

    await waitFor(() =>
      expect(preflightSimulation).toHaveBeenCalledWith({
        interface: 'eth0',
        templateName: 'labs/routed.yaml',
        attachment: 'tester',
        attachmentMode: 'access',
        accessVlan: 200,
      }),
    );
    expect(startSimulation).not.toHaveBeenCalled();

    await user.click(screen.getByTestId('wizard-preflight-start'));

    await waitFor(() =>
      expect(startSimulation).toHaveBeenCalledWith({
        interface: 'eth0',
        templateName: 'labs/routed.yaml',
        attachment: 'tester',
        attachmentMode: 'access',
        accessVlan: 200,
      }),
    );
  });
});
