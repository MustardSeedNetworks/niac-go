/**
 * RunHistoryCard.test.tsx — locks the #recent-runs anchor that Dashboard's
 * "View all history" link targets. The run history moved from the fault
 * injection page to /runtime when the TUI's history viewer was removed, so the
 * anchor has to keep working from its new home.
 */
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { HistoryRecord } from '../../api/types';
import '../../i18n';
import { RunHistoryCard } from './RunHistoryCard';

const fetchHistory = vi.fn<(before?: number) => Promise<HistoryRecord[]>>();

vi.mock('../../api/client', () => ({
  fetchHistory: () => fetchHistory(),
  fetchHistoryPage: (before?: number) => fetchHistory(before),
}));

describe('RunHistoryCard', () => {
  beforeEach(() => {
    fetchHistory.mockReset().mockResolvedValue([]);
  });

  it('renders a #recent-runs anchor on the Recent Runs card', async () => {
    render(
      <MemoryRouter initialEntries={['/runtime']}>
        <RunHistoryCard />
      </MemoryRouter>,
    );
    const heading = await screen.findByText('Run history');
    expect(heading.closest('[id="recent-runs"]')).not.toBeNull();
  });

  it('scrolls the #recent-runs card into view when navigated to with that hash', async () => {
    const scrollIntoView = vi.fn();
    HTMLElement.prototype.scrollIntoView = scrollIntoView;

    render(
      <MemoryRouter initialEntries={['/runtime#recent-runs']}>
        <RunHistoryCard />
      </MemoryRouter>,
    );

    await waitFor(() => expect(scrollIntoView).toHaveBeenCalledWith({ behavior: 'smooth' }));
  });

  it('lists the runs the daemon recorded', async () => {
    fetchHistory.mockResolvedValue([
      {
        id: 1,
        configName: 'clinic.yaml',
        startedAt: new Date('2026-09-04T12:00:00Z').toISOString(),
        duration: '30s',
        interface: 'lo0',
        deviceCount: 12,
        packetsReceived: 4096,
        packetsSent: 2048,
        errors: 0,
      },
    ]);

    render(
      <MemoryRouter initialEntries={['/runtime']}>
        <RunHistoryCard />
      </MemoryRouter>,
    );

    expect(await screen.findByText('clinic.yaml')).toBeInTheDocument();
  });

  it('makes all forty runs reachable and returns to newly recorded runs', async () => {
    const records = Array.from(
      { length: 40 },
      (_, index): HistoryRecord => ({
        id: 40 - index,
        configName: `run-${40 - index}`,
        startedAt: '2026-09-04T12:00:00Z',
        duration: '30s',
        interface: 'lo0',
        deviceCount: 1,
        packetsReceived: 1,
        packetsSent: 1,
        errors: 0,
      }),
    );
    fetchHistory.mockImplementation(async (before) =>
      records.filter((item) => !before || item.id < before).slice(0, 20),
    );
    render(
      <MemoryRouter>
        <RunHistoryCard />
      </MemoryRouter>,
    );
    expect(await screen.findByText('run-21')).toBeInTheDocument();
    expect(screen.getAllByTestId('history-run')).toHaveLength(20);
    const newest = records[0];
    if (!newest) throw new Error('Missing history fixture');
    records.unshift({ ...newest, id: 41, configName: 'new-run' });
    fireEvent.click(screen.getByTestId('history-older'));
    expect(await screen.findByText('run-1')).toBeInTheDocument();
    expect(screen.getAllByTestId('history-run')).toHaveLength(20);
    expect(fetchHistory).toHaveBeenLastCalledWith(21);
    fireEvent.click(screen.getByTestId('history-newer'));
    expect(await screen.findByText('new-run')).toBeInTheDocument();
    expect(screen.getByTestId('history-newer')).toBeDisabled();
  });
});
