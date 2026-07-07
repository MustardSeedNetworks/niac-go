/**
 * LibraryFilesPage.test.tsx — the walks/pcaps library browser.
 *
 * Locks the revert-to-original contract: the "edited" Tag + Revert
 * button appear only for walk entries whose `edited` flag is true,
 * confirming the modal calls revertWalk with the entry's name and
 * refetches the list, and a failed revert surfaces an error without
 * losing the user's place.
 */
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { LibraryFileEntry } from '../api/client';
import { LibraryPcapsPage, LibraryWalksPage } from './LibraryFilesPage';

const fetchLibraryWalks = vi.fn();
const fetchLibraryPcaps = vi.fn();
const revertWalk = vi.fn();

vi.mock('../api/client', () => ({
  fetchLibraryWalks: () => fetchLibraryWalks(),
  fetchLibraryPcaps: () => fetchLibraryPcaps(),
  revertWalk: (name: string) => revertWalk(name),
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

  it('surfaces an error and keeps the modal open when revert fails', async () => {
    fetchLibraryWalks.mockResolvedValue([editedWalk]);
    revertWalk.mockRejectedValue(new Error('daemon unavailable'));

    render(<LibraryWalksPage />);

    fireEvent.click(await screen.findByTestId(`revert-walk-${editedWalk.name}`));
    const dialog = await screen.findByRole('dialog');
    fireEvent.click(within(dialog).getByRole('button', { name: 'Revert' }));

    expect(await screen.findByRole('alert')).toHaveTextContent('daemon unavailable');
    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(fetchLibraryWalks).toHaveBeenCalledTimes(1);
  });
});

describe('LibraryFilesPage (pcaps)', () => {
  beforeEach(() => {
    fetchLibraryWalks.mockReset();
    fetchLibraryPcaps.mockReset();
    revertWalk.mockReset();
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
