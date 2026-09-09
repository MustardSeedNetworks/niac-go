// This isolated component fixture represents an authenticated operator.
vi.mock('../contexts/ScopeContext', () => ({
  useActionPermission: () => ({ disabled: false }),
}));

/**
 * DevicesPage.deviceDetail.test.tsx — the list + detail behaviour.
 *
 * The defect these cover is that selection used to mean nothing: the device
 * list and the YAML editor were two cards that did not talk to each other, so
 * the only way to change one device was to find it by eye in the whole config.
 * Selecting a device now opens that device's own block, and saving splices it
 * back into the config — which is still the only thing the daemon accepts.
 */
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryDataRouter } from '../test/MemoryDataRouter';
import { renderWithResources as render } from '../test/renderWithResources';
import '../i18n';
import { fetchDevices } from '../api/client';
import { ApiError } from '../api/errors';
import { POLL_INTERVALS } from '../constants/polling';
import { useApiResource } from '../hooks/useApiResource';
import { DevicesPage } from './DevicesPage';

const CONFIG = `# operator's note, must survive an edit
devices:
  - name: api-router
    type: router
    ips:
      - "10.10.0.1"

  - name: core-switch
    type: switch
`;

const updateConfig = vi.fn();
const fetchConfig = vi.fn();

vi.mock('../contexts/AppContext', () => ({
  useAppState: () =>
    useApiResource(() => fetchDevices('test-session'), ['devices', 'test-session']),
}));

vi.mock('../api/client', () => ({
  fetchDevices: () =>
    Promise.resolve([
      { name: 'api-router', type: 'router', ips: ['10.10.0.1'], protocols: ['snmp'] },
      { name: 'core-switch', type: 'switch', ips: [], protocols: [] },
    ]),
  fetchConfig: () => fetchConfig(),
  updateConfig: (...args: unknown[]) => updateConfig(...args),
}));
vi.mock('../api/library-client', () => ({
  fetchLibraryWalks: () => Promise.resolve([]),
}));

vi.mock('../components/config/YamlEditor', () => ({
  YamlEditor: ({ value, onChange }: { value: string; onChange?: (v: string) => void }) => (
    <textarea
      aria-label="yaml-editor-stub"
      value={value}
      onChange={(e) => onChange?.(e.target.value)}
    />
  ),
}));

function renderPage() {
  return render(
    <MemoryDataRouter>
      <DevicesPage />
    </MemoryDataRouter>,
  );
}

const editor = () => screen.getByLabelText('yaml-editor-stub') as HTMLTextAreaElement;

describe('DevicesPage — device detail', () => {
  beforeEach(() => {
    fetchConfig.mockReset();
    fetchConfig.mockResolvedValue({
      content: CONFIG,
      path: '/tmp/config.yaml',
      modifiedAt: '2026-01-01T00:00:00Z',
      sizeBytes: CONFIG.length,
    });
    updateConfig.mockReset();
    updateConfig.mockResolvedValue({
      content: CONFIG,
      path: '/tmp/config.yaml',
      modifiedAt: '2026-01-01T00:00:00Z',
      sizeBytes: CONFIG.length,
    });
  });
  afterEach(() => vi.useRealTimers());

  it.each([
    new Error('Config poll failed'),
    new ApiError('Config poll failed', 400, 'config_read_failed'),
  ])('keeps dirty edits and their navigation dialog accessible after %s', async (error) => {
    vi.useFakeTimers();
    await Promise.resolve(
      act(async () => {
        renderPage();
      }),
    );
    fireEvent.click(screen.getByTestId('device-select-api-router'));
    const edited = 'name: api-router\ntype: firewall\n';
    fireEvent.change(editor(), { target: { value: edited } });
    fetchConfig.mockRejectedValue(error);
    await Promise.resolve(
      act(async () => {
        await vi.advanceTimersByTimeAsync(POLL_INTERVALS.verySlow);
      }),
    );
    expect(editor().value).toBe(edited);
    expect(screen.getByRole('alert')).toHaveTextContent('Config poll failed');
    fireEvent.click(screen.getByTestId('device-select-core-switch'));
    expect(screen.getByRole('dialog', { name: 'Unsaved changes' })).toBeInTheDocument();
    fireEvent.click(screen.getByTestId('unsaved-cancel'));
    expect(editor().value).toBe(edited);
  });

  it('opens with the whole config, because nothing is selected yet', async () => {
    renderPage();
    await waitFor(() => expect(editor().value).toContain('devices:'));
    expect(editor().value).toContain('core-switch');
  });

  it('shows only the selected device once one is chosen', async () => {
    const user = userEvent.setup();
    renderPage();
    await screen.findByTestId('device-select-api-router');

    await user.click(screen.getByTestId('device-select-api-router'));

    await waitFor(() => expect(editor().value).toContain('name: api-router'));
    expect(editor().value).not.toContain('core-switch');
    expect(editor().value).not.toContain('devices:');
  });

  it('splices the edit back into the whole config, preserving what it did not touch', async () => {
    const user = userEvent.setup();
    renderPage();
    await screen.findByTestId('device-select-api-router');
    await user.click(screen.getByTestId('device-select-api-router'));
    await waitFor(() => expect(editor().value).toContain('name: api-router'));

    await user.clear(editor());
    await user.type(editor(), 'name: api-router{enter}type: firewall');
    await user.click(screen.getByRole('button', { name: /save|reload|guardar/i }));

    await waitFor(() => expect(updateConfig).toHaveBeenCalledTimes(1));
    const sent = updateConfig.mock.calls[0]?.[0] as { content: string };
    expect(sent.content).toContain("# operator's note, must survive an edit");
    expect(sent.content).toContain('  - name: api-router\n    type: firewall');
    expect(sent.content).toContain('  - name: core-switch');
    expect(sent.content).not.toContain('10.10.0.1');
  });

  it('refuses a fragment that does not parse, instead of writing it to the config', async () => {
    const user = userEvent.setup();
    renderPage();
    await screen.findByTestId('device-select-api-router');
    await user.click(screen.getByTestId('device-select-api-router'));
    await waitFor(() => expect(editor().value).toContain('name: api-router'));

    await user.clear(editor());
    await user.type(editor(), 'name: [[unclosed');
    await user.click(screen.getByRole('button', { name: /save|reload|guardar/i }));

    expect(updateConfig).not.toHaveBeenCalled();
  });

  it('refuses a fragment with no name, which would orphan the device', async () => {
    const user = userEvent.setup();
    renderPage();
    await screen.findByTestId('device-select-api-router');
    await user.click(screen.getByTestId('device-select-api-router'));
    await waitFor(() => expect(editor().value).toContain('name: api-router'));

    await user.clear(editor());
    await user.type(editor(), 'type: firewall');
    await user.click(screen.getByRole('button', { name: /save|reload|guardar/i }));

    expect(updateConfig).not.toHaveBeenCalled();
  });

  it('returns to the whole config when the selection is cleared', async () => {
    const user = userEvent.setup();
    renderPage();
    await screen.findByTestId('device-select-api-router');
    await user.click(screen.getByTestId('device-select-api-router'));
    await waitFor(() => expect(editor().value).toContain('name: api-router'));

    await user.click(screen.getByTestId('edit-whole-config'));

    await waitFor(() => expect(editor().value).toContain('devices:'));
    expect(editor().value).toContain('core-switch');
  });

  it.each(['cancel', 'discard', 'save', 'failed-save'])(
    'guards device switching: %s',
    async (choice) => {
      const user = userEvent.setup();
      renderPage();
      await user.click(await screen.findByTestId('device-select-api-router'));
      await waitFor(() => expect(editor().value).toContain('name: api-router'));
      const edited = 'name: api-router\ntype: firewall\n';
      fireEvent.change(editor(), { target: { value: edited } });
      await user.click(screen.getByTestId('device-select-core-switch'));
      expect(await screen.findByRole('dialog', { name: 'Unsaved changes' })).toBeInTheDocument();
      if (choice === 'failed-save') updateConfig.mockRejectedValueOnce(new Error('Save refused'));
      await user.click(screen.getByTestId(`unsaved-${choice === 'failed-save' ? 'save' : choice}`));
      if (choice === 'cancel' || choice === 'failed-save') {
        expect(editor().value).toBe(edited);
        expect(screen.queryByRole('dialog')).toBe(
          choice === 'cancel' ? null : screen.getByRole('dialog'),
        );
      } else {
        await waitFor(() => expect(editor().value).toContain('name: core-switch'));
        expect(updateConfig).toHaveBeenCalledTimes(choice === 'save' ? 1 : 0);
      }
    },
  );
});
