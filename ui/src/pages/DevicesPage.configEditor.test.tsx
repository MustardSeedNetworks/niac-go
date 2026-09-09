// This isolated component fixture represents an authenticated operator.
vi.mock('../contexts/ScopeContext', () => ({
  useActionPermission: () => ({ disabled: false }),
}));

/**
 * DevicesPage.configEditor.test.tsx — Phase 5d structured YAML parse errors.
 *
 * When the backend rejects a config save with a structured error detail
 * (`{issue, line}`, see internal/api/yaml_errors.go), the editor card must:
 *   1. Surface a line-numbered message instead of the generic error text.
 *   2. Pass the line down to YamlEditor so it can highlight/scroll to it.
 *
 * YamlEditor's own line-highlight rendering (CodeMirror decorations) is
 * covered by src/components/config/YamlEditor.test.tsx; this file only
 * pins the DevicesPage <-> ApiError.details wiring, so YamlEditor is
 * mocked to a prop-capturing stub (same pattern as ConfigPicker.test.tsx).
 */
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '../api/errors';
import { MemoryDataRouter } from '../test/MemoryDataRouter';
import { renderWithResources as render } from '../test/renderWithResources';
import '../i18n';
import { fetchDevices } from '../api/client';
import { useApiResource } from '../hooks/useApiResource';
import { DevicesPage } from './DevicesPage';

const updateConfig = vi.fn();
const configDocument = {
  content: 'devices: []',
  path: '/tmp/config.yaml',
  modifiedAt: '2026-01-01T00:00:00Z',
  sizeBytes: 12,
};
const fetchConfig = vi.fn();

// Runtime reads name their session, so a page rendered on its own has to say
// which scenario it is looking at.
vi.mock('../contexts/AppContext', () => ({
  useAppState: () =>
    useApiResource(() => fetchDevices('test-session'), ['devices', 'test-session']),
}));

vi.mock('../api/client', () => ({
  fetchDevices: () => Promise.resolve([]),
  fetchConfig: () => fetchConfig(),
  updateConfig: (...args: unknown[]) => updateConfig(...args),
}));
vi.mock('../api/library-client', () => ({
  fetchLibraryWalks: () => Promise.resolve([]),
}));

let latestErrorLine: number | null | undefined;

vi.mock('../components/config/YamlEditor', () => ({
  YamlEditor: ({
    value,
    onChange,
    errorLine,
  }: {
    value: string;
    onChange?: (v: string) => void;
    errorLine?: number | null;
  }) => {
    latestErrorLine = errorLine;
    return (
      <textarea
        aria-label="yaml-editor-stub"
        value={value}
        onChange={(e) => onChange?.(e.target.value)}
      />
    );
  },
}));

describe('DevicesPage — config editor structured parse errors', () => {
  beforeEach(() => {
    updateConfig.mockReset();
    fetchConfig.mockReset().mockResolvedValue(configDocument);
    latestErrorLine = undefined;
  });

  it('does not let a read started before save replace the accepted configuration', async () => {
    const client = new QueryClient();
    const user = userEvent.setup();
    const { unmount } = render(
      <QueryClientProvider client={client}>
        <MemoryDataRouter>
          <DevicesPage />
        </MemoryDataRouter>
      </QueryClientProvider>,
    );
    let resolveRead: (value: typeof configDocument) => void = () => {};
    const oldRead = new Promise<typeof configDocument>((resolve) => {
      resolveRead = resolve;
    });
    try {
      const editor = await screen.findByLabelText('yaml-editor-stub');
      await waitFor(() => expect(editor).toHaveValue(configDocument.content));
      fetchConfig.mockReturnValueOnce(oldRead);
      let refresh: Promise<void> = Promise.resolve();
      act(() => {
        refresh = client.refetchQueries({ queryKey: ['config'], exact: true });
      });
      await waitFor(() => expect(fetchConfig).toHaveBeenCalledTimes(2));
      await user.type(editor, '\n# saved');
      const accepted = { ...configDocument, content: 'devices: []\n# saved' };
      updateConfig.mockResolvedValue(accepted);
      await user.click(screen.getByRole('button', { name: /save & reload simulation/i }));
      await waitFor(() => expect(client.getQueryData(['config'])).toEqual(accepted));
      act(() => resolveRead(configDocument));
      await oldRead;
      await refresh;
      expect(client.getQueryData(['config'])).toEqual(accepted);
      expect(editor).toHaveValue(accepted.content);
    } finally {
      resolveRead(configDocument);
      unmount();
      client.clear();
    }
  });

  it('surfaces the line-numbered message and passes the line to the editor on save failure', async () => {
    updateConfig.mockRejectedValue(
      new ApiError('Configuration validation failed', 400, 'config_invalid', [
        { issue: 'line 3: mapping values are not allowed in this context', line: 3 },
      ]),
    );

    const user = userEvent.setup();
    render(
      <MemoryDataRouter>
        <DevicesPage />
      </MemoryDataRouter>,
    );

    const editor = await screen.findByLabelText('yaml-editor-stub');
    await user.type(editor, 'x');

    const saveButton = await screen.findByRole('button', { name: /save & reload simulation/i });
    await user.click(saveButton);

    expect(
      await screen.findByText('Line 3: line 3: mapping values are not allowed in this context'),
    ).toBeInTheDocument();
    expect(latestErrorLine).toBe(3);
  });

  it('clears the error line once the operator edits the content again', async () => {
    updateConfig.mockRejectedValue(
      new ApiError('Configuration validation failed', 400, 'config_invalid', [
        { issue: 'boom', line: 5 },
      ]),
    );

    const user = userEvent.setup();
    render(
      <MemoryDataRouter>
        <DevicesPage />
      </MemoryDataRouter>,
    );

    const editor = await screen.findByLabelText('yaml-editor-stub');
    await user.type(editor, 'x');
    const saveButton = await screen.findByRole('button', { name: /save & reload simulation/i });
    await user.click(saveButton);

    expect(await screen.findByText('Line 5: boom')).toBeInTheDocument();
    expect(latestErrorLine).toBe(5);

    await user.type(editor, 'y');
    expect(latestErrorLine).toBeNull();
  });
});
