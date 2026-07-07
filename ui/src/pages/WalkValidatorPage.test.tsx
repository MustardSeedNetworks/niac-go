/**
 * WalkValidatorPage.test.tsx
 *
 * Data-safety regression test: "Auto-fix" rewrites the walk file in place
 * (the .bak safety net was previously mentioned only in a hover tooltip).
 * Clicking it must open a confirmation naming the target file and the
 * .bak behavior, and the file must only be rewritten after the user
 * confirms — never on the click itself.
 */
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import '../i18n';
import { WalkValidatorPage } from './WalkValidatorPage';

const fetchLibraryWalks = vi.fn();
const fixWalk = vi.fn();
const validateWalk = vi.fn();

vi.mock('../api/client', () => ({
  fetchLibraryWalks: () => fetchLibraryWalks(),
  fixWalk: (filename: string) => fixWalk(filename),
  validateWalk: (filename: string) => validateWalk(filename),
}));

const files = [
  { name: 'cisco/c3900.walk', sizeBytes: 2048, modifiedAt: '', source: 'user', edited: false },
];

afterEach(() => {
  vi.clearAllMocks();
});

describe('WalkValidatorPage — Auto-fix confirmation', () => {
  it('does not rewrite the file on the Auto-fix click alone', async () => {
    fetchLibraryWalks.mockResolvedValue(files);
    render(<WalkValidatorPage />);

    await waitFor(() => expect(screen.getByText(/cisco\/c3900\.walk/)).toBeInTheDocument());

    fireEvent.click(screen.getByRole('button', { name: /auto-fix/i }));

    expect(fixWalk).not.toHaveBeenCalled();
    expect(screen.getByRole('heading', { name: /auto-fix walk file/i })).toBeInTheDocument();
    expect(screen.getByText(/backup of the original is created/i)).toBeInTheDocument();
  });

  it('calls fixWalk only after the user confirms', async () => {
    fetchLibraryWalks.mockResolvedValue(files);
    fixWalk.mockResolvedValueOnce({
      message: 'Fixed',
      result: { totalLines: 1, validLines: 1, valid: true, issues: [], fixedCount: 1 },
    });
    render(<WalkValidatorPage />);

    await waitFor(() => expect(screen.getByText(/cisco\/c3900\.walk/)).toBeInTheDocument());

    fireEvent.click(screen.getByRole('button', { name: /auto-fix/i }));
    const confirmButtons = screen.getAllByRole('button', { name: /auto-fix/i });
    // The dialog's confirm button is the last "Auto-fix"-labeled control.
    fireEvent.click(confirmButtons[confirmButtons.length - 1]);

    await waitFor(() => expect(fixWalk).toHaveBeenCalledWith('cisco/c3900.walk'));
  });

  it('cancels without calling fixWalk', async () => {
    fetchLibraryWalks.mockResolvedValue(files);
    render(<WalkValidatorPage />);

    await waitFor(() => expect(screen.getByText(/cisco\/c3900\.walk/)).toBeInTheDocument());

    fireEvent.click(screen.getByRole('button', { name: /auto-fix/i }));
    fireEvent.click(screen.getByRole('button', { name: /cancel/i }));

    expect(fixWalk).not.toHaveBeenCalled();
    expect(screen.queryByRole('heading', { name: /auto-fix walk file/i })).not.toBeInTheDocument();
  });
});
