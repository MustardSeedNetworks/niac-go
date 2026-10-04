/**
 * One failed fetch must read as a failure, not as "you have none" (#2177).
 *
 * The lists this section shows — usable interfaces and saved configs — used to
 * load through one `Promise.all` whose only failure handling was
 * `console.error`, so a daemon 500, a timeout or an expired token rendered the
 * empty-state copy of every list at once.
 */

import { fireEvent, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fetchUsableInterfaces } from '../../api/client';
import { fetchLibraryNetworks } from '../../api/library-client';
import type { InterfacesResponse, LibraryNetwork } from '../../api/types';
import { useUIStore } from '../../stores/ui-store';
import { renderWithResources } from '../../test/renderWithResources';
import { SimulationSection } from './SimulationSection';

vi.mock('../../api/client', () => ({
  fetchUsableInterfaces: vi.fn(),
}));
vi.mock('../../api/library-client', () => ({
  fetchLibraryNetworks: vi.fn(),
}));

const interfaces: InterfacesResponse = {
  interfaces: [{ name: 'eth0', description: 'Ethernet', addresses: ['10.0.0.5/24'] }],
};

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
  userConfigs: vi.mocked(fetchLibraryNetworks),
};

/**
 * The picker stays disabled until the interfaces resource settles, so its
 * enabled state is the ready signal. Waiting on it with a test-id lookup keeps
 * role queries out of the timed wait: the first `*ByRole` call in a worker
 * spends ~135 ms initialising its accessibility tree, and under full-suite load
 * that cold start alone used to exhaust `findByRole`'s 1 s default (#2439).
 */
async function interfacesLoaded(): Promise<void> {
  await waitFor(() => expect(screen.getByTestId('simulation-interface')).toBeEnabled());
}

function resolveAll(): void {
  mocks.interfaces.mockResolvedValue(interfaces);
  mocks.userConfigs.mockResolvedValue(userConfigs);
}

describe('SimulationSection', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // The store is a module singleton: without this, the tab one test selects
    // is the tab the next test opens on.
    useUIStore.getState().resetSimulationSettings();
    resolveAll();
  });

  it('lists what each fetch returned when both succeed', async () => {
    renderWithResources(<SimulationSection />);

    await interfacesLoaded();
    expect(screen.getByRole('option', { name: /eth0/ })).toBeInTheDocument();
    expect(await screen.findByText('my-lab')).toBeInTheDocument();
    expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  });

  it('selects and clears the interface with the shared picker', async () => {
    renderWithResources(<SimulationSection />);
    await interfacesLoaded();
    const picker = screen.getByRole('combobox', { name: 'Network Interface' });
    fireEvent.change(picker, { target: { value: 'eth0' } });
    expect(useUIStore.getState().simulationSettings.selectedInterface).toBe('eth0');
    fireEvent.change(picker, { target: { value: '' } });
    expect(useUIStore.getState().simulationSettings.selectedInterface).toBe('');
    expect(picker).toHaveClass('appearance-none', 'min-h-11');
  });

  it('shows a load failure for the interfaces, not an empty picker', async () => {
    mocks.interfaces.mockRejectedValue(new Error('daemon unreachable'));
    renderWithResources(<SimulationSection />);

    const alert = await screen.findByTestId('simulation-interfaces-error');
    expect(alert).toHaveTextContent(/daemon unreachable/);
    // The saved-configs fetch succeeded and must still render.
    expect(await screen.findByText('my-lab')).toBeInTheDocument();
  });

  it('shows a load failure for the saved configs, not "none uploaded yet"', async () => {
    mocks.userConfigs.mockRejectedValue(new Error('token expired'));
    renderWithResources(<SimulationSection />);

    const alert = await screen.findByTestId('simulation-configs-error');
    expect(alert).toHaveTextContent(/token expired/);
    expect(screen.queryByText('No user configs uploaded yet')).not.toBeInTheDocument();
    await interfacesLoaded();
    expect(screen.getByRole('option', { name: /eth0/ })).toBeInTheDocument();
  });

  it('retries only the resource that failed', async () => {
    mocks.userConfigs.mockRejectedValueOnce(new Error('500 internal error'));
    renderWithResources(<SimulationSection />);

    fireEvent.click(await screen.findByTestId('simulation-configs-retry'));

    await waitFor(() => expect(screen.getByText('my-lab')).toBeInTheDocument());
    expect(mocks.userConfigs).toHaveBeenCalledTimes(2);
    expect(mocks.interfaces).toHaveBeenCalledTimes(1);
  });

  describe('config-source tabs', () => {
    // The switcher declared `role="tablist"` over plain buttons: a screen
    // reader was promised tabs that did not exist, and the keyboard had no way
    // between them (#2242). The library is the one list of starter networks,
    // so there is no separate Built-in tab (#2131).
    const tabNames = ['My Configs', 'Upload'];
    const tab = (name: string) => screen.getByRole('tab', { name });

    it('exposes the tabs, one selected, controlling one panel', async () => {
      renderWithResources(<SimulationSection />);
      await screen.findByText('my-lab');

      expect(screen.getAllByRole('tab').map((tab) => tab.textContent)).toEqual(tabNames);
      expect(screen.getAllByRole('tab').map((tab) => tab.getAttribute('aria-selected'))).toEqual([
        'true',
        'false',
      ]);

      const configs = tab('My Configs');
      const panel = screen.getByRole('tabpanel');
      for (const sourceTab of screen.getAllByRole('tab')) {
        expect(sourceTab).toHaveAttribute('aria-controls', panel.id);
      }
      expect(panel).toHaveAttribute('aria-labelledby', configs.id);

      // The pairing has to follow the selection, not name one tab forever.
      const upload = tab('Upload');
      fireEvent.click(upload);
      const uploadPanel = screen.getByRole('tabpanel');
      expect(uploadPanel).toHaveAttribute('aria-labelledby', upload.id);
      expect(upload).toHaveAttribute('aria-controls', uploadPanel.id);
    });

    it('moves selection with the arrow keys and wraps, per the APG tab pattern', async () => {
      renderWithResources(<SimulationSection />);
      await screen.findByText('my-lab');
      const configs = tab('My Configs');
      const upload = tab('Upload');

      configs.focus();
      fireEvent.keyDown(configs, { key: 'ArrowRight' });
      expect(upload).toHaveAttribute('aria-selected', 'true');
      expect(upload).toHaveFocus();

      fireEvent.keyDown(upload, { key: 'ArrowRight' });
      expect(configs).toHaveAttribute('aria-selected', 'true');
      expect(configs).toHaveFocus();

      fireEvent.keyDown(configs, { key: 'ArrowLeft' });
      expect(upload).toHaveAttribute('aria-selected', 'true');

      fireEvent.keyDown(upload, { key: 'Home' });
      expect(configs).toHaveAttribute('aria-selected', 'true');

      fireEvent.keyDown(configs, { key: 'End' });
      expect(upload).toHaveAttribute('aria-selected', 'true');
    });

    it('keeps one tab stop for the whole tablist (roving tabindex)', async () => {
      renderWithResources(<SimulationSection />);
      await screen.findByText('my-lab');

      expect(screen.getAllByRole('tab').map((tab) => tab.tabIndex)).toEqual([0, -1]);

      fireEvent.click(screen.getByRole('tab', { name: 'Upload' }));
      expect(screen.getAllByRole('tab').map((tab) => tab.tabIndex)).toEqual([-1, 0]);
    });
  });
});
