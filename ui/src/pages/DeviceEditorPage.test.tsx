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
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { createElement } from 'react';
import { MemoryRouter } from 'react-router';
import { afterEach, describe, expect, it, vi } from 'vitest';
import '../i18n';
import { DeviceEditorPage } from './DeviceEditorPage';

const mockNavigate = vi.fn();
vi.mock('react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router')>();
  return {
    ...actual,
    useNavigate: () => mockNavigate,
    useParams: () => ({ hostname: 'new' }),
    useLocation: () => ({ pathname: '/device-config/new' }),
  };
});

vi.mock('../api/client', () => ({
  createDevice: vi.fn(),
  updateDevice: vi.fn(),
  deleteDevice: vi.fn(),
  fetchConfigDevice: () => Promise.resolve({ device: null }),
  fetchDeviceEditorSchema: () => Promise.resolve({ visibleSections: ['basic'] }),
}));
vi.mock('../api/library-client', () => ({
  fetchLibraryWalks: () => Promise.resolve([]),
}));

afterEach(() => {
  vi.clearAllMocks();
});

function renderPage() {
  return render(createElement(MemoryRouter, null, createElement(DeviceEditorPage)));
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
const flushPendingEffects = () => act(() => new Promise((resolve) => setTimeout(resolve, 0)));

describe('DeviceEditorPage — unsaved-changes navigation guard', () => {
  it('shows a confirmation on Back when the form is dirty, and Stay keeps the user on the page', async () => {
    renderPage();
    await waitFor(() => expect(screen.getByLabelText(/hostname/i)).toBeInTheDocument());
    await flushPendingEffects();

    fireEvent.change(screen.getByLabelText(/hostname/i), {
      target: { value: 'edge-01' },
    });

    fireEvent.click(screen.getByTitle('Back to device list'));

    expect(screen.getByText(/leave without saving/i)).toBeInTheDocument();
    expect(mockNavigate).not.toHaveBeenCalled();

    fireEvent.click(screen.getByRole('button', { name: /^stay$/i }));
    expect(mockNavigate).not.toHaveBeenCalled();
    expect(screen.queryByText(/leave without saving/i)).not.toBeInTheDocument();
  });

  it('navigates away after the user confirms Leave', async () => {
    renderPage();
    await waitFor(() => expect(screen.getByLabelText(/hostname/i)).toBeInTheDocument());
    await flushPendingEffects();

    fireEvent.change(screen.getByLabelText(/hostname/i), {
      target: { value: 'edge-01' },
    });

    fireEvent.click(screen.getByTitle('Back to device list'));
    fireEvent.click(screen.getByRole('button', { name: /^leave$/i }));

    expect(mockNavigate).toHaveBeenCalledWith('/device-config');
  });

  it('navigates immediately on Back when the form has no unsaved changes', async () => {
    renderPage();
    await waitFor(() => expect(screen.getByLabelText(/hostname/i)).toBeInTheDocument());
    await flushPendingEffects();

    fireEvent.click(screen.getByTitle('Back to device list'));

    expect(mockNavigate).toHaveBeenCalledWith('/device-config');
    expect(screen.queryByText(/leave without saving/i)).not.toBeInTheDocument();
  });
});
