/**
 * BpfFilterBar.test.tsx — Phase 5b: the "Invalid BPF filter expression"
 * alert must show the real libpcap compile error the API attaches as a
 * structured detail, not just the generic top-level message.
 */
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import '../i18n';
import { ApiError } from '../api/errors';
import { BpfFilterBar } from './BpfFilterBar';

const getCaptureFilter = vi.fn();
const setCaptureFilter = vi.fn();
const clearCaptureFilter = vi.fn();

vi.mock('../api/capture', () => ({
  getCaptureFilter: () => getCaptureFilter(),
  setCaptureFilter: (filter: string) => setCaptureFilter(filter),
  clearCaptureFilter: () => clearCaptureFilter(),
}));

describe('BpfFilterBar', () => {
  beforeEach(() => {
    getCaptureFilter.mockReset().mockResolvedValue({ active: false, filter: '' });
    setCaptureFilter.mockReset();
    clearCaptureFilter.mockReset();
  });

  it('shows the real libpcap compile error from the structured detail', async () => {
    setCaptureFilter.mockRejectedValue(
      new ApiError('Invalid BPF filter expression', 400, 'invalid_filter', [
        {
          field: 'filter',
          issue: "failed to set BPF filter: can't parse filter expression: syntax error",
          value: 'tcp port',
        },
      ]),
    );

    render(<BpfFilterBar />);

    const input = await screen.findByPlaceholderText(/./);
    fireEvent.change(input, { target: { value: 'tcp port' } });
    fireEvent.click(screen.getByText('Apply'));

    await waitFor(() =>
      expect(screen.getByText(/can't parse filter expression: syntax error/)).toBeInTheDocument(),
    );
  });

  it('falls back to the generic message when the server sends no detail', async () => {
    setCaptureFilter.mockRejectedValue(new ApiError('Invalid BPF filter expression', 400));

    render(<BpfFilterBar />);

    const input = await screen.findByPlaceholderText(/./);
    fireEvent.change(input, { target: { value: 'garbage' } });
    fireEvent.click(screen.getByText('Apply'));

    await waitFor(() =>
      expect(screen.getByText('Invalid BPF filter expression')).toBeInTheDocument(),
    );
  });
});
