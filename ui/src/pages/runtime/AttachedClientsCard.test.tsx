// This isolated component fixture represents an authenticated operator.
vi.mock('../../contexts/ScopeContext', () => ({
  useActionPermission: () => ({ disabled: false }),
}));

import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ApiError } from '../../api/errors';
import type { AttachmentPin, ObservedClient, SimulationStatus } from '../../api/types';
import { renderWithResources as render } from '../../test/renderWithResources';
import '../../i18n';
import { AttachedClientsCard } from './AttachedClientsCard';

const fetchSessionClients = vi.fn<(sessionId: string) => Promise<ObservedClient[]>>();
const pinSessionClient = vi.fn<(sessionId: string, pin: AttachmentPin) => Promise<AttachmentPin>>();

vi.mock('../../api/client', () => ({
  fetchSessionClients: (sessionId: string) => fetchSessionClients(sessionId),
  pinSessionClient: (sessionId: string, pin: AttachmentPin) => pinSessionClient(sessionId, pin),
}));

const client = (overrides: Partial<ObservedClient>): ObservedClient => ({
  mac: '00:c0:17:00:00:01',
  firstSeen: '2026-09-26T08:00:00Z',
  lastSeen: new Date().toISOString(),
  expireAt: '2026-09-26T08:05:00Z',
  frames: 3,
  ...overrides,
});

const port = (device: string, iface: string) => ({
  device,
  interface: iface,
  vlan: 210,
  network: 'clinical',
});

// A running session bound to the `cyberscope` pool: four free ports over two
// access switches, one of them already pinned to another MAC.
const poolFabric = (pins: AttachmentPin[] = []): SimulationStatus['fabric'] => ({
  topology: {
    binding: {
      attachment: 'cyberscope',
      interface: 'eth1',
      mode: 'direct',
      network: 'clinical',
      wireTagged: false,
    },
    networks: [],
    interfaces: [],
    routes: [],
    dhcpScopes: [],
    attachments: [
      {
        name: 'cyberscope',
        device: 'MED-ACC-SW01',
        network: 'clinical',
        ports: [
          port('MED-ACC-SW01', 'GigabitEthernet1/0/45'),
          port('MED-ACC-SW01', 'GigabitEthernet1/0/46'),
          port('MED-ACC-SW02', 'GigabitEthernet1/0/45'),
          port('MED-ACC-SW02', 'GigabitEthernet1/0/46'),
        ],
        pins,
      },
    ],
  },
  forwarded: 0,
  drops: 0,
  received: 0,
  transmitted: 0,
});

const portOptions = () =>
  within(screen.getByTestId('attached-client-move-port'))
    .getAllByRole('option')
    .map((option) => option.textContent);

describe('AttachedClientsCard', () => {
  beforeEach(() => {
    fetchSessionClients.mockReset().mockResolvedValue([]);
    pinSessionClient.mockReset();
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

  describe('moving a client', () => {
    const placed = [
      client({
        mac: '00:c0:17:00:00:01',
        device: 'MED-ACC-SW01',
        interface: 'GigabitEthernet1/0/45',
      }),
      client({
        mac: '00:c0:17:00:00:02',
        device: 'MED-ACC-SW01',
        interface: 'GigabitEthernet1/0/46',
      }),
    ];

    it('offers no move when the attachment names a network rather than a pool', async () => {
      fetchSessionClients.mockResolvedValue(placed);
      const fabric = poolFabric();
      if (fabric) fabric.topology.attachments = [];
      render(<AttachedClientsCard sessionId="hospital" fabric={fabric} />);

      await screen.findByTestId('attached-client-00:c0:17:00:00:01');
      expect(screen.queryByTestId('attached-client-move')).not.toBeInTheDocument();
    });

    it('offers only the pool ports nobody else holds', async () => {
      fetchSessionClients.mockResolvedValue(placed);
      render(
        <AttachedClientsCard
          sessionId="hospital"
          fabric={poolFabric([
            {
              mac: '00:c0:17:00:00:09',
              device: 'MED-ACC-SW02',
              interface: 'GigabitEthernet1/0/46',
            },
          ])}
        />,
      );

      await screen.findByTestId('attached-client-move');
      // Its own port, the other client's port and another MAC's pin are all out.
      expect(portOptions()).toEqual(['Choose a free port', 'MED-ACC-SW02 GigabitEthernet1/0/45']);
    });

    it('pins the chosen client to the chosen port and re-reads the clients', async () => {
      fetchSessionClients.mockResolvedValue(placed);
      pinSessionClient.mockImplementation((_session, pin) => Promise.resolve(pin));
      render(<AttachedClientsCard sessionId="hospital" fabric={poolFabric()} />);

      fireEvent.change(await screen.findByTestId('attached-client-move-mac'), {
        target: { value: '00:c0:17:00:00:02' },
      });
      fireEvent.change(screen.getByTestId('attached-client-move-port'), {
        target: { value: 'MED-ACC-SW02|GigabitEthernet1/0/46' },
      });
      const reads = fetchSessionClients.mock.calls.length;
      fireEvent.click(screen.getByTestId('attached-client-move-submit'));

      expect(await screen.findByRole('status')).toHaveTextContent(
        'Pinned 00:c0:17:00:00:02 to MED-ACC-SW02 GigabitEthernet1/0/46. The scenario restarted.',
      );
      expect(pinSessionClient).toHaveBeenCalledWith('hospital', {
        mac: '00:c0:17:00:00:02',
        device: 'MED-ACC-SW02',
        interface: 'GigabitEthernet1/0/46',
      });
      await waitFor(() => expect(fetchSessionClients.mock.calls.length).toBeGreaterThan(reads));
    });

    it('cannot submit before a port is chosen', async () => {
      fetchSessionClients.mockResolvedValue(placed);
      render(<AttachedClientsCard sessionId="hospital" fabric={poolFabric()} />);

      expect(await screen.findByTestId('attached-client-move-submit')).toBeDisabled();
    });

    it('shows why the daemon refused the pin', async () => {
      fetchSessionClients.mockResolvedValue(placed);
      pinSessionClient.mockRejectedValue(
        new ApiError('Scenario preflight failed', 400, 'preflight_failed', [
          { field: 'attachments[0].pins[0]', issue: 'port is held by another pin' },
        ]),
      );
      render(<AttachedClientsCard sessionId="hospital" fabric={poolFabric()} />);

      fireEvent.change(await screen.findByTestId('attached-client-move-port'), {
        target: { value: 'MED-ACC-SW02|GigabitEthernet1/0/45' },
      });
      fireEvent.click(screen.getByTestId('attached-client-move-submit'));

      const alert = await screen.findByRole('alert');
      expect(alert).toHaveTextContent('Could not move the client: Scenario preflight failed');
      expect(alert).toHaveTextContent('attachments[0].pins[0]: port is held by another pin');
      expect(screen.queryByRole('status')).not.toBeInTheDocument();
    });
  });
});
