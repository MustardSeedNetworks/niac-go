/**
 * NewSimulationWizardPage.test.tsx
 *
 * Covers draft-first authoring: the picked source is saved as a revisioned
 * draft, edits update that draft, and runtime does not start until the final
 * preflight succeeds.
 */
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { ReactNode } from 'react';
import { MemoryRouter } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { ScenarioDraft } from '../api/library-client';
import type { LibraryNetwork, SimulationStatus, Template } from '../api/types';
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
const createScenarioDraft = vi.fn<(name: string, content: string) => Promise<ScenarioDraft>>();
const createScenarioDraftFromTemplate =
  vi.fn<(name: string, templateName: string) => Promise<ScenarioDraft>>();
const replaceScenarioDraft =
  vi.fn<(name: string, revision: string, content: string) => Promise<ScenarioDraft>>();
const emptyDraftContent = `devices:
  - name: new-device
    type: host
    mac: 02:00:00:00:00:01
`;

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
    startSimulation: (payload: unknown) => startSimulation(payload),
    preflightSimulation: (payload: unknown) => preflightSimulation(payload),
  };
});

vi.mock('../api/library-client', () => ({
  fetchLibraryNetworks: () => fetchLibraryNetworks(),
  createScenarioDraft: (name: string, content: string) => createScenarioDraft(name, content),
  createScenarioDraftFromTemplate: (name: string, templateName: string) =>
    createScenarioDraftFromTemplate(name, templateName),
  replaceScenarioDraft: (name: string, revision: string, content: string) =>
    replaceScenarioDraft(name, revision, content),
}));

vi.mock('../components/config/YamlEditor', () => ({
  YamlEditor: ({ value, onChange }: { value: string; onChange?: (value: string) => void }) => (
    <textarea
      data-testid="wizard-draft-editor"
      value={value}
      onChange={(event) => onChange?.(event.target.value)}
    />
  ),
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
  createScenarioDraft.mockResolvedValue({
    name: 'scenario-20260728-120000',
    content: emptyDraftContent,
    format: 'yaml',
    revision: 'revision-1',
    modifiedAt: '2026-07-28T12:00:00Z',
    sizeBytes: emptyDraftContent.length,
  });
  createScenarioDraftFromTemplate.mockResolvedValue({
    name: 'scenario-20260728-120000',
    content: emptyDraftContent,
    format: 'yaml',
    revision: 'revision-1',
    modifiedAt: '2026-07-28T12:00:00Z',
    sizeBytes: emptyDraftContent.length,
  });
  replaceScenarioDraft.mockResolvedValue({
    name: 'scenario-20260728-120000',
    content: 'devices:\n  - name: edge-1\n    type: router\n    mac: 02:00:00:00:00:01\n',
    format: 'yaml',
    revision: 'revision-2',
    modifiedAt: '2026-07-28T12:01:00Z',
    sizeBytes: 76,
  });
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

  it('creates and edits a draft without starting or changing the active runtime', async () => {
    const user = userEvent.setup();
    renderWizard();

    await waitFor(() => expect(screen.getByTestId('wizard-interface-select')).not.toBeDisabled());
    await user.selectOptions(screen.getByTestId('wizard-interface-select'), 'lo0');
    await user.click(screen.getByTestId('wizard-start-empty'));
    await user.click(screen.getByTestId('wizard-next-button'));

    await waitFor(() => expect(screen.getByTestId('wizard-draft-editor')).toBeInTheDocument());
    expect(createScenarioDraft).toHaveBeenCalledWith(
      expect.stringMatching(/^scenario-/),
      emptyDraftContent,
    );
    expect(startSimulation).not.toHaveBeenCalled();
    expect(screen.getByTestId('wizard-draft-editor')).toHaveTextContent('new-device');

    fireEvent.change(screen.getByTestId('wizard-draft-editor'), {
      target: {
        value: 'devices:\n  - name: edge-1\n    type: router\n    mac: 02:00:00:00:00:01\n',
      },
    });
    await user.click(screen.getByTestId('wizard-next-button'));
    await waitFor(() => expect(replaceScenarioDraft).toHaveBeenCalledTimes(1));
    expect(replaceScenarioDraft).toHaveBeenCalledWith(
      'scenario-20260728-120000',
      'revision-1',
      expect.stringContaining('edge-1'),
    );
    expect(startSimulation).not.toHaveBeenCalled();

    expect(screen.getByTestId('wizard-step-template')).toHaveAttribute('data-status', 'done');
    expect(screen.getByTestId('wizard-step-protocols')).toHaveAttribute('data-status', 'active');

    await user.click(screen.getByTestId('wizard-back-button'));
    await waitFor(() => expect(screen.getByTestId('wizard-draft-editor')).toBeInTheDocument());
  });

  it('saves dirty edits before Back and resumes the same draft for an unchanged source', async () => {
    const user = userEvent.setup();
    renderWizard();

    await waitFor(() => expect(screen.getByTestId('wizard-interface-select')).not.toBeDisabled());
    await user.selectOptions(screen.getByTestId('wizard-interface-select'), 'lo0');
    await user.click(screen.getByTestId('wizard-start-empty'));
    await user.click(screen.getByTestId('wizard-next-button'));
    await waitFor(() => expect(screen.getByTestId('wizard-draft-editor')).toBeInTheDocument());

    fireEvent.change(screen.getByTestId('wizard-draft-editor'), {
      target: {
        value: 'devices:\n  - name: edge-1\n    type: router\n    mac: 02:00:00:00:00:01\n',
      },
    });
    await user.click(screen.getByTestId('wizard-back-button'));

    await waitFor(() => expect(replaceScenarioDraft).toHaveBeenCalledTimes(1));
    expect(await screen.findByTestId('wizard-interface-select')).toHaveValue('lo0');
    await user.click(screen.getByTestId('wizard-next-button'));

    await waitFor(() => expect(screen.getByTestId('wizard-draft-editor')).toBeInTheDocument());
    expect(createScenarioDraft).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId('wizard-draft-editor')).toHaveTextContent('edge-1');
  });

  it('starts only after draft review and a successful preflight', async () => {
    const user = userEvent.setup();
    renderWizard();

    await waitFor(() => expect(screen.getByTestId('wizard-interface-select')).not.toBeDisabled());
    await user.selectOptions(screen.getByTestId('wizard-interface-select'), 'lo0');
    await user.click(screen.getByTestId('wizard-start-empty'));
    await user.click(screen.getByTestId('wizard-next-button'));
    await waitFor(() => expect(screen.getByTestId('wizard-draft-editor')).toBeInTheDocument());
    await user.click(screen.getByTestId('wizard-next-button'));
    await user.click(screen.getByTestId('wizard-next-button'));
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
        configData: emptyDraftContent,
        attachmentMode: 'access',
        accessVlan: 200,
      }),
    );
    expect(await screen.findByTestId('wizard-finish-draft-name')).toHaveTextContent(
      'scenario-20260728-120000',
    );
  });

  it('keeps the selected source and interface after a failed preflight and allows retry', async () => {
    const user = userEvent.setup();
    preflightSimulation.mockRejectedValueOnce(new Error('preflight unavailable'));
    renderWizard();

    await waitFor(() => expect(screen.getByTestId('wizard-interface-select')).not.toBeDisabled());
    await user.selectOptions(screen.getByTestId('wizard-interface-select'), 'lo0');
    await user.click(screen.getByTestId('wizard-start-empty'));
    await user.click(screen.getByTestId('wizard-next-button'));
    await waitFor(() => expect(screen.getByTestId('wizard-draft-editor')).toBeInTheDocument());
    await user.click(screen.getByTestId('wizard-next-button'));
    await user.click(screen.getByTestId('wizard-next-button'));
    await user.click(screen.getByTestId('wizard-next-button'));
    await user.click(screen.getByTestId('wizard-preflight-check'));

    expect(await screen.findByRole('alert')).toHaveTextContent('preflight unavailable');
    expect(screen.getByTestId('wizard-preflight-start')).toBeDisabled();

    await user.click(screen.getByTestId('wizard-back-button'));
    await user.click(screen.getByTestId('wizard-back-button'));
    await user.click(screen.getByTestId('wizard-back-button'));
    await user.click(screen.getByTestId('wizard-back-button'));

    expect(await screen.findByTestId('wizard-interface-select')).toHaveValue('lo0');
    expect(screen.getByTestId('wizard-start-empty')).toHaveClass('border-brand-accent');
    expect(screen.getByTestId('wizard-next-button')).not.toBeDisabled();

    await user.click(screen.getByTestId('wizard-next-button'));
    await waitFor(() => expect(screen.getByTestId('wizard-draft-editor')).toBeInTheDocument());
    expect(createScenarioDraft).toHaveBeenCalledTimes(1);
    await user.click(screen.getByTestId('wizard-next-button'));
    await user.click(screen.getByTestId('wizard-next-button'));
    await user.click(screen.getByTestId('wizard-next-button'));
    await user.click(screen.getByTestId('wizard-preflight-check'));

    await waitFor(() => expect(screen.getByTestId('wizard-preflight-start')).not.toBeDisabled());
    expect(preflightSimulation).toHaveBeenLastCalledWith(
      expect.objectContaining({ interface: 'lo0', configData: emptyDraftContent }),
    );
  });
});
