/**
 * LibraryFilesPage.test.tsx — the walks/pcaps library browser.
 *
 * Locks the revert-to-original contract: the "edited" Tag + Revert
 * button appear only for walk entries whose `edited` flag is true,
 * confirming the modal calls revertWalk with the entry's name and
 * refetches the list, and a failed revert surfaces an error toast
 * (see refactor/global-error-toasts) without losing the user's place —
 * no more bespoke inline revert-error banner in the modal.
 *
 * Also locks the sanitize contract (Phase 6 Slice B): a per-row Sanitize
 * action confirms then calls sanitizeWalk, and a selection-driven batch
 * action confirms then calls sanitizeWalksBatch — both refetch on success
 * and surface a toast (success or, on partial batch failure, warning).
 */
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { LibraryFileEntry } from '../api/client';
import '../i18n';
import { useUIStore } from '../stores/ui-store';
import { ToastContainer } from '../ui/ToastContainer';
import { LibraryPcapsPage, LibraryWalksPage } from './LibraryFilesPage';

const fetchLibraryWalks = vi.fn();
const fetchLibraryPcaps = vi.fn();
const revertWalk = vi.fn();
const sanitizeWalk = vi.fn();
const sanitizeWalksBatch = vi.fn();

vi.mock('../api/client', () => ({
  fetchLibraryWalks: () => fetchLibraryWalks(),
  fetchLibraryPcaps: () => fetchLibraryPcaps(),
  revertWalk: (name: string) => revertWalk(name),
  sanitizeWalk: (name: string) => sanitizeWalk(name),
  sanitizeWalksBatch: (names: string[]) => sanitizeWalksBatch(names),
}));

const editedWalk: LibraryFileEntry = {
  name: 'cisco/cisco-c3900-01.walk',
  sizeBytes: 2048,
  modifiedAt: '2026-07-01T00:00:00Z',
  source: 'user',
  edited: true,
};

const cleanWalk: LibraryFileEntry = {
  name: 'router.walk',
  sizeBytes: 1024,
  modifiedAt: '2026-07-01T00:00:00Z',
  source: 'starter',
  edited: false,
};

describe('LibraryFilesPage (walks)', () => {
  beforeEach(() => {
    fetchLibraryWalks.mockReset();
    fetchLibraryPcaps.mockReset();
    revertWalk.mockReset();
    sanitizeWalk.mockReset();
    sanitizeWalksBatch.mockReset();
    useUIStore.getState().clearNotifications();
  });

  it('shows a loading state instead of the empty state while the initial fetch is pending', async () => {
    let resolveFetch: (value: LibraryFileEntry[]) => void = () => {};
    fetchLibraryWalks.mockReturnValue(
      new Promise<LibraryFileEntry[]>((resolve) => {
        resolveFetch = resolve;
      }),
    );

    render(<LibraryWalksPage />);

    expect(screen.getByText('Loading walks…')).toBeInTheDocument();
    expect(screen.queryByText('No walks installed yet.')).not.toBeInTheDocument();

    resolveFetch([]);

    await waitFor(() => expect(screen.getByText('No walks installed yet.')).toBeInTheDocument());
    expect(screen.queryByText('Loading walks…')).not.toBeInTheDocument();
  });

  it('shows a Revert button and edited badge only for edited walks', async () => {
    fetchLibraryWalks.mockResolvedValue([editedWalk, cleanWalk]);

    render(<LibraryWalksPage />);

    await screen.findByTestId(`revert-walk-${editedWalk.name}`);
    expect(screen.queryByTestId(`revert-walk-${cleanWalk.name}`)).not.toBeInTheDocument();
    expect(screen.getByText('edited')).toBeInTheDocument();
  });

  it('calls revertWalk with the entry name on confirm, then refetches the list', async () => {
    fetchLibraryWalks.mockResolvedValueOnce([editedWalk]).mockResolvedValueOnce([cleanWalk]);
    revertWalk.mockResolvedValue({ ...editedWalk, edited: false });

    render(<LibraryWalksPage />);

    fireEvent.click(await screen.findByTestId(`revert-walk-${editedWalk.name}`));
    const dialog = await screen.findByRole('dialog');
    fireEvent.click(within(dialog).getByRole('button', { name: 'Revert' }));

    await waitFor(() => expect(revertWalk).toHaveBeenCalledWith(editedWalk.name));
    await waitFor(() => expect(fetchLibraryWalks).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
  });

  it('does not call revertWalk when the confirmation is cancelled', async () => {
    fetchLibraryWalks.mockResolvedValue([editedWalk]);

    render(<LibraryWalksPage />);

    fireEvent.click(await screen.findByTestId(`revert-walk-${editedWalk.name}`));
    const dialog = await screen.findByRole('dialog');
    fireEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }));

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(revertWalk).not.toHaveBeenCalled();
  });

  it('surfaces a failed revert as a toast and keeps the modal open', async () => {
    fetchLibraryWalks.mockResolvedValue([editedWalk]);
    revertWalk.mockRejectedValue(new Error('daemon unavailable'));

    render(
      <>
        <LibraryWalksPage />
        <ToastContainer />
      </>,
    );

    fireEvent.click(await screen.findByTestId(`revert-walk-${editedWalk.name}`));
    const dialog = await screen.findByRole('dialog');
    fireEvent.click(within(dialog).getByRole('button', { name: 'Revert' }));

    // Toast, not a bespoke inline banner inside the modal.
    expect(await screen.findByRole('alert')).toHaveTextContent('daemon unavailable');
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(fetchLibraryWalks).toHaveBeenCalledTimes(1);
  });

  it('calls sanitizeWalk with the entry name on confirm, then refetches and toasts success', async () => {
    fetchLibraryWalks.mockResolvedValueOnce([cleanWalk]).mockResolvedValueOnce([editedWalk]);
    sanitizeWalk.mockResolvedValue({ ...cleanWalk, edited: true });

    render(
      <>
        <LibraryWalksPage />
        <ToastContainer />
      </>,
    );

    fireEvent.click(await screen.findByTestId(`sanitize-walk-${cleanWalk.name}`));
    const dialog = await screen.findByRole('dialog');
    fireEvent.click(within(dialog).getByRole('button', { name: 'Sanitize' }));

    await waitFor(() => expect(sanitizeWalk).toHaveBeenCalledWith(cleanWalk.name));
    await waitFor(() => expect(fetchLibraryWalks).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(await screen.findByRole('alert')).toHaveTextContent('Walk sanitized');
  });

  it('does not call sanitizeWalk when the sanitize confirmation is cancelled', async () => {
    fetchLibraryWalks.mockResolvedValue([cleanWalk]);

    render(<LibraryWalksPage />);

    fireEvent.click(await screen.findByTestId(`sanitize-walk-${cleanWalk.name}`));
    const dialog = await screen.findByRole('dialog');
    fireEvent.click(within(dialog).getByRole('button', { name: 'Cancel' }));

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(sanitizeWalk).not.toHaveBeenCalled();
  });

  it('surfaces a failed sanitize as a toast and keeps the modal open', async () => {
    fetchLibraryWalks.mockResolvedValue([cleanWalk]);
    sanitizeWalk.mockRejectedValue(new Error('daemon unavailable'));

    render(
      <>
        <LibraryWalksPage />
        <ToastContainer />
      </>,
    );

    fireEvent.click(await screen.findByTestId(`sanitize-walk-${cleanWalk.name}`));
    const dialog = await screen.findByRole('dialog');
    fireEvent.click(within(dialog).getByRole('button', { name: 'Sanitize' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('daemon unavailable');
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(fetchLibraryWalks).toHaveBeenCalledTimes(1);
  });

  it('selects walks via row checkboxes and sanitizes the selection as a batch', async () => {
    fetchLibraryWalks.mockResolvedValueOnce([cleanWalk, editedWalk]).mockResolvedValueOnce([]);
    sanitizeWalksBatch.mockResolvedValue({
      results: [
        { name: cleanWalk.name, success: true, ipsTransformed: 2, hostnamesTransformed: 1 },
        { name: editedWalk.name, success: true, ipsTransformed: 0, hostnamesTransformed: 0 },
      ],
      sanitized: 2,
      failed: 0,
    });

    render(
      <>
        <LibraryWalksPage />
        <ToastContainer />
      </>,
    );

    fireEvent.click(await screen.findByLabelText(`Select ${cleanWalk.name}`));
    fireEvent.click(screen.getByLabelText(`Select ${editedWalk.name}`));

    fireEvent.click(await screen.findByTestId('sanitize-selected-walks'));
    const dialog = await screen.findByRole('dialog');
    fireEvent.click(within(dialog).getByRole('button', { name: 'Sanitize selected' }));

    await waitFor(() =>
      expect(sanitizeWalksBatch).toHaveBeenCalledWith([cleanWalk.name, editedWalk.name]),
    );
    await waitFor(() => expect(fetchLibraryWalks).toHaveBeenCalledTimes(2));
    expect(await screen.findByRole('alert')).toHaveTextContent('2 sanitized, 0 failed');
    // Selection clears after a successful batch.
    expect(screen.queryByTestId('sanitize-selected-walks')).not.toBeInTheDocument();
  });

  it('reports partial batch failures as a warning toast', async () => {
    fetchLibraryWalks.mockResolvedValue([cleanWalk, editedWalk]);
    sanitizeWalksBatch.mockResolvedValue({
      results: [
        { name: cleanWalk.name, success: true, ipsTransformed: 1, hostnamesTransformed: 0 },
        { name: editedWalk.name, success: false, error: 'not found' },
      ],
      sanitized: 1,
      failed: 1,
    });

    render(
      <>
        <LibraryWalksPage />
        <ToastContainer />
      </>,
    );

    fireEvent.click(await screen.findByLabelText(`Select ${cleanWalk.name}`));
    fireEvent.click(screen.getByLabelText(`Select ${editedWalk.name}`));
    fireEvent.click(await screen.findByTestId('sanitize-selected-walks'));
    const dialog = await screen.findByRole('dialog');
    fireEvent.click(within(dialog).getByRole('button', { name: 'Sanitize selected' }));

    await waitFor(() => expect(sanitizeWalksBatch).toHaveBeenCalled());
    expect(await screen.findByRole('alert')).toHaveTextContent('1 sanitized, 1 failed');
  });
});

describe('LibraryFilesPage (pcaps)', () => {
  beforeEach(() => {
    fetchLibraryWalks.mockReset();
    fetchLibraryPcaps.mockReset();
    revertWalk.mockReset();
    sanitizeWalk.mockReset();
    sanitizeWalksBatch.mockReset();
    useUIStore.getState().clearNotifications();
  });

  it('never renders an Actions column or edited badge for pcaps', async () => {
    // Even a hypothetical edited=true pcap entry must not surface the
    // walks-only revert control — the feature is scoped to walks.
    fetchLibraryPcaps.mockResolvedValue([{ ...editedWalk, source: 'user' }]);

    render(<LibraryPcapsPage />);

    await screen.findByText(editedWalk.name);
    expect(screen.queryByText('Actions')).not.toBeInTheDocument();
    expect(screen.queryByText('edited')).not.toBeInTheDocument();
    expect(screen.queryByTestId(`revert-walk-${editedWalk.name}`)).not.toBeInTheDocument();
  });
});
