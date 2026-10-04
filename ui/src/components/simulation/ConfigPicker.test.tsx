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

const fetchLibraryNetworks = vi.fn();
const importConfig = vi.fn();

vi.mock('../../api/client', () => ({
  importConfig: (...args: unknown[]) => importConfig(...args),
}));
vi.mock('../../api/library-client', () => ({
  fetchLibraryNetworks: () => fetchLibraryNetworks(),
}));

const network = (name: string, description: string): LibraryNetwork => ({
  name,
  description,
  deviceCount: 1,
  modifiedAt: '2026-10-04T00:00:00Z',
  sizeBytes: 64,
  source: 'starter',
  valid: true,
});

describe('ConfigPicker', () => {
  beforeEach(() => {
    fetchLibraryNetworks.mockReset();
    importConfig.mockReset();
  });

  // The library is the one list of starter networks: there is no second,
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
});
