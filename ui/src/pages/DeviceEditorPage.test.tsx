/**
 * DeviceEditorPage.test.tsx
 *
 * Data-safety regression test: the device editor previously had no
 * unsaved-changes guard — clicking "Back" while the form was dirty
 * navigated away immediately and silently discarded the in-progress
 * edit. This pins the integration: a dirty new-device form shows a
 * confirmation on Back, "Stay" keeps the user on the page, and "Leave"
 * proceeds with the navigation.
 */
import { act, fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { createMemoryRouter, RouterProvider } from 'react-router';
import { afterEach, describe, expect, it, vi } from 'vitest';
import '../i18n';
import { createDevice } from '../api/client';
import { DeviceEditorPage } from './DeviceEditorPage';

vi.mock('../api/client', () => ({
  createDevice: vi.fn(),
  updateDevice: vi.fn(),
  deleteDevice: vi.fn(),
  fetchConfigDevice: (name: string) =>
    Promise.resolve({ rawYaml: `name: ${name}\ntype: switch\nmac: 00:1A:2B:3C:4D:5E\n` }),
  fetchDeviceEditorSchema: () => Promise.resolve({ visibleSections: ['basic'] }),
}));
vi.mock('../api/library-client', () => ({
  fetchLibraryWalks: () => Promise.resolve([]),
}));

afterEach(() => {
  vi.clearAllMocks();
});

function renderPage(path = '/device-config/new') {
  const router = createMemoryRouter(
    [
      { path: '/device-config/:hostname', element: <DeviceEditorPage /> },
      { path: '/device-config', element: <p>Device list</p> },
    ],
    { initialEntries: [path] },
  );
  render(<RouterProvider router={router} />);
  return router;
}

/**
 * The new-device form loads through a chain of resolved promises
 * (fetchConfigDevice, fetchDeviceEditorSchema) that each trigger a
 * `reset()` of the underlying react-hook-form state. Waiting only for
 * the hostname field to appear can race a change fired immediately
 * after — the pending `reset()` clobbers it. A macrotask tick drains
 * every already-queued microtask (including the reset chain) before the
 * test interacts with the form.
 */
const flushPendingEffects = () =>
  new Promise<void>((resolve) => {
    setTimeout(() => act(resolve), 0);
  });

describe('DeviceEditorPage — unsaved-changes navigation guard', () => {
  it('leaves after deleting a dirty device without asking to save the deleted device', async () => {
    const router = renderPage('/device-config/edge-01');
    const name = await screen.findByRole('textbox', { name: /hostname/i });
    await waitFor(() => expect(name).toHaveValue('edge-01'));
    fireEvent.change(name, { target: { value: 'changed-name' } });
    fireEvent.click(screen.getByRole('button', { name: /^delete$/i }));
    fireEvent.click(within(screen.getByRole('dialog')).getByRole('button', { name: /^delete$/i }));
    await waitFor(() => expect(router.state.location.pathname).toBe('/device-config'));
    expect(screen.queryByText(/leave without saving/i)).not.toBeInTheDocument();
  });
  it('navigates to a successfully created device without an unsaved prompt', async () => {
    let resolveSave: (value: Awaited<ReturnType<typeof createDevice>>) => void = () => {};
    vi.mocked(createDevice).mockReturnValueOnce(
      new Promise((resolve) => {
        resolveSave = resolve;
      }),
    );
    const router = renderPage();
    await waitFor(() => expect(screen.getByLabelText(/hostname/i)).toBeInTheDocument());
    await flushPendingEffects();
    fireEvent.change(screen.getByLabelText(/hostname/i), { target: { value: 'edge-01' } });
    fireEvent.change(screen.getByRole('textbox', { name: /mac address/i }), {
      target: { value: '00:1A:2B:3C:4D:5E' },
    });
    fireEvent.click(screen.getByTestId('device-editor-save'));
    expect(screen.getByRole('textbox', { name: /hostname/i })).toBeDisabled();
    resolveSave({ success: true, message: 'Created' });
    await waitFor(() => expect(router.state.location.pathname).toBe('/device-config/edge-01'));
    expect(screen.queryByText(/leave without saving/i)).not.toBeInTheDocument();
  });
  it('shows a confirmation on Back when the form is dirty, and Stay keeps the user on the page', async () => {
    const router = renderPage();
    await waitFor(() => expect(screen.getByLabelText(/hostname/i)).toBeInTheDocument());
    await flushPendingEffects();

    fireEvent.change(screen.getByLabelText(/hostname/i), {
      target: { value: 'edge-01' },
    });

    fireEvent.click(screen.getByTitle('Back to device list'));

    expect(screen.getByText(/leave without saving/i)).toBeInTheDocument();
    expect(router.state.location.pathname).toBe('/device-config/new');

    fireEvent.click(screen.getByRole('button', { name: /^stay$/i }));
    expect(router.state.location.pathname).toBe('/device-config/new');
    expect(screen.queryByText(/leave without saving/i)).not.toBeInTheDocument();
  });

  it('navigates away after the user confirms Leave', async () => {
    const router = renderPage();
    await waitFor(() => expect(screen.getByLabelText(/hostname/i)).toBeInTheDocument());
    await flushPendingEffects();

    fireEvent.change(screen.getByLabelText(/hostname/i), {
      target: { value: 'edge-01' },
    });

    fireEvent.click(screen.getByTitle('Back to device list'));
    fireEvent.click(screen.getByRole('button', { name: /^leave$/i }));

    await waitFor(() => expect(router.state.location.pathname).toBe('/device-config'));
  });

  it('navigates immediately on Back when the form has no unsaved changes', async () => {
    const router = renderPage();
    await waitFor(() => expect(screen.getByLabelText(/hostname/i)).toBeInTheDocument());
    await flushPendingEffects();

    fireEvent.click(screen.getByTitle('Back to device list'));

    await waitFor(() => expect(router.state.location.pathname).toBe('/device-config'));
    expect(screen.queryByText(/leave without saving/i)).not.toBeInTheDocument();
  });
});
