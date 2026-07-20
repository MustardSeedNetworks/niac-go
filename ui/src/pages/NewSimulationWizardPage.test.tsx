/**
 * NewSimulationWizardPage.test.tsx
 *
 * Covers the two behaviors called out in issue #958: step
 * navigation (Next/Back, Next disabled until step 1 is complete) and
 * that finishing the wizard's step 1 ("start empty" + interface) drives
 * the existing `startSimulation` endpoint — the materialise-then-start
 * seam that unlocks the embedded Devices step.
 */
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { ReactNode } from 'react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { DeviceListResponse, LibraryNetwork, SimulationStatus, Template } from '../api/types';
import { AppProvider } from '../contexts/AppContext';
import '../i18n';
import { NewSimulationWizardPage } from './NewSimulationWizardPage';

const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: vi.fn((key: string) => store[key] ?? null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value;
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key];
    }),
    clear: vi.fn(() => {
      store = {};
    }),
    get length() {
      return Object.keys(store).length;
    },
    key: vi.fn((index: number) => Object.keys(store)[index] ?? null),
  };
})();
Object.defineProperty(globalThis, 'localStorage', { value: localStorageMock, writable: true });

const startSimulation = vi.fn<(payload: unknown) => Promise<SimulationStatus>>();
const preflightSimulation = vi.fn();
const fetchUsableInterfaces = vi.fn();
const fetchTemplates = vi.fn<() => Promise<Template[]>>();
const fetchLibraryNetworks = vi.fn<() => Promise<LibraryNetwork[]>>();
const fetchConfigDevices = vi.fn<() => Promise<DeviceListResponse>>();

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    // AppProvider's background polls — kept quiet/empty for this test.
    fetchSimulationStatus: vi
      .fn()
      .mockResolvedValue({ running: true, interface: 'lo0', deviceCount: 0 }),
    fetchStats: vi.fn(),
    fetchDevices: vi.fn().mockResolvedValue([]),
    fetchHistory: vi.fn(),
    fetchNeighbors: vi.fn(),
    fetchVersion: vi.fn(),
    fetchErrorTypes: vi.fn(),
    fetchInterfaces: vi.fn(),
    // Wizard-specific
    fetchUsableInterfaces: () => fetchUsableInterfaces(),
    fetchTemplates: () => fetchTemplates(),
    fetchConfigDevices: () => fetchConfigDevices(),
    startSimulation: (payload: unknown) => startSimulation(payload),
    preflightSimulation: (payload: unknown) => preflightSimulation(payload),
  };
});

vi.mock('../api/library-client', () => ({
  fetchLibraryNetworks: () => fetchLibraryNetworks(),
}));

const wrapper = ({ children }: { children: ReactNode }) => (
  <MemoryRouter>
    <AppProvider>{children}</AppProvider>
  </MemoryRouter>
);

function renderWizard() {
  return render(<NewSimulationWizardPage />, { wrapper });
}

beforeEach(() => {
  vi.clearAllMocks();
  fetchUsableInterfaces.mockResolvedValue({
    interfaces: [{ name: 'lo0', addresses: ['127.0.0.1'], isUp: true, isLoopback: true }],
  });
  fetchTemplates.mockResolvedValue([]);
  fetchLibraryNetworks.mockResolvedValue([]);
  fetchConfigDevices.mockResolvedValue({ devices: [], total: 0 });
  startSimulation.mockResolvedValue({
    running: true,
    interface: 'lo0',
    deviceCount: 0,
    uptimeSeconds: 0,
  });
  preflightSimulation.mockResolvedValue({
    safe: true,
    topology: {
      binding: {
        attachment: 'tester',
        interface: 'lo0',
        mode: 'access',
        accessVlan: 200,
        network: 'lab-access',
        wireTagged: false,
      },
      networks: [],
      interfaces: [],
      routes: [],
      dhcpScopes: [],
    },
    diagnostics: [],
  });
});

describe('NewSimulationWizardPage — step navigation', () => {
  it('disables Next on step 1 until an interface and a source are picked', async () => {
    const user = userEvent.setup();
    renderWizard();

    await waitFor(() => expect(screen.getByTestId('wizard-next-button')).toBeInTheDocument());
    expect(screen.getByTestId('wizard-next-button')).toBeDisabled();

    await waitFor(() => expect(screen.getByTestId('wizard-interface-select')).not.toBeDisabled());
    await user.selectOptions(screen.getByTestId('wizard-interface-select'), 'lo0');
    expect(screen.getByTestId('wizard-next-button')).toBeDisabled();

    await user.click(screen.getByTestId('wizard-start-empty'));
    expect(screen.getByTestId('wizard-next-button')).not.toBeDisabled();
  });

  it('starts the simulation and advances to Devices on Next, then Back returns to Template', async () => {
    const user = userEvent.setup();
    renderWizard();

    await waitFor(() => expect(screen.getByTestId('wizard-interface-select')).not.toBeDisabled());
    await user.selectOptions(screen.getByTestId('wizard-interface-select'), 'lo0');
    await user.click(screen.getByTestId('wizard-start-empty'));
    await user.click(screen.getByTestId('wizard-next-button'));

    await waitFor(() => expect(screen.getByTestId('wizard-preflight-check')).toBeInTheDocument());
    expect(startSimulation).not.toHaveBeenCalled();
    await user.click(screen.getByTestId('wizard-preflight-check'));
    await waitFor(() => expect(screen.getByTestId('wizard-preflight-start')).not.toBeDisabled());
    await user.click(screen.getByTestId('wizard-preflight-start'));

    await waitFor(() => expect(startSimulation).toHaveBeenCalledTimes(1));
    expect(startSimulation).toHaveBeenCalledWith(
      expect.objectContaining({
        interface: 'lo0',
        attachmentMode: 'access',
        accessVlan: 200,
      }),
    );

    await waitFor(() =>
      expect(screen.getByTestId('wizard-step-devices-content')).toBeInTheDocument(),
    );
    expect(screen.getByTestId('wizard-step-template')).toHaveAttribute('data-status', 'done');
    expect(screen.getByTestId('wizard-step-devices')).toHaveAttribute('data-status', 'active');

    await user.click(screen.getByTestId('wizard-back-button'));
    await waitFor(() => expect(screen.getByTestId('wizard-preflight-check')).toBeInTheDocument());
  });
});
