/**
 * ReplayControlPanel.test.tsx — Phase 5a live replay progress. Locks that
 * the status card renders a packets/bytes-sent progress bar while a replay
 * is running, and falls back to "unknown total" copy (never a fake 0%
 * bar) when the backend hasn't reported packetsTotal/percentComplete yet.
 */
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { ReplayState } from '../api/api-response-types';
import { startReplay } from '../api/client';
import '../i18n';
import { ReplayControlPanel } from './ReplayControlPanel';

const fetchLibraryPcaps = vi.fn();
const fetchReplayStatus = vi.fn<() => Promise<ReplayState>>();

vi.mock('../api/client', () => ({
  fetchReplayStatus: () => fetchReplayStatus(),
  startReplay: vi.fn(),
  stopReplay: vi.fn(),
}));
vi.mock('../api/library-client', () => ({
  fetchLibraryPcaps: () => fetchLibraryPcaps(),
}));

const runningWithProgress: ReplayState = {
  running: true,
  file: 'sample.pcap',
  loopMs: 0,
  scale: 1,
  startedAt: new Date().toISOString(),
  packetsSent: 25,
  bytesSent: 2048,
  packetsTotal: 100,
  bytesTotal: 8192,
  percentComplete: 25,
  passes: 1,
  packetsFiltered: 0,
};

describe('ReplayControlPanel — live progress', () => {
  beforeEach(() => {
    fetchLibraryPcaps.mockReset().mockResolvedValue([]);
    fetchReplayStatus.mockReset();
  });

  it('renders a progress bar with sent/total counts and percent while running', async () => {
    fetchReplayStatus.mockResolvedValue(runningWithProgress);

    render(<ReplayControlPanel />);

    const progress = await screen.findByTestId('replay-progress');
    expect(progress).toBeInTheDocument();

    const bar = screen.getByRole('progressbar');
    expect(bar).toHaveAttribute('aria-valuenow', '25');
    expect(bar).toHaveAttribute('aria-valuemax', '100');

    expect(await screen.findByText(/25 \/ 100 packets \(25%\)/)).toBeInTheDocument();
  });

  it('shows unknown-total copy instead of a fake percentage when packetsTotal is 0', async () => {
    fetchReplayStatus.mockResolvedValue({
      ...runningWithProgress,
      packetsTotal: 0,
      percentComplete: undefined,
    });

    render(<ReplayControlPanel />);

    const bar = await screen.findByRole('progressbar');
    expect(bar).not.toHaveAttribute('aria-valuenow');
    expect(screen.getByText(/total unknown/i)).toBeInTheDocument();
  });

  it('does not render the progress bar when no replay is running', async () => {
    fetchReplayStatus.mockResolvedValue({
      running: false,
      file: '',
      loopMs: 0,
      scale: 1,
      packetsSent: 0,
      bytesSent: 0,
      packetsTotal: 0,
      bytesTotal: 0,
      passes: 0,
      packetsFiltered: 0,
    });

    render(<ReplayControlPanel />);

    await screen.findByText('PCAP File');
    expect(screen.queryByTestId('replay-progress')).not.toBeInTheDocument();
  });

  it('sends the rate/loop/filter fields on start', async () => {
    fetchReplayStatus.mockResolvedValue({
      running: false,
      file: '',
      loopMs: 0,
      scale: 1,
      packetsSent: 0,
      bytesSent: 0,
      packetsTotal: 0,
      bytesTotal: 0,
      passes: 0,
      packetsFiltered: 0,
    });
    fetchLibraryPcaps.mockResolvedValue([
      { name: 'a.pcap', path: 'a.pcap', sizeBytes: 1024, modifiedAt: '' },
    ]);
    vi.mocked(startReplay)
      .mockReset()
      .mockResolvedValue({} as ReplayState);

    const user = userEvent.setup();
    render(<ReplayControlPanel />);

    await screen.findByText('PCAP File');
    await user.selectOptions(screen.getByLabelText('PCAP File'), 'a.pcap');
    await user.selectOptions(screen.getByLabelText('Rate Mode'), 'pps');
    await user.clear(screen.getByLabelText('Rate (pps)'));
    await user.type(screen.getByLabelText('Rate (pps)'), '500');
    await user.clear(screen.getByLabelText('Loop Count'));
    await user.type(screen.getByLabelText('Loop Count'), '3');
    await user.type(screen.getByLabelText('BPF Filter'), 'udp port 53');
    await user.click(screen.getByRole('button', { name: /start replay/i }));

    expect(vi.mocked(startReplay)).toHaveBeenCalledWith(
      expect.objectContaining({
        file: 'a.pcap',
        rateMode: 'pps',
        pps: 500,
        loopCount: 3,
        bpfFilter: 'udp port 53',
      }),
    );
  });
});
