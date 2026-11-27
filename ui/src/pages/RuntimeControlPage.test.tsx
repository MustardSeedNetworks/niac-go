// This isolated component fixture represents an authenticated operator.
vi.mock('../contexts/ScopeContext', () => ({
  useActionPermission: () => ({ disabled: false }),
}));

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
import type { SimulationRequest, SimulationStatus } from '../api/types';
import { AppProvider } from '../contexts/AppContext';
import '../i18n';
import { useUIStore } from '../stores/ui-store';
import { RuntimeControlPage } from './RuntimeControlPage';

const stopSimulation = vi.fn<(sessionId?: string) => Promise<void>>();
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
    fetchHistoryPage: vi.fn().mockResolvedValue([]),
    fetchNeighbors: vi.fn(),
    fetchVersion: vi.fn(),
    fetchErrorTypes: vi.fn(),
    fetchInterfaces: vi.fn(),
    fetchUsableInterfaces: vi.fn(),
    // AP-0: the binding inputs read their choices from the daemon instead of
    // defaulting to a typed `tester` no generated scenario answers to.
    fetchAttachmentPolicies: () =>
      Promise.resolve({ policies: [{ interface: 'eth0', mode: 'access', accessVlan: 200 }] }),
    fetchSimulationAttachments: () =>
      Promise.resolve({ routed: true, attachments: ['cyberscope'] }),
    preflightSimulation: (request: SimulationPreflightRequest) => preflightSimulation(request),
    startSimulation: (request: SimulationRequest) => startSimulation(request),
    stopSimulation: (sessionId?: string) => stopSimulation(sessionId),
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
    vi.mocked(client.fetchInterfaces).mockResolvedValue({ interfaces: [] });
    vi.mocked(client.fetchUsableInterfaces).mockResolvedValue({
      interfaces: [],
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

  it('stops only the scenario selected from the active-session table', async () => {
    const client = await import('../api/client');
    vi.mocked(client.fetchSimulationStatus).mockResolvedValue({
      ...running,
      sessionId: 'warehouse',
      sessions: [
        { ...running, sessionId: 'hospital', physicalVlan: 200 },
        { ...running, sessionId: 'warehouse', physicalVlan: 201 },
      ],
    });
    renderPage();
    fireEvent.click(await screen.findByTestId('session-stop-hospital'));
    await screen.findByText(/interrupt the current run/i);
    act(() => fireEvent.click(screen.getByRole('button', { name: /^stop$/i })));

    await waitFor(() => expect(stopSimulation).toHaveBeenCalledWith('hospital'));
  });

  // Selection is this browser's, not the daemon's: picking a scenario here
  // rescopes this tab's runtime reads and sends nothing to the daemon, so a
  // second tab keeps reading whatever scenario it had.
  it('rescopes runtime reads to the scenario selected from the table', async () => {
    const client = await import('../api/client');
    vi.mocked(client.fetchSimulationStatus).mockResolvedValue({
      ...running,
      sessionId: 'warehouse',
      sessions: [
        { ...running, sessionId: 'hospital', physicalVlan: 200 },
        { ...running, sessionId: 'warehouse', physicalVlan: 201 },
      ],
    });
    renderPage();
    await waitFor(() => expect(client.fetchDevices).toHaveBeenCalledWith('warehouse'));

    fireEvent.click(await screen.findByTestId('session-select-hospital'));

    await waitFor(() => expect(client.fetchDevices).toHaveBeenCalledWith('hospital'));
  });

  // The running-scenario card sits directly under the table. Left on the
  // daemon's top-level status it would name one scenario while the table
  // marked another selected, and its Stop would stop the one it named.
  it('describes and stops the selected scenario, not the daemon default', async () => {
    const client = await import('../api/client');
    vi.mocked(client.fetchSimulationStatus).mockResolvedValue({
      ...running,
      sessionId: 'warehouse',
      interface: 'veth-warehouse',
      sessions: [
        { ...running, sessionId: 'hospital', interface: 'veth-hospital', physicalVlan: 200 },
        { ...running, sessionId: 'warehouse', interface: 'veth-warehouse', physicalVlan: 201 },
      ],
    });
    renderPage();
    fireEvent.click(await screen.findByTestId('session-select-hospital'));

    // The rollup and the card both describe the run; neither may still be
    // describing the daemon's default.
    await waitFor(() => expect(screen.getAllByText('veth-hospital').length).toBeGreaterThan(0));
    expect(screen.queryByText('veth-warehouse')).not.toBeInTheDocument();

    // The card's Stop, not one of the table's per-row buttons.
    const cardStop = screen.getByTestId('runtime-stop');
    fireEvent.click(cardStop);
    await screen.findByText(/interrupt the current run/i);
    act(() => fireEvent.click(screen.getByRole('button', { name: /^stop$/i })));

    await waitFor(() => expect(stopSimulation).toHaveBeenCalledWith('hospital'));
  });
});

const ROUTED_YAML = 'devices:\n  - name: r1\n    type: router\n';

describe('RuntimeControlPage — routed start preflight', () => {
  beforeEach(async () => {
    vi.clearAllMocks();
    useUIStore.getState().reset();
    useUIStore.getState().setSimulationSettings({
      selectedInterface: 'eth0',
      configSource: 'userConfig',
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
    vi.mocked(client.fetchInterfaces).mockResolvedValue({ interfaces: [] });
    vi.mocked(client.fetchUsableInterfaces).mockResolvedValue({
      interfaces: [{ name: 'eth0', description: 'Ethernet', addresses: [] }],
    });
    const library = await import('../api/library-client');
    vi.mocked(library.fetchLibraryNetworks).mockResolvedValue([
      {
        name: 'routed-lab',
        description: 'Routed acceptance fixture',
        deviceCount: 3,
        modifiedAt: '2026-10-04T00:00:00Z',
        sizeBytes: 64,
        source: 'user',
        valid: true,
      },
    ]);
    vi.mocked(library.fetchLibraryNetworkContent).mockResolvedValue({
      name: 'routed-lab',
      content: ROUTED_YAML,
      format: 'yaml',
      source: 'user',
    });
    preflightSimulation.mockResolvedValue({
      safe: true,
      topology: {
        binding: {
          attachment: 'cyberscope',
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
        attachments: [],
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

  it('preflights the selected routed library network by name and starts with the approved binding', async () => {
    const user = userEvent.setup();
    renderPage();

    await screen.findByText('routed-lab');
    await user.click(screen.getByRole('button', { name: /^select$/i }));
    await user.click(await screen.findByRole('button', { name: /review connection/i }));
    await user.click(await screen.findByTestId('wizard-preflight-check'));

    await waitFor(() =>
      expect(preflightSimulation).toHaveBeenCalledWith({
        interface: 'eth0',
        scenarioName: 'routed-lab',
        attachment: 'cyberscope',
        attachmentMode: 'access',
        accessVlan: 200,
      }),
    );
    expect(startSimulation).not.toHaveBeenCalled();

    await user.click(screen.getByTestId('wizard-preflight-start'));

    await waitFor(() =>
      expect(startSimulation).toHaveBeenCalledWith({
        interface: 'eth0',
        scenarioName: 'routed-lab',
        attachment: 'cyberscope',
        attachmentMode: 'access',
        accessVlan: 200,
      }),
    );
  });

  // A content pack's networks sit in the library but name their walks relative
  // to it (`include_path: ../walks`). Posted inline they resolve against the
  // configs directory and the daemon answers "walk file not found", so every
  // library source, bundle included, is started by name (#3154).
  it.each(['bundle', 'user'] as const)(
    'starts a %s library network by name, not inline',
    async (source) => {
      const user = userEvent.setup();
      const library = await import('../api/library-client');
      vi.mocked(library.fetchLibraryNetworks).mockResolvedValue([
        {
          name: 'modern-hospital',
          description: 'Installed pack network',
          deviceCount: 310,
          modifiedAt: '2026-10-04T00:00:00Z',
          sizeBytes: 5_000_000,
          source,
          valid: true,
        },
      ]);
      renderPage();

      await screen.findByText('modern-hospital');
      await user.click(screen.getByRole('button', { name: /^select$/i }));
      await user.click(await screen.findByRole('button', { name: /review connection/i }));
      await user.click(await screen.findByTestId('wizard-preflight-check'));

      await waitFor(() => expect(preflightSimulation).toHaveBeenCalled());
      const sent = preflightSimulation.mock.calls[0]?.[0];
      expect(sent?.scenarioName).toBe('modern-hospital');
      expect(sent?.configData).toBeUndefined();
    },
  );

  // A shipped example has no file in the library: it is started by name, so the
  // request never carries a document that, for a pack, is megabytes.
  it('starts a shipped example by name instead of sending its content', async () => {
    const user = userEvent.setup();
    const library = await import('../api/library-client');
    vi.mocked(library.fetchLibraryNetworks).mockResolvedValue([
      {
        name: 'routed-lab',
        description: 'Shipped example',
        deviceCount: 3,
        modifiedAt: '2026-10-04T00:00:00Z',
        sizeBytes: 0,
        source: 'example',
        valid: true,
      },
    ]);
    renderPage();

    await screen.findByText('routed-lab');
    await user.click(screen.getByRole('button', { name: /^select$/i }));
    await user.click(await screen.findByRole('button', { name: /review connection/i }));
    await user.click(await screen.findByTestId('wizard-preflight-check'));

    await waitFor(() =>
      expect(preflightSimulation).toHaveBeenCalledWith({
        interface: 'eth0',
        scenarioName: 'routed-lab',
        attachment: 'cyberscope',
        attachmentMode: 'access',
        accessVlan: 200,
      }),
    );
    // The preview may still read the document to show it; the request must not carry it.
    expect(preflightSimulation.mock.calls[0]?.[0].configData).toBeUndefined();
  });
});
