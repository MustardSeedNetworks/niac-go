import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { LibraryNetwork } from '../../api/types';
import '../../i18n';
import { ConfigPicker } from './ConfigPicker';

// ConfigPicker reads/writes view-mode + favorites prefs via localStorage
// (useFavorites, readPref/writePref); jsdom in this project doesn't wire
// up a usable localStorage by default (see device-store.test.ts), so mock it.
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

Object.defineProperty(globalThis, 'localStorage', {
  value: localStorageMock,
  writable: true,
});

// This isolated component fixture represents an authenticated operator.
vi.mock('../../contexts/ScopeContext', () => ({
  useActionPermission: () => ({ disabled: false }),
}));

const fetchLibraryNetworks = vi.fn();
const duplicateLibraryNetwork = vi.fn();
const importConfig = vi.fn();

vi.mock('../../api/client', () => ({
  importConfig: (...args: unknown[]) => importConfig(...args),
}));
vi.mock('../../api/library-client', () => ({
  fetchLibraryNetworks: () => fetchLibraryNetworks(),
  duplicateLibraryNetwork: (...args: unknown[]) => duplicateLibraryNetwork(...args),
}));

const network = (
  name: string,
  description: string,
  source: LibraryNetwork['source'] = 'user',
): LibraryNetwork => ({
  name,
  description,
  deviceCount: 1,
  modifiedAt: '2026-10-04T00:00:00Z',
  sizeBytes: 64,
  source,
  valid: true,
});

const renderPicker = (select = vi.fn()) =>
  render(
    <MemoryRouter>
      <ConfigPicker
        selection={{ source: null, name: '' }}
        onSelectUserConfig={select}
        onUpload={vi.fn()}
        uploadFile={null}
      />
    </MemoryRouter>,
  );

describe('ConfigPicker', () => {
  beforeEach(() => {
    fetchLibraryNetworks.mockReset();
    duplicateLibraryNetwork.mockReset();
    importConfig.mockReset();
  });

  // The library is the one list of networks, shipped examples included: there is no second,
  // built-in listing that an installed host could not fill (#2131).
  it('lists the library networks, searches names and descriptions, and selects one', async () => {
    const user = userEvent.setup();
    const networks = [
      network('small-office', 'Branch office'),
      network('data-center', 'Spine and leaf'),
    ];
    fetchLibraryNetworks.mockResolvedValue(networks);
    const select = vi.fn();
    render(
      <MemoryRouter>
        <ConfigPicker
          selection={{ source: null, name: '' }}
          onSelectUserConfig={select}
          onUpload={vi.fn()}
          uploadFile={null}
        />
      </MemoryRouter>,
    );
    await screen.findByTestId('config-item-saved:small-office');
    expect(screen.getAllByTestId(/^config-item-/)).toHaveLength(2);
    expect(screen.queryByTestId('config-picker-family')).not.toBeInTheDocument();

    const search = screen.getByTestId('config-picker-search');
    for (const query of ['SMALL', 'branch']) {
      await user.clear(search);
      await user.type(search, query);
      expect(screen.getAllByTestId(/^config-item-/)).toHaveLength(1);
      expect(screen.getByTestId('config-item-saved:small-office')).toBeVisible();
    }
    await user.click(
      within(screen.getByTestId('config-item-saved:small-office')).getByRole('button', {
        name: 'Select',
      }),
    );
    expect(select).toHaveBeenCalledExactlyOnceWith(networks[0]);
  });

  describe('shipped examples', () => {
    const mixed = () => [
      network('basic-network', 'Router and switch', 'example'),
      network('my-lab', 'Mine'),
    ];

    it('badges an example and offers Duplicate on it alone', async () => {
      fetchLibraryNetworks.mockResolvedValue(mixed());
      renderPicker();

      await screen.findByTestId('config-item-saved:basic-network');
      expect(screen.getByTestId('config-example-badge-basic-network')).toHaveTextContent('Example');
      expect(screen.getByTestId('config-duplicate-basic-network')).toBeEnabled();
      expect(screen.queryByTestId('config-example-badge-my-lab')).not.toBeInTheDocument();
      expect(screen.queryByTestId('config-duplicate-my-lab')).not.toBeInTheDocument();
    });

    it('still selects an example like any other network', async () => {
      const user = userEvent.setup();
      const networks = mixed();
      fetchLibraryNetworks.mockResolvedValue(networks);
      const select = vi.fn();
      renderPicker(select);

      await screen.findByTestId('config-item-saved:basic-network');
      await user.click(
        within(screen.getByTestId('config-item-saved:basic-network')).getByRole('button', {
          name: 'Select',
        }),
      );
      expect(select).toHaveBeenCalledExactlyOnceWith(networks[0]);
    });

    it('duplicates server-side, reloads the list and reports the saved name', async () => {
      const user = userEvent.setup();
      fetchLibraryNetworks
        .mockResolvedValueOnce(mixed())
        .mockResolvedValueOnce([...mixed(), network('basic-network-copy', 'Router and switch')]);
      duplicateLibraryNetwork.mockResolvedValue({ success: true, name: 'basic-network-copy' });
      renderPicker();

      await user.click(await screen.findByTestId('config-duplicate-basic-network'));

      expect(duplicateLibraryNetwork).toHaveBeenCalledExactlyOnceWith({
        copyFrom: 'basic-network',
      });
      expect(await screen.findByTestId('config-item-saved:basic-network-copy')).toBeVisible();
      expect(screen.getByTestId('config-duplicate-saved')).toHaveTextContent('basic-network-copy');
      expect(screen.queryByTestId('config-duplicate-basic-network-copy')).not.toBeInTheDocument();
    });

    it('reports a failed duplicate and keeps the list as it was', async () => {
      const user = userEvent.setup();
      fetchLibraryNetworks.mockResolvedValue(mixed());
      duplicateLibraryNetwork.mockRejectedValue(new Error('already exists'));
      renderPicker();

      await user.click(await screen.findByTestId('config-duplicate-basic-network'));

      expect(await screen.findByTestId('config-duplicate-failed')).toHaveTextContent(
        'already exists',
      );
      expect(screen.getAllByTestId(/^config-item-/)).toHaveLength(2);
      expect(fetchLibraryNetworks).toHaveBeenCalledTimes(1);
    });

    it('offers Duplicate in the compact list too', async () => {
      const user = userEvent.setup();
      fetchLibraryNetworks.mockResolvedValue(mixed());
      renderPicker();

      await screen.findByTestId('config-item-saved:basic-network');
      await user.click(screen.getByRole('button', { name: 'Compact list' }));

      expect(screen.getByTestId('config-example-badge-basic-network')).toBeVisible();
      expect(screen.getByTestId('config-duplicate-basic-network')).toBeVisible();
    });
  });
});
