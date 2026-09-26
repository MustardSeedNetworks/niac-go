import { screen, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { ObservedClient } from '../../api/types';
import { renderWithResources as render } from '../../test/renderWithResources';
import '../../i18n';
import { AttachedClientsCard } from './AttachedClientsCard';

const fetchSessionClients = vi.fn<(sessionId: string) => Promise<ObservedClient[]>>();

vi.mock('../../api/client', () => ({
  fetchSessionClients: (sessionId: string) => fetchSessionClients(sessionId),
}));

const client = (overrides: Partial<ObservedClient>): ObservedClient => ({
  mac: '00:c0:17:00:00:01',
  firstSeen: '2026-09-26T08:00:00Z',
  lastSeen: new Date().toISOString(),
  expireAt: '2026-09-26T08:05:00Z',
  frames: 3,
  ...overrides,
});

describe('AttachedClientsCard', () => {
  beforeEach(() => {
    fetchSessionClients.mockReset().mockResolvedValue([]);
  });

  it('reads the clients of the session it was given', async () => {
    render(<AttachedClientsCard sessionId="hospital" />);
    await screen.findByText('No client has sent a frame into this scenario yet.');
    expect(fetchSessionClients).toHaveBeenCalledWith('hospital');
  });

  it('shows each client at its own device and port', async () => {
    fetchSessionClients.mockResolvedValue([
      client({
        mac: '00:c0:17:00:00:01',
        ip: '10.51.210.20',
        device: 'MED-ACC-SW01',
        interface: 'GigabitEthernet1/0/45',
      }),
      client({
        mac: '00:c0:17:00:00:02',
        device: 'MED-ACC-SW02',
        interface: 'GigabitEthernet1/0/46',
        vlan: 210,
      }),
    ]);
    render(<AttachedClientsCard sessionId="hospital" />);

    const first = await screen.findByTestId('attached-client-00:c0:17:00:00:01');
    expect(within(first).getByTestId('attached-client-device')).toHaveTextContent('MED-ACC-SW01');
    expect(within(first).getByTestId('attached-client-port')).toHaveTextContent(
      'GigabitEthernet1/0/45',
    );
    expect(first).toHaveTextContent('10.51.210.20');
    expect(first).toHaveTextContent('untagged');

    const second = screen.getByTestId('attached-client-00:c0:17:00:00:02');
    expect(within(second).getByTestId('attached-client-device')).toHaveTextContent('MED-ACC-SW02');
    expect(within(second).getByTestId('attached-client-port')).toHaveTextContent(
      'GigabitEthernet1/0/46',
    );
    expect(second).toHaveTextContent('210');
  });

  it('says a client is not placed rather than leaving its port blank', async () => {
    fetchSessionClients.mockResolvedValue([client({})]);
    render(<AttachedClientsCard sessionId="hospital" />);

    const row = await screen.findByTestId('attached-client-00:c0:17:00:00:01');
    expect(within(row).getByTestId('attached-client-device')).toHaveTextContent('Not placed');
    expect(within(row).getByTestId('attached-client-port')).toHaveTextContent('Not placed');
  });

  it('reports a failed read instead of an empty list', async () => {
    fetchSessionClients.mockRejectedValue(new Error('daemon unreachable'));
    render(<AttachedClientsCard sessionId="hospital" />);

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Could not load attached clients: daemon unreachable',
    );
    expect(
      screen.queryByText('No client has sent a frame into this scenario yet.'),
    ).not.toBeInTheDocument();
  });
});
