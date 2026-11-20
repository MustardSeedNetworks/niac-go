// This isolated component fixture represents an authenticated operator.
vi.mock('../contexts/ScopeContext', () => ({
  useActionPermission: () => ({ disabled: false }),
}));

/**
 * NewSimulationWizardPage.startingPoint.test.tsx
 *
 * The scenario step offers several ways into the same wizard -- a scenario
 * pack, the generated fleet tuned by hand, a blank start -- and exactly one of
 * them can be the starting point Next will build. Kept apart from
 * NewSimulationWizardPage.test.tsx, which mocks the pack picker away to cover
 * draft handling.
 */
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defaultScenarioRequest, type ScenarioPack } from '../api/scenario-client';
import { AppProvider } from '../contexts/AppContext';
import { MemoryDataRouter } from '../test/MemoryDataRouter';
import '../i18n';
import { NewSimulationWizardPage } from './NewSimulationWizardPage';

const hospital: ScenarioPack = {
  id: 'hospital',
  version: '1.2.0',
  manifestVersion: 4,
  name: 'Hospital network',
  description: 'Acute-care and ambulatory sites.',
  mapPurpose: 'presentation',
  request: { ...defaultScenarioRequest(), domain: 'care.example' },
  manifest: {
    deviceCount: 75,
    networkCount: 12,
    linkCount: 88,
    deviceNamesSha256: 'devices',
    networksSha256: 'networks',
    linksSha256: 'links',
    interfacesSha256: 'interfaces',
  },
};

const fetchSimulationStatus = vi.hoisted(() => vi.fn());
const fetchHistory = vi.hoisted(() => vi.fn());
const fetchUsableInterfaces = vi.hoisted(() => vi.fn());
const generateScenario = vi.hoisted(() => vi.fn());

const idleStatus = { running: false, interface: '', deviceCount: 0, uptimeSeconds: 0 };
const loopbackOnly = {
  interfaces: [{ name: 'lo0', description: '', addresses: ['127.0.0.1'] }],
};

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    // AppProvider's background polls — kept quiet for this fixture.
    fetchSimulationStatus,
    fetchStats: vi.fn(),
    fetchDevices: vi.fn().mockResolvedValue([]),
    fetchHistory,
    fetchNeighbors: vi.fn(),
    fetchVersion: vi.fn(),
    fetchErrorTypes: vi.fn(),
    fetchInterfaces: vi.fn(),
    fetchUsableInterfaces,
    fetchAttachmentPolicies: () => Promise.resolve({ policies: [] }),
    fetchSimulationAttachments: () => Promise.resolve({ routed: true, attachments: [] }),
  };
});

vi.mock('../api/library-client', () => ({
  fetchLibraryNetworks: vi.fn().mockResolvedValue([]),
  createScenarioDraft,
  replaceScenarioDraft: vi.fn(),
  deleteScenarioDraft: vi.fn(),
}));

const fetchScenarioPacks = vi.hoisted(() => vi.fn());
const createScenarioDraft = vi.hoisted(() => vi.fn());

vi.mock('../api/scenario-client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/scenario-client')>();
  return { ...actual, fetchScenarioPacks, generateScenario };
});

const wrapper = ({ children }: { children: ReactNode }) => (
  <MemoryDataRouter>
    <AppProvider>{children}</AppProvider>
  </MemoryDataRouter>
);

// Every starting-point control carries data-wizard-source and reports its state
// through aria-pressed, so "one selected card" is a single query rather than a
// CSS read. The attribute is what separates a starting point from the config
// picker's view filter and favourite stars, which are aria-pressed too.
const pressed = (): string[] =>
  Array.from(
    screen
      .getByTestId('wizard-step-panel')
      .querySelectorAll('[data-wizard-source][aria-pressed="true"]'),
  ).map((element) => element.getAttribute('data-testid') ?? '');

describe('new simulation starting point', () => {
  beforeEach(() => {
    fetchScenarioPacks.mockReset().mockResolvedValue([hospital]);
    fetchSimulationStatus.mockReset().mockResolvedValue(idleStatus);
    fetchHistory.mockReset().mockResolvedValue([]);
    fetchUsableInterfaces.mockReset().mockResolvedValue(loopbackOnly);
    generateScenario.mockReset().mockResolvedValue({ content: 'devices: []\n' });
    createScenarioDraft.mockReset().mockResolvedValue({
      name: 'draft',
      content: 'devices: []\n',
      format: 'yaml',
      revision: 'r1',
      modifiedAt: '2026-11-20T12:00:00Z',
      sizeBytes: 12,
    });
  });

  // UI-NIAC-1 / #2185. A pack card and the "Use generated fleet" button are two
  // ways of choosing the same starting point, so only one of them is ever the
  // selection. Before the fix a pack click only loaded the request — it never
  // recorded a source, so Next stayed disabled — and pressing the fleet button
  // afterwards left both cards highlighted.
  it('treats a scenario pack and the generated fleet as one selection', async () => {
    const user = userEvent.setup();
    render(<NewSimulationWizardPage />, { wrapper });

    await waitFor(() => expect(screen.getByTestId('wizard-interface-select')).not.toBeDisabled());
    await user.selectOptions(screen.getByTestId('wizard-interface-select'), 'lo0');

    await user.click(await screen.findByTestId('scenario-pack-hospital'));

    // Picking a pack is itself a choice of starting point.
    expect(screen.getByTestId('fleet-domain')).toHaveValue('care.example');
    expect(screen.getByTestId('wizard-next-button')).toBeEnabled();
    expect(pressed()).toEqual(['scenario-pack-hospital']);

    // Choosing the fleet on its own terms drops the pack, keeping its counts as
    // the starting point the operator now tunes by hand.
    await user.click(screen.getByTestId('wizard-select-fleet'));
    expect(pressed()).toEqual(['wizard-select-fleet']);
    expect(screen.getByTestId('fleet-domain')).toHaveValue('care.example');

    // ...and going back to the pack drops the hand-tuned selection.
    await user.click(screen.getByTestId('scenario-pack-hospital'));
    expect(pressed()).toEqual(['scenario-pack-hospital']);

    // Editing any generator field is no longer that pack.
    fireEvent.change(screen.getByTestId('fleet-domain'), { target: { value: 'ward.example' } });
    expect(pressed()).toEqual(['wizard-select-fleet']);
  });

  it('a blank start replaces the generated selection', async () => {
    const user = userEvent.setup();
    render(<NewSimulationWizardPage />, { wrapper });

    await waitFor(() => expect(screen.getByTestId('wizard-interface-select')).not.toBeDisabled());
    await user.click(await screen.findByTestId('scenario-pack-hospital'));
    await user.click(screen.getByTestId('wizard-start-empty'));

    expect(pressed()).toEqual(['wizard-start-empty']);
  });

  // AUD-5. An unmodified pack is a start-to-finish run: the draft is built
  // exactly as Next builds it, but the content-editing steps are skipped.
  it('starts a pack as-is, straight to the connection step', async () => {
    const user = userEvent.setup();
    render(<NewSimulationWizardPage />, { wrapper });

    await waitFor(() => expect(screen.getByTestId('wizard-interface-select')).not.toBeDisabled());
    expect(screen.queryByTestId('wizard-start-as-is-button')).not.toBeInTheDocument();
    await user.selectOptions(screen.getByTestId('wizard-interface-select'), 'lo0');
    expect(screen.queryByTestId('wizard-start-as-is-button')).not.toBeInTheDocument();
    await user.click(await screen.findByTestId('scenario-pack-hospital'));
    await user.click(screen.getByTestId('wizard-start-as-is-button'));

    expect(await screen.findByTestId('wizard-preflight-check')).toBeInTheDocument();
    expect(generateScenario).toHaveBeenCalledWith(hospital.request);
    expect(createScenarioDraft).toHaveBeenCalledTimes(1);
    // Back from Connection lands on Review, which needs the draft.
    await user.click(screen.getByTestId('wizard-back-button'));
    expect(screen.getByTestId('wizard-next-button')).toBeEnabled();
  });

  it('offers no as-is start for authored content', async () => {
    const user = userEvent.setup();
    render(<NewSimulationWizardPage />, { wrapper });

    await waitFor(() => expect(screen.getByTestId('wizard-interface-select')).not.toBeDisabled());
    await user.selectOptions(screen.getByTestId('wizard-interface-select'), 'lo0');
    await user.click(screen.getByTestId('wizard-start-empty'));
    expect(screen.getByTestId('wizard-next-button')).toBeEnabled();
    expect(screen.queryByTestId('wizard-start-as-is-button')).not.toBeInTheDocument();

    await user.click(await screen.findByTestId('scenario-pack-hospital'));
    expect(screen.getByTestId('wizard-start-as-is-button')).toBeEnabled();
    await user.click(screen.getByTestId('wizard-select-fleet'));
    expect(screen.queryByTestId('wizard-start-as-is-button')).not.toBeInTheDocument();
  });

  it('says so when editing the generator turns a pack into a custom fleet', async () => {
    const user = userEvent.setup();
    render(<NewSimulationWizardPage />, { wrapper });

    await waitFor(() => expect(screen.getByTestId('wizard-interface-select')).not.toBeDisabled());
    await user.click(await screen.findByTestId('scenario-pack-hospital'));
    expect(screen.queryByTestId('fleet-customized-notice')).not.toBeInTheDocument();

    await user.click(screen.getByTestId('fleet-customize'));
    fireEvent.change(screen.getByTestId('fleet-domain'), { target: { value: 'ward.example' } });
    expect(screen.getByTestId('fleet-customized-notice')).toHaveTextContent(
      'Customized from Hospital network. This is now a custom fleet.',
    );

    await user.click(screen.getByTestId('scenario-pack-hospital'));
    expect(screen.queryByTestId('fleet-customized-notice')).not.toBeInTheDocument();
  });

  // AUD-6.
  describe('interface choice', () => {
    const interfaces = {
      interfaces: [
        { name: 'lo', description: '', addresses: ['127.0.0.1'] },
        { name: 'eth0', description: '', addresses: ['10.0.0.5'] },
        { name: 'eth1', description: '', addresses: ['10.0.1.5'] },
      ],
    };

    beforeEach(() => {
      fetchUsableInterfaces.mockResolvedValue(interfaces);
    });

    it('defaults to the most recent run and hides loopback', async () => {
      fetchHistory.mockResolvedValue([
        { id: 1, startedAt: '2026-11-18T09:00:00Z', interface: 'eth0' },
        { id: 3, startedAt: '2026-11-20T09:00:00Z', interface: 'eth1' },
        { id: 2, startedAt: '2026-11-19T09:00:00Z', interface: 'lo' },
      ]);
      render(<NewSimulationWizardPage />, { wrapper });

      const select = await screen.findByTestId('wizard-interface-select');
      await waitFor(() => expect(select).toHaveValue('eth1'));
      expect(screen.queryByRole('option', { name: /^lo\b/ })).not.toBeInTheDocument();
      expect(screen.getAllByRole('option').map((o) => o.getAttribute('value'))).toEqual([
        '',
        'eth0',
        'eth1',
      ]);
    });

    it('leaves the choice empty when no recorded run matches an offered interface', async () => {
      fetchHistory.mockResolvedValue([
        { id: 1, startedAt: '2026-11-18T09:00:00Z', interface: 'en9' },
      ]);
      render(<NewSimulationWizardPage />, { wrapper });

      const select = await screen.findByTestId('wizard-interface-select');
      await waitFor(() => expect(select).not.toBeDisabled());
      expect(select).toHaveValue('');
    });

    it('still lists the interfaces when run history cannot be read', async () => {
      fetchHistory.mockRejectedValue(new Error('history unavailable'));
      render(<NewSimulationWizardPage />, { wrapper });

      const select = await screen.findByTestId('wizard-interface-select');
      await waitFor(() => expect(select).not.toBeDisabled());
      expect(select).toHaveValue('');
    });

    it('names the running session that already holds the chosen interface', async () => {
      const user = userEvent.setup();
      fetchSimulationStatus.mockResolvedValue({
        ...idleStatus,
        running: true,
        sessions: [
          {
            running: true,
            sessionId: 'hospital-1',
            interface: 'eth0',
            deviceCount: 3,
            uptimeSeconds: 5,
          },
          {
            running: true,
            sessionId: 'campus-2',
            interface: 'eth1',
            deviceCount: 3,
            uptimeSeconds: 5,
          },
        ],
      });
      render(<NewSimulationWizardPage />, { wrapper });

      const select = await screen.findByTestId('wizard-interface-select');
      await waitFor(() => expect(select).not.toBeDisabled());
      expect(screen.queryByTestId('wizard-interface-in-use')).not.toBeInTheDocument();

      await user.selectOptions(select, 'eth0');
      expect(screen.getByTestId('wizard-interface-in-use')).toHaveTextContent(
        'hospital-1 is already running on eth0.',
      );
      await user.selectOptions(select, 'eth1');
      expect(screen.getByTestId('wizard-interface-in-use')).toHaveTextContent(
        'campus-2 is already running on eth1.',
      );
    });
  });
});
