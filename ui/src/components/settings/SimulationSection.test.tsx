/**
 * One failed fetch must read as a failure, not as "you have none" (#2177).
 *
 * The three lists this section shows — usable interfaces, templates and saved
 * configs — used to load through one `Promise.all` whose only failure handling
 * was `console.error`, so a daemon 500, a timeout or an expired token rendered
 * the empty-state copy of all three at once.
 */

import { screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fetchTemplates, fetchUsableInterfaces } from '../../api/client';
import { fetchLibraryNetworks } from '../../api/library-client';
import type { InterfacesResponse, LibraryNetwork, Template } from '../../api/types';
import { renderWithResources } from '../../test/renderWithResources';
import { SimulationSection } from './SimulationSection';

vi.mock('../../api/client', () => ({
  fetchTemplates: vi.fn(),
  fetchUsableInterfaces: vi.fn(),
}));
vi.mock('../../api/library-client', () => ({
  fetchLibraryNetworks: vi.fn(),
}));

const interfaces: InterfacesResponse = {
  interfaces: [{ name: 'eth0', description: 'Ethernet', addresses: ['10.0.0.5/24'] }],
};

const templates: Template[] = [
  { name: 'hospital', description: 'Hospital pack', deviceCount: 12, type: 'basic' },
];

const userConfigs: LibraryNetwork[] = [
  {
    name: 'my-lab',
    deviceCount: 3,
    modifiedAt: '2026-09-17T00:00:00Z',
    sizeBytes: 1024,
    source: 'user',
    valid: true,
  },
];

const mocks = {
  interfaces: vi.mocked(fetchUsableInterfaces),
  templates: vi.mocked(fetchTemplates),
  userConfigs: vi.mocked(fetchLibraryNetworks),
};

function resolveAll(): void {
  mocks.interfaces.mockResolvedValue(interfaces);
  mocks.templates.mockResolvedValue(templates);
  mocks.userConfigs.mockResolvedValue(userConfigs);
}

describe('SimulationSection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    resolveAll();
  });

  it('lists what each fetch returned when all three succeed', async () => {
    renderWithResources(<SimulationSection />);

    expect(await screen.findByRole('option', { name: /eth0/ })).toBeInTheDocument();
    expect(await screen.findByText('hospital')).toBeInTheDocument();
    expect(screen.queryByText('No templates available')).not.toBeInTheDocument();
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('shows a load failure for the interfaces, not an empty picker', async () => {
    mocks.interfaces.mockRejectedValue(new Error('daemon unreachable'));
    renderWithResources(<SimulationSection />);

    const alert = await screen.findByTestId('simulation-interfaces-error');
    expect(alert).toHaveTextContent(/daemon unreachable/);
    // The other two fetches succeeded and must still render.
    expect(await screen.findByText('hospital')).toBeInTheDocument();
  });

  it('shows a load failure for the templates, not "no templates available"', async () => {
    mocks.templates.mockRejectedValue(new Error('500 internal error'));
    renderWithResources(<SimulationSection />);

    const alert = await screen.findByTestId('simulation-templates-error');
    expect(alert).toHaveTextContent(/500 internal error/);
    expect(screen.queryByText('No templates available')).not.toBeInTheDocument();
    expect(await screen.findByRole('option', { name: /eth0/ })).toBeInTheDocument();
  });

  it('shows a load failure for the saved configs, not "none uploaded yet"', async () => {
    mocks.userConfigs.mockRejectedValue(new Error('token expired'));
    renderWithResources(<SimulationSection />);

    await screen.findByText('hospital');
    screen.getByRole('tab', { name: 'My Configs' }).click();

    const alert = await screen.findByTestId('simulation-configs-error');
    expect(alert).toHaveTextContent(/token expired/);
    expect(screen.queryByText('No user configs uploaded yet')).not.toBeInTheDocument();
  });

  it('retries only the resource that failed', async () => {
    mocks.templates.mockRejectedValueOnce(new Error('500 internal error'));
    renderWithResources(<SimulationSection />);

    (await screen.findByTestId('simulation-templates-retry')).click();

    await waitFor(() => expect(screen.getByText('hospital')).toBeInTheDocument());
    expect(mocks.templates).toHaveBeenCalledTimes(2);
    expect(mocks.interfaces).toHaveBeenCalledTimes(1);
  });
});
