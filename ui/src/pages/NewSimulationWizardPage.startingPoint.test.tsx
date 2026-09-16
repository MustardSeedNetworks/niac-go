// This isolated component fixture represents an authenticated operator.
vi.mock('../contexts/ScopeContext', () => ({
  useActionPermission: () => ({ disabled: false }),
}));

/**
 * NewSimulationWizardPage.startingPoint.test.tsx
 *
 * The template step offers several ways into the same wizard -- a scenario
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

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    // AppProvider's background polls — kept quiet for this fixture.
    fetchSimulationStatus: vi
      .fn()
      .mockResolvedValue({ running: false, interface: '', deviceCount: 0, uptimeSeconds: 0 }),
    fetchStats: vi.fn(),
    fetchDevices: vi.fn().mockResolvedValue([]),
    fetchHistory: vi.fn(),
    fetchNeighbors: vi.fn(),
    fetchVersion: vi.fn(),
    fetchErrorTypes: vi.fn(),
    fetchInterfaces: vi.fn(),
    fetchTemplates: vi.fn().mockResolvedValue([]),
    fetchUsableInterfaces: vi.fn().mockResolvedValue({
      interfaces: [{ name: 'lo0', addresses: ['127.0.0.1'], isUp: true, isLoopback: true }],
    }),
  };
});

vi.mock('../api/library-client', () => ({
  fetchLibraryNetworks: vi.fn().mockResolvedValue([]),
  createScenarioDraft: vi.fn(),
  createScenarioDraftFromTemplate: vi.fn(),
  replaceScenarioDraft: vi.fn(),
  deleteScenarioDraft: vi.fn(),
}));

const fetchScenarioPacks = vi.hoisted(() => vi.fn());

vi.mock('../api/scenario-client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/scenario-client')>();
  return { ...actual, fetchScenarioPacks };
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
});
