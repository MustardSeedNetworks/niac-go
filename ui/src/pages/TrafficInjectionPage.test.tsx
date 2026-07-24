/**
 * TrafficInjectionPage.test.tsx — locks the #recent-runs anchor added so
 * Dashboard's "View all history" link lands on the actual expanded run
 * history instead of the top of a page whose primary content is fault
 * injection (PR "Dashboard honesty + shell sim-status").
 */
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { HistoryRecord } from '../api/types';
import '../i18n';
import { TrafficInjectionPage } from './TrafficInjectionPage';

const fetchHistory = vi.fn<() => Promise<HistoryRecord[]>>();
const fetchDevices = vi.fn();
const fetchErrorTypes = vi.fn();
const fetchLibraryPcaps = vi.fn();
const fetchReplayStatus = vi.fn();
const hasFeature = vi.fn<(feature: string) => boolean>();

vi.mock('../api/client', () => ({
  fetchHistory: () => fetchHistory(),
  fetchDevices: () => fetchDevices(),
  fetchErrorTypes: () => fetchErrorTypes(),
  fetchReplayStatus: () => fetchReplayStatus(),
  injectError: vi.fn(),
  clearError: vi.fn(),
  clearAllErrors: vi.fn(),
  startReplay: vi.fn(),
  stopReplay: vi.fn(),
}));
vi.mock('../api/library-client', () => ({
  fetchLibraryPcaps: () => fetchLibraryPcaps(),
}));
vi.mock('../contexts/LicenseContext', () => ({
  useLicense: () => ({ hasFeature, loading: false }),
}));

describe('TrafficInjectionPage', () => {
  beforeEach(() => {
    fetchHistory.mockReset().mockResolvedValue([]);
    fetchDevices.mockReset().mockResolvedValue([]);
    fetchErrorTypes.mockReset().mockResolvedValue({ availableTypes: [], info: '' });
    fetchLibraryPcaps.mockReset().mockResolvedValue([]);
    fetchReplayStatus.mockReset().mockResolvedValue({ running: false });
    hasFeature.mockReset().mockReturnValue(true);
  });

  it('renders a #recent-runs anchor on the Recent Runs card', async () => {
    render(
      <MemoryRouter initialEntries={['/traffic']}>
        <TrafficInjectionPage />
      </MemoryRouter>,
    );
    const heading = await screen.findByText('Recent runs');
    const card = heading.closest('[id="recent-runs"]');
    expect(card).not.toBeNull();
  });

  it('scrolls the #recent-runs card into view when navigated to with that hash', async () => {
    const scrollIntoView = vi.fn();
    HTMLElement.prototype.scrollIntoView = scrollIntoView;

    render(
      <MemoryRouter initialEntries={['/traffic#recent-runs']}>
        <TrafficInjectionPage />
      </MemoryRouter>,
    );

    await waitFor(() => expect(scrollIntoView).toHaveBeenCalledWith({ behavior: 'smooth' }));
  });

  it('shows the Pro upgrade treatment instead of fault controls on Free', async () => {
    hasFeature.mockReturnValue(false);
    render(
      <MemoryRouter initialEntries={['/traffic']}>
        <TrafficInjectionPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText('Error injection requires NIAC Pro.')).toBeInTheDocument();
    expect(fetchErrorTypes).not.toHaveBeenCalled();
  });
});
