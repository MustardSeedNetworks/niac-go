import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { Template } from '../../api/template-types';
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

const fetchTemplates = vi.fn();
const fetchLibraryNetworks = vi.fn();
const fetchTemplateContent = vi.fn();
const importConfig = vi.fn();
const copyToClipboard = vi.fn();

vi.mock('../../api/client', () => ({
  fetchTemplates: () => fetchTemplates(),
  fetchTemplateContent: (name: string) => fetchTemplateContent(name),
  importConfig: (...args: unknown[]) => importConfig(...args),
}));
vi.mock('../../api/library-client', () => ({
  fetchLibraryNetworks: () => fetchLibraryNetworks(),
}));

vi.mock('../../utils/file', () => ({
  copyToClipboard: (value: string) => copyToClipboard(value),
}));

// TemplatePreviewModal renders the CodeMirror-backed YamlViewer, which
// requires a real ResizeObserver constructor that jsdom/the shared test
// setup doesn't provide. Stub it out — this test only cares about the
// Copy YAML wiring, not the editor widget.
vi.mock('../config/YamlEditor', () => ({
  YamlViewer: ({ value }: { value: string }) => <pre>{value}</pre>,
}));

const template: Template = {
  name: 'basic-router',
  description: 'A basic router',
  deviceCount: 1,
  type: 'router',
};

describe('ConfigPicker — TemplatePreviewModal Copy YAML', () => {
  beforeEach(() => {
    fetchTemplates.mockReset().mockResolvedValue([template]);
    fetchLibraryNetworks.mockReset().mockResolvedValue([]);
    fetchTemplateContent.mockReset();
    importConfig.mockReset();
    copyToClipboard.mockReset().mockResolvedValue(undefined);
  });

  it('copies the previewed template YAML to the clipboard instead of no-oping', async () => {
    const user = userEvent.setup();
    fetchTemplateContent.mockResolvedValue({
      name: template.name,
      content: 'devices:\n  - name: r1\n',
      format: 'yaml',
    });

    render(
      <MemoryRouter>
        <ConfigPicker
          selection={{ source: null, name: '' }}
          onSelectTemplate={vi.fn()}
          onSelectUserConfig={vi.fn()}
          onUpload={vi.fn()}
          uploadFile={null}
        />
      </MemoryRouter>,
    );

    await user.click(await screen.findByTitle('Preview YAML'));
    await user.click(await screen.findByRole('button', { name: /Copy YAML/i }));

    expect(copyToClipboard).toHaveBeenCalledWith('devices:\n  - name: r1\n');
  });
});
