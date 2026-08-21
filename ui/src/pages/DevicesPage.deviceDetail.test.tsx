/**
 * DevicesPage.deviceDetail.test.tsx — the list + detail behaviour.
 *
 * The defect these cover is that selection used to mean nothing: the device
 * list and the YAML editor were two cards that did not talk to each other, so
 * the only way to change one device was to find it by eye in the whole config.
 * Selecting a device now opens that device's own block, and saving splices it
 * back into the config — which is still the only thing the daemon accepts.
 */
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import '../i18n';
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

vi.mock('../contexts/AppContext', () => ({
  useAppContext: () => ({ sessionId: 'test-session', setSessionId: vi.fn() }),
}));

vi.mock('../api/client', () => ({
  fetchDevices: () =>
    Promise.resolve([
      { name: 'api-router', type: 'router', ips: ['10.10.0.1'], protocols: ['snmp'] },
      { name: 'core-switch', type: 'switch', ips: [], protocols: [] },
    ]),
  fetchConfig: () =>
    Promise.resolve({
      content: CONFIG,
      path: '/tmp/config.yaml',
      modifiedAt: '2026-01-01T00:00:00Z',
      sizeBytes: CONFIG.length,
    }),
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
    <MemoryRouter>
      <DevicesPage />
    </MemoryRouter>,
  );
}

const editor = () => screen.getByLabelText('yaml-editor-stub') as HTMLTextAreaElement;

describe('DevicesPage — device detail', () => {
  beforeEach(() => {
    updateConfig.mockReset();
    updateConfig.mockResolvedValue({
      content: CONFIG,
      path: '/tmp/config.yaml',
      modifiedAt: '2026-01-01T00:00:00Z',
      sizeBytes: CONFIG.length,
    });
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
});
