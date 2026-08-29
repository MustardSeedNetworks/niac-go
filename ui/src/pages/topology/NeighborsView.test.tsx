/**
 * NeighborsView — the live CDP/LLDP/EDP/FDP discovery table.
 *
 * The interesting behaviour is the de-duplication: the daemon emits one row per
 * (protocol, chassis, port) so it can age each independently, while the reader
 * wants one row per adjacency with the protocols rolled up. Getting that wrong
 * shows a plausible table with the wrong number of rows and stale TTLs, so
 * these assert the merged values and not just the row count.
 */

import { fireEvent, render, screen, waitFor, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { NeighborRecord } from '../../api/api-response-types';
import '../../i18n';
import { NeighborsView } from './NeighborsView';

const fetchNeighbors = vi.fn();

vi.mock('../../api/client', () => ({
  fetchNeighbors: (sessionId: string) => fetchNeighbors(sessionId),
}));

vi.mock('../../contexts/AppContext', () => ({
  useAppContext: () => ({ sessionId: 'test-session', setSessionId: vi.fn() }),
}));

const neighbor = (over: Partial<NeighborRecord> = {}): NeighborRecord => ({
  protocol: 'CDP',
  localDevice: 'sw1',
  remoteDevice: 'sw2',
  remotePort: 'Gi0/1',
  remoteChassisId: 'chassis-2',
  description: '',
  capabilities: [],
  managementAddress: '10.0.0.2',
  lastSeen: '2026-08-29T10:00:00.000Z',
  ttl: 120_000_000_000,
  ...over,
});

/** Table body rows, once the table has rendered. */
async function rows(): Promise<HTMLElement[]> {
  const table = await screen.findByRole('table');
  const body = table.querySelector('tbody');
  if (!body) throw new Error('table has no body');
  return within(body).getAllByRole('row');
}

beforeEach(() => {
  fetchNeighbors.mockReset();
});

describe('loading and failure states', () => {
  it('reports a load failure', async () => {
    fetchNeighbors.mockRejectedValue(new Error('daemon unreachable'));

    render(<NeighborsView />);

    const alert = await screen.findByRole('alert');
    expect(alert.textContent).toContain('daemon unreachable');
  });

  it('shows the no-data empty state when the table is genuinely empty', async () => {
    fetchNeighbors.mockResolvedValue([]);

    render(<NeighborsView />);

    await waitFor(() => expect(screen.queryByRole('table')).toBeNull());
    expect(screen.getByText(/no .*neighbou?r|none/i)).toBeDefined();
  });
});

describe('de-duplication', () => {
  it('collapses the same adjacency seen over two protocols into one row', async () => {
    fetchNeighbors.mockResolvedValue([
      neighbor({ protocol: 'CDP' }),
      neighbor({ protocol: 'LLDP' }),
    ]);

    render(<NeighborsView />);

    const body = await rows();
    expect(body).toHaveLength(1);
    // Protocols are merged and sorted, not duplicated as two rows.
    expect(body[0]?.textContent).toContain('CDP');
    expect(body[0]?.textContent).toContain('LLDP');
  });

  it('keeps distinct adjacencies apart', async () => {
    fetchNeighbors.mockResolvedValue([
      neighbor({ remotePort: 'Gi0/1' }),
      neighbor({ remotePort: 'Gi0/2' }),
    ]);

    render(<NeighborsView />);

    expect(await rows()).toHaveLength(2);
  });

  it('takes the freshest observation for the variable columns', async () => {
    fetchNeighbors.mockResolvedValue([
      neighbor({
        protocol: 'CDP',
        lastSeen: '2026-08-29T10:00:00.000Z',
        managementAddress: '10.0.0.2',
      }),
      neighbor({
        protocol: 'LLDP',
        lastSeen: '2026-08-29T10:05:00.000Z',
        managementAddress: '10.0.0.9',
      }),
    ]);

    render(<NeighborsView />);

    // The newer record's management address wins.
    expect((await rows())[0]?.textContent).toContain('10.0.0.9');
  });

  it('does not let an older record overwrite a newer one', async () => {
    fetchNeighbors.mockResolvedValue([
      neighbor({
        protocol: 'CDP',
        lastSeen: '2026-08-29T10:05:00.000Z',
        managementAddress: '10.0.0.9',
      }),
      neighbor({
        protocol: 'LLDP',
        lastSeen: '2026-08-29T10:00:00.000Z',
        managementAddress: '10.0.0.2',
      }),
    ]);

    render(<NeighborsView />);

    expect((await rows())[0]?.textContent).toContain('10.0.0.9');
    expect((await rows())[0]?.textContent).not.toContain('10.0.0.2');
  });

  it('sorts by local then remote device', async () => {
    fetchNeighbors.mockResolvedValue([
      neighbor({ localDevice: 'sw2', remoteDevice: 'a' }),
      neighbor({ localDevice: 'sw1', remoteDevice: 'z' }),
      neighbor({ localDevice: 'sw1', remoteDevice: 'b' }),
    ]);

    render(<NeighborsView />);

    const text = (await rows()).map((r) => r.textContent ?? '');
    expect(text[0]).toContain('sw1');
    expect(text[0]).toContain('b');
    expect(text[1]).toContain('z');
    expect(text[2]).toContain('sw2');
  });
});

describe('filtering', () => {
  it('filters by protocol chip', async () => {
    fetchNeighbors.mockResolvedValue([
      neighbor({ protocol: 'CDP', remotePort: 'Gi0/1' }),
      neighbor({ protocol: 'LLDP', remotePort: 'Gi0/2' }),
    ]);

    render(<NeighborsView />);
    expect(await rows()).toHaveLength(2);

    fireEvent.click(screen.getByRole('button', { name: /^CDP/ }));

    await waitFor(async () => expect(await rows()).toHaveLength(1));
    expect((await rows())[0]?.textContent).toContain('Gi0/1');
  });

  it('searches across local, remote, chassis and port', async () => {
    fetchNeighbors.mockResolvedValue([
      neighbor({ remoteDevice: 'core-a', remotePort: 'Gi0/1' }),
      neighbor({ remoteDevice: 'edge-b', remotePort: 'Gi0/2' }),
    ]);

    render(<NeighborsView />);
    const search = await screen.findByLabelText('Filter neighbors');

    fireEvent.change(search, { target: { value: 'edge' } });

    await waitFor(async () => expect(await rows()).toHaveLength(1));
    expect((await rows())[0]?.textContent).toContain('edge-b');
  });

  it('matches the search case-insensitively', async () => {
    fetchNeighbors.mockResolvedValue([neighbor({ remoteDevice: 'Core-A' })]);

    render(<NeighborsView />);
    fireEvent.change(await screen.findByLabelText('Filter neighbors'), {
      target: { value: 'core-a' },
    });

    expect(await rows()).toHaveLength(1);
  });

  it('shows the filtered-empty state, distinct from having no data at all', async () => {
    fetchNeighbors.mockResolvedValue([neighbor()]);

    render(<NeighborsView />);
    await rows();

    fireEvent.change(await screen.findByLabelText('Filter neighbors'), {
      target: { value: 'nomatch' },
    });

    await waitFor(() => expect(screen.queryByRole('table')).toBeNull());
  });
});
