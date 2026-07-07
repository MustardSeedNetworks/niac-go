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
import { useUIStore } from '../stores/ui-store';
import { ToastContainer } from '../ui/ToastContainer';
import { WalkValidatorPage } from './WalkValidatorPage';

const fetchLibraryWalks = vi.fn();
const fixWalk = vi.fn();
const validateWalk = vi.fn();
const validateAllWalks = vi.fn();

vi.mock('../api/client', () => ({
  fetchLibraryWalks: () => fetchLibraryWalks(),
  fixWalk: (filename: string) => fixWalk(filename),
  validateWalk: (filename: string) => validateWalk(filename),
  validateAllWalks: () => validateAllWalks(),
}));

const files = [
  { name: 'cisco/c3900.walk', sizeBytes: 2048, modifiedAt: '', source: 'user', edited: false },
];

afterEach(() => {
  vi.clearAllMocks();
  useUIStore.getState().clearNotifications();
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

describe('WalkValidatorPage — Validate all', () => {
  it('renders a per-file results table on success', async () => {
    fetchLibraryWalks.mockResolvedValue(files);
    validateAllWalks.mockResolvedValueOnce({
      success: false,
      message: '1 of 2 walk files have issues',
      totalFiles: 2,
      invalidFiles: 1,
      totalIssues: 3,
      results: {
        'cisco/c3900.walk': {
          filename: 'cisco/c3900.walk',
          valid: true,
          totalLines: 10,
          validLines: 10,
          issues: [],
        },
        'juniper/mx.walk': {
          filename: 'juniper/mx.walk',
          valid: false,
          totalLines: 5,
          validLines: 2,
          issues: [
            {
              line: 0,
              severity: 'error',
              message: 'walk file not found',
              original: '',
              autoFix: false,
            },
          ],
        },
      },
    });

    render(<WalkValidatorPage />);

    await waitFor(() => expect(screen.getByText(/cisco\/c3900\.walk/)).toBeInTheDocument());

    fireEvent.click(screen.getByRole('button', { name: /validate all/i }));

    await waitFor(() => expect(validateAllWalks).toHaveBeenCalledTimes(1));

    expect(await screen.findByText('1 of 2 walk files have issues')).toBeInTheDocument();
    expect(screen.getByText('juniper/mx.walk')).toBeInTheDocument();
    expect(screen.getAllByText(/valid/i).length).toBeGreaterThan(0);
    expect(screen.getAllByText(/invalid/i).length).toBeGreaterThan(0);
  });

  it('surfaces a failed batch validation as a toast', async () => {
    fetchLibraryWalks.mockResolvedValue(files);
    validateAllWalks.mockRejectedValueOnce(new Error('daemon unavailable'));

    render(
      <>
        <WalkValidatorPage />
        <ToastContainer />
      </>,
    );

    await waitFor(() => expect(screen.getByText(/cisco\/c3900\.walk/)).toBeInTheDocument());

    fireEvent.click(screen.getByRole('button', { name: /validate all/i }));

    expect(await screen.findByRole('alert')).toHaveTextContent('daemon unavailable');
  });
});

describe('WalkValidatorPage — OID column and filter', () => {
  const validateResult = {
    message: 'Walk file has 2 issues',
    result: {
      totalLines: 2,
      validLines: 0,
      valid: false,
      issues: [
        {
          line: 1,
          severity: 'warning',
          message: "Type 'strng' appears to be misspelled",
          original: '.1.3.6.1.2.1.1.5.0 = strng: "router1"',
          autoFix: true,
          oid: '.1.3.6.1.2.1.1.5.0',
        },
        {
          line: 2,
          severity: 'error',
          message: "Missing ':' separator between type and value",
          original: '.1.3.6.1.4.1.9.1.1 = STRINGnope',
          autoFix: false,
          oid: '.1.3.6.1.4.1.9.1.1',
        },
      ],
    },
  };

  it('renders the OID for each issue in the results table', async () => {
    fetchLibraryWalks.mockResolvedValue(files);
    validateWalk.mockResolvedValueOnce(validateResult);
    render(<WalkValidatorPage />);

    await waitFor(() => expect(screen.getByText(/cisco\/c3900\.walk/)).toBeInTheDocument());

    fireEvent.click(screen.getByRole('button', { name: /^validate$/i }));

    expect(await screen.findByText('.1.3.6.1.2.1.1.5.0')).toBeInTheDocument();
    expect(screen.getByText('.1.3.6.1.4.1.9.1.1')).toBeInTheDocument();
  });

  it('narrows the displayed issues by OID substring', async () => {
    fetchLibraryWalks.mockResolvedValue(files);
    validateWalk.mockResolvedValueOnce(validateResult);
    render(<WalkValidatorPage />);

    await waitFor(() => expect(screen.getByText(/cisco\/c3900\.walk/)).toBeInTheDocument());
    fireEvent.click(screen.getByRole('button', { name: /^validate$/i }));

    await waitFor(() => expect(screen.getByText('.1.3.6.1.2.1.1.5.0')).toBeInTheDocument());
    expect(screen.getByText('.1.3.6.1.4.1.9.1.1')).toBeInTheDocument();

    const filterInput = screen.getByPlaceholderText(/1\.3\.6\.1\.2\.1\.1/);
    fireEvent.change(filterInput, { target: { value: '9.1.1' } });

    await waitFor(() => expect(screen.queryByText('.1.3.6.1.2.1.1.5.0')).not.toBeInTheDocument());
    expect(screen.getByText('.1.3.6.1.4.1.9.1.1')).toBeInTheDocument();
  });
});
