import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { NeighborRecord } from '../api/types';

// Mock the API client at module-load time so the page picks up the mock
// during its useApiResource fetch. Vitest hoists vi.mock above imports.
vi.mock('../api/client', () => ({
  fetchNeighbors: vi.fn(),
}));

// Re-import after the mock so we can drive return values per test.
import { fetchNeighbors } from '../api/client';
import { NeighborsPage } from './NeighborsPage';

// Distinct chassis IDs so the device-name text is unambiguous in queries.
const SAMPLE: NeighborRecord[] = [
  {
    protocol: 'CDP',
    localDevice: 'router-1',
    remoteDevice: 'switch-2',
    remotePort: 'Gi0/1',
    remoteChassisId: 'aa:aa:aa:aa:aa:01',
    description: '',
    capabilities: [],
    managementAddress: '10.0.0.2',
    lastSeen: new Date().toISOString(),
    ttl: 180_000_000_000, // 180s in ns
  },
  {
    protocol: 'LLDP',
    localDevice: 'router-1',
    remoteDevice: 'firewall-3',
    remotePort: 'eth0',
    remoteChassisId: 'bb:bb:bb:bb:bb:02',
    description: '',
    capabilities: [],
    managementAddress: '',
    lastSeen: new Date().toISOString(),
    ttl: 120_000_000_000,
  },
];

describe('NeighborsPage', () => {
  beforeEach(() => {
    vi.mocked(fetchNeighbors).mockReset();
  });

  it('renders neighbor rows once data loads', async () => {
    vi.mocked(fetchNeighbors).mockResolvedValueOnce(SAMPLE);
    render(<NeighborsPage />);

    expect(await screen.findByText('switch-2')).toBeInTheDocument();
    expect(await screen.findByText('firewall-3')).toBeInTheDocument();
    expect(await screen.findByText('Gi0/1')).toBeInTheDocument();
  });

  it('shows the empty state when no neighbors are reported', async () => {
    vi.mocked(fetchNeighbors).mockResolvedValueOnce([]);
    render(<NeighborsPage />);

    await waitFor(() =>
      expect(
        screen.getByText(/No neighbors discovered yet/i),
      ).toBeInTheDocument(),
    );
  });

  it('filters by protocol when a chip is clicked', async () => {
    vi.mocked(fetchNeighbors).mockResolvedValue(SAMPLE);
    const user = userEvent.setup();
    render(<NeighborsPage />);

    await screen.findByText('switch-2');
    await screen.findByText('firewall-3');

    // Click the LLDP chip — switch-2 (CDP) should disappear, firewall-3 stays.
    await user.click(screen.getByRole('button', { name: /LLDP/i }));
    await waitFor(() => expect(screen.queryByText('switch-2')).not.toBeInTheDocument());
    expect(screen.getByText('firewall-3')).toBeInTheDocument();
  });

  it('filters by free-text search', async () => {
    vi.mocked(fetchNeighbors).mockResolvedValue(SAMPLE);
    const user = userEvent.setup();
    render(<NeighborsPage />);

    await screen.findByText('switch-2');

    const searchBox = screen.getByLabelText(/Filter neighbors/i);
    await user.type(searchBox, 'firewall');
    await waitFor(() => expect(screen.queryByText('switch-2')).not.toBeInTheDocument());
    expect(screen.getByText('firewall-3')).toBeInTheDocument();
  });

  it('renders the error state when the fetch fails', async () => {
    vi.mocked(fetchNeighbors).mockRejectedValueOnce(new Error('boom'));
    render(<NeighborsPage />);

    await waitFor(() =>
      expect(screen.getByText(/Failed to load neighbors/i)).toBeInTheDocument(),
    );
  });
});
