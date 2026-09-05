/**
 * RunHistoryCard.test.tsx — locks the #recent-runs anchor that Dashboard's
 * "View all history" link targets. The run history moved from the fault
 * injection page to /runtime when the TUI's history viewer was removed, so the
 * anchor has to keep working from its new home.
 */
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { HistoryRecord } from '../../api/types';
import '../../i18n';
import { RunHistoryCard } from './RunHistoryCard';

const fetchHistory = vi.fn<() => Promise<HistoryRecord[]>>();

vi.mock('../../api/client', () => ({
  fetchHistory: () => fetchHistory(),
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
    const heading = await screen.findByText('Recent runs');
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
        duration: 30,
        deviceCount: 12,
        packetsReceived: 4096,
        packetsSent: 2048,
      } as HistoryRecord,
    ]);

    render(
      <MemoryRouter initialEntries={['/runtime']}>
        <RunHistoryCard />
      </MemoryRouter>,
    );

    expect(await screen.findByText('clinic.yaml')).toBeInTheDocument();
  });
});
