/**
 * TopologyPage.error.test.tsx — a failed load must not look like an empty
 * network.
 *
 * The page destructured only `data` and `loading` from useApiResource, so a
 * daemon it could not reach fell through to the "no topology data" branch and
 * rendered exactly like a network with no devices in it. handleRefresh's own
 * comment claimed the errors surfaced "through the per-hook error states",
 * which the page never read.
 */
import { render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AppProvider } from '../contexts/AppContext';
import '../i18n';
import { TopologyPage } from './TopologyPage';

const fetchTopology = vi.fn();
const fetchDevices = vi.fn();
const fetchNeighbors = vi.fn();
const fetchSimulationStatus = vi.fn();

vi.mock('../api/client', async () => {
  const actual = await vi.importActual<Record<string, unknown>>('../api/client');
  return {
    ...actual,
    fetchTopology: () => fetchTopology(),
    fetchDevices: () => fetchDevices(),
    fetchNeighbors: () => fetchNeighbors(),
    fetchSimulationStatus: () => fetchSimulationStatus(),
  };
});

function renderPage() {
  return render(
    <MemoryRouter>
      <AppProvider>
        <TopologyPage />
      </AppProvider>
    </MemoryRouter>,
  );
}

describe('TopologyPage load failure', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    fetchNeighbors.mockResolvedValue([]);
    // The page only fetches once a session exists, so without one there is
    // nothing to fail and nothing to assert.
    fetchSimulationStatus.mockResolvedValue({
      running: true,
      sessionId: 'once',
      sessions: [{ sessionId: 'once', running: true }],
    });
  });

  it('reports a failed topology fetch instead of an empty graph', async () => {
    fetchTopology.mockRejectedValue(new Error('daemon unreachable'));
    fetchDevices.mockResolvedValue([]);

    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId('topology-load-error')).toBeInTheDocument();
    });
    expect(screen.getByRole('alert')).toBeInTheDocument();
    expect(screen.getByText(/daemon unreachable/)).toBeInTheDocument();
  });

  it('reports a failed device fetch too', async () => {
    fetchTopology.mockResolvedValue({ devices: [], links: [] });
    fetchDevices.mockRejectedValue(new Error('devices endpoint returned 500'));

    renderPage();

    await waitFor(() => {
      expect(screen.getByTestId('topology-load-error')).toBeInTheDocument();
    });
    expect(screen.getByText(/devices endpoint returned 500/)).toBeInTheDocument();
  });

  it('still shows the empty state when both fetches succeed with nothing', async () => {
    fetchTopology.mockResolvedValue({ devices: [], links: [] });
    fetchDevices.mockResolvedValue([]);

    renderPage();

    await waitFor(() => {
      expect(screen.queryByTestId('topology-load-error')).not.toBeInTheDocument();
    });
  });
});
