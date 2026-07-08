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
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '../api/errors';
import '../i18n';
import { DevicesPage } from './DevicesPage';

const updateConfig = vi.fn();

vi.mock('../api/client', () => ({
  fetchDevices: () => Promise.resolve([]),
  fetchConfig: () =>
    Promise.resolve({
      content: 'devices: []',
      path: '/tmp/config.yaml',
      modifiedAt: '2026-01-01T00:00:00Z',
      sizeBytes: 12,
    }),
  fetchLibraryWalks: () => Promise.resolve([]),
  updateConfig: (...args: unknown[]) => updateConfig(...args),
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
    latestErrorLine = undefined;
  });

  it('surfaces the line-numbered message and passes the line to the editor on save failure', async () => {
    updateConfig.mockRejectedValue(
      new ApiError('Configuration validation failed', 400, 'config_invalid', [
        { issue: 'line 3: mapping values are not allowed in this context', line: 3 },
      ]),
    );

    const user = userEvent.setup();
    render(
      <MemoryRouter>
        <DevicesPage />
      </MemoryRouter>,
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
      <MemoryRouter>
        <DevicesPage />
      </MemoryRouter>,
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
