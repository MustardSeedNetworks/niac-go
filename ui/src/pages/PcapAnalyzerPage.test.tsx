// This isolated component fixture represents an authenticated operator.
vi.mock('../contexts/ScopeContext', () => ({
  useActionPermission: () => ({ disabled: false }),
}));

/**
 * PcapAnalyzerPage.test.tsx — cancelling an upload in flight (U7).
 *
 * The upload is a single JSON POST whose body carries the whole capture
 * base64-encoded; the daemon holds the analysis in `pcapCache` and never
 * writes a file (internal/api/pcap.go). So "cancel leaves no partial file"
 * is really "cancel leaves no analysis, no success banner and no error
 * toast" — an aborted request is the user's own decision, not a failure.
 */
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import '../i18n';
import { PcapAnalyzerPage } from './PcapAnalyzerPage';

const uploadPcapWithProgress = vi.fn();
const fetchPcapAnalysis = vi.fn();

vi.mock('../api/client', () => ({
  uploadPcapWithProgress: (...args: unknown[]) => uploadPcapWithProgress(...args),
  fetchPcapAnalysis: (...args: unknown[]) => fetchPcapAnalysis(...args),
}));

const showError = vi.fn();
vi.mock('../hooks/useErrorToast', () => ({ useErrorToast: () => showError }));

vi.mock('../utils/file', () => ({
  fileToBase64: () => Promise.resolve('AAAA'),
}));

/** The signal the page handed to the upload call, captured per test. */
let uploadSignal: AbortSignal | undefined;

/**
 * Resolve the upload only when the caller aborts, mirroring what an XHR
 * does: it stays in flight indefinitely and rejects with an AbortError the
 * moment the signal fires. Also drives the progress callback once so the
 * determinate bar — and with it the cancel button — is on screen.
 */
function uploadThatOnlyEndsOnAbort() {
  return (
    _payload: unknown,
    onProgress: (percent: number) => void,
    signal: AbortSignal | undefined,
  ) => {
    uploadSignal = signal;
    onProgress(42);
    return new Promise((_resolve, reject) => {
      signal?.addEventListener('abort', () =>
        reject(new DOMException('Request aborted', 'AbortError')),
      );
    });
  };
}

async function selectFileAndAnalyze(user: ReturnType<typeof userEvent.setup>) {
  const input = document.querySelector('input[type="file"]') as HTMLInputElement;
  await user.upload(
    input,
    new File(['xx'], 'sample.pcap', { type: 'application/vnd.tcpdump.pcap' }),
  );
  await user.click(screen.getByRole('button', { name: 'Analyze PCAP' }));
  await screen.findByTestId('pcap-upload-progress');
}

describe('PcapAnalyzerPage — cancelling an upload', () => {
  beforeEach(() => {
    uploadSignal = undefined;
    showError.mockReset();
    fetchPcapAnalysis.mockReset();
    uploadPcapWithProgress.mockReset().mockImplementation(uploadThatOnlyEndsOnAbort());
  });

  it('aborts the request and returns to ready without a toast or a result', async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter>
        <PcapAnalyzerPage />
      </MemoryRouter>,
    );

    await selectFileAndAnalyze(user);
    expect(uploadSignal?.aborted).toBe(false);

    await user.click(screen.getByRole('button', { name: 'Cancel upload' }));

    await waitFor(() => expect(uploadSignal?.aborted).toBe(true));
    // The progress bar goes away and the page is analysable again.
    await waitFor(() => expect(screen.queryByTestId('pcap-upload-progress')).toBeNull());
    expect(screen.getByRole('button', { name: 'Analyze PCAP' })).toBeEnabled();

    // A cancel is not a failure: no toast, no analysis fetched, no success.
    expect(showError).not.toHaveBeenCalled();
    expect(fetchPcapAnalysis).not.toHaveBeenCalled();
    expect(screen.queryByText(/Analyzed/)).toBeNull();
  });

  it('aborts an in-flight upload when the page unmounts', async () => {
    const user = userEvent.setup();
    const { unmount } = render(
      <MemoryRouter>
        <PcapAnalyzerPage />
      </MemoryRouter>,
    );

    await selectFileAndAnalyze(user);
    expect(uploadSignal?.aborted).toBe(false);

    unmount();

    expect(uploadSignal?.aborted).toBe(true);
    expect(showError).not.toHaveBeenCalled();
  });
});
