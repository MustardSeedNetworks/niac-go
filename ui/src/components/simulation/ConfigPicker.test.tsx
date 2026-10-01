import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { BuiltinScenario } from '../../api/builtin-scenario-types';
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

const fetchBuiltinScenarios = vi.fn();
const fetchLibraryNetworks = vi.fn();
const fetchBuiltinScenarioContent = vi.fn();
const importConfig = vi.fn();
const copyToClipboard = vi.fn();

vi.mock('../../api/client', () => ({
  fetchBuiltinScenarios: () => fetchBuiltinScenarios(),
  fetchBuiltinScenarioContent: (name: string) => fetchBuiltinScenarioContent(name),
  importConfig: (...args: unknown[]) => importConfig(...args),
}));
vi.mock('../../api/library-client', () => ({
  fetchLibraryNetworks: () => fetchLibraryNetworks(),
}));

vi.mock('../../utils/file', () => ({
  copyToClipboard: (value: string) => copyToClipboard(value),
}));

// ScenarioPreviewModal renders the CodeMirror-backed YamlViewer, which
// requires a real ResizeObserver constructor that jsdom/the shared test
// setup doesn't provide. Stub it out — this test only cares about the
// Copy YAML wiring, not the editor widget.
vi.mock('../config/YamlEditor', () => ({
  YamlViewer: ({ value }: { value: string }) => <pre>{value}</pre>,
}));

const builtin: BuiltinScenario = {
  name: 'basic-router',
  description: 'A basic router',
  deviceCount: 1,
  type: 'router',
};

describe('ConfigPicker', () => {
  beforeEach(() => {
    fetchBuiltinScenarios.mockReset().mockResolvedValue([builtin]);
    fetchLibraryNetworks.mockReset().mockResolvedValue([]);
    fetchBuiltinScenarioContent.mockReset();
    importConfig.mockReset();
    copyToClipboard.mockReset().mockResolvedValue(undefined);
  });

  it('copies the previewed built-in scenario YAML to the clipboard instead of no-oping', async () => {
    const user = userEvent.setup();
    fetchBuiltinScenarioContent.mockResolvedValue({
      name: builtin.name,
      content: 'devices:\n  - name: r1\n',
      format: 'yaml',
    });

    render(
      <MemoryRouter>
        <ConfigPicker
          selection={{ source: null, name: '' }}
          onSelectBuiltin={vi.fn()}
          onSelectUserConfig={vi.fn()}
          onUpload={vi.fn()}
          uploadFile={null}
        />
      </MemoryRouter>,
    );

    await user.click(await screen.findByRole('button', { name: 'Preview YAML' }));
    await user.click(await screen.findByRole('button', { name: /Copy YAML/i }));

    expect(copyToClipboard).toHaveBeenCalledWith('devices:\n  - name: r1\n');
  });

  it('searches names, display labels, vendors and tags together with the device family', async () => {
    const user = userEvent.setup();
    const builtins: BuiltinScenario[] = [
      {
        name: 'edge-a',
        displayName: 'Campus core',
        description: '',
        vendor: 'Acme',
        type: 'switch',
        tags: ['distribution'],
        deviceCount: 1,
      },
      { name: 'edge-b', description: '', vendor: 'Other', type: 'router', deviceCount: 1 },
    ];
    fetchBuiltinScenarios.mockResolvedValue(builtins);
    const select = vi.fn();
    render(
      <MemoryRouter>
        <ConfigPicker
          selection={{ source: null, name: '' }}
          onSelectBuiltin={select}
          onSelectUserConfig={vi.fn()}
          onUpload={vi.fn()}
          uploadFile={null}
          filterByDeviceFamily
        />
      </MemoryRouter>,
    );
    await screen.findByTestId('config-item-builtin:edge-a');
    expect(screen.getAllByTestId(/^config-item-builtin:/)).toHaveLength(2);
    const search = screen.getByTestId('config-picker-search');
    for (const query of ['CAMPUS', 'Acme', 'distribution', 'edge-a']) {
      await user.clear(search);
      await user.type(search, query);
      expect(screen.getAllByTestId(/^config-item-builtin:/)).toHaveLength(1);
      expect(screen.getByTestId('config-item-builtin:edge-a')).toBeVisible();
    }
    await user.clear(search);
    const family = screen.getByTestId('config-picker-family');
    await user.selectOptions(family, 'router');
    expect(screen.getAllByTestId(/^config-item-builtin:/)).toHaveLength(1);
    expect(screen.getByTestId('config-item-builtin:edge-b')).toBeVisible();
    await user.type(search, 'Acme');
    expect(screen.queryAllByTestId(/^config-item-builtin:/)).toHaveLength(0);
    await user.selectOptions(family, 'switch');
    await user.click(
      within(screen.getByTestId('config-item-builtin:edge-a')).getByRole('button', {
        name: 'Select',
      }),
    );
    expect(select).toHaveBeenCalledExactlyOnceWith(builtins[0]);
  });
});
