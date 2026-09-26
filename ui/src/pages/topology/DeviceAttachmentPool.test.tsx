// This isolated component fixture represents an authenticated operator.
vi.mock('../../contexts/ScopeContext', () => ({
  useActionPermission: () => ({ disabled: false }),
}));

import { screen, waitFor, within } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { AttachmentPin, CompiledAttachment, ObservedClient } from '../../api/types';
import { renderWithResources as render } from '../../test/renderWithResources';
import '../../i18n';
import { DeviceAttachmentPool } from './DeviceAttachmentPool';

const fetchSessionClients = vi.fn<(sessionId: string) => Promise<ObservedClient[]>>();

vi.mock('../../api/client', () => ({
  fetchSessionClients: (sessionId: string) => fetchSessionClients(sessionId),
  pinSessionClient: vi.fn(),
}));

const client = (mac: string, device: string, iface: string): ObservedClient => ({
  mac,
  device,
  interface: iface,
  firstSeen: '2026-09-26T08:00:00Z',
  lastSeen: new Date().toISOString(),
  expireAt: '2026-09-26T08:05:00Z',
  frames: 3,
});

const port = (device: string, iface: string) => ({
  device,
  interface: iface,
  vlan: 210,
  network: 'clinical',
});

// Three ports on SW01 and two on SW02; one SW01 port is pinned to a MAC the
// session has not seen.
const pool = (pins: AttachmentPin[] = []): CompiledAttachment => ({
  name: 'cyberscope',
  device: 'MED-ACC-SW01',
  network: 'clinical',
  ports: [
    port('MED-ACC-SW01', 'GigabitEthernet1/0/45'),
    port('MED-ACC-SW01', 'GigabitEthernet1/0/46'),
    port('MED-ACC-SW01', 'GigabitEthernet1/0/47'),
    port('MED-ACC-SW02', 'GigabitEthernet1/0/45'),
    port('MED-ACC-SW02', 'GigabitEthernet1/0/46'),
  ],
  pins,
});

const here = '00:c0:17:00:00:01';
const there = '00:c0:17:00:00:02';
const absent = '00:c0:17:00:00:03';

const occupant = (iface: string) =>
  within(screen.getByTestId(`device-attachment-port-${iface}`)).getByTestId(
    'device-attachment-port-occupant',
  );

describe('DeviceAttachmentPool', () => {
  beforeEach(() => {
    fetchSessionClients.mockReset();
  });

  it("lists this device's pool ports with who holds each", async () => {
    fetchSessionClients.mockResolvedValue([
      client(here, 'MED-ACC-SW01', 'GigabitEthernet1/0/45'),
      client(there, 'MED-ACC-SW02', 'GigabitEthernet1/0/46'),
    ]);
    render(
      <DeviceAttachmentPool
        sessionId="hospital"
        pool={pool([{ mac: absent, device: 'MED-ACC-SW01', interface: 'GigabitEthernet1/0/47' }])}
        device="MED-ACC-SW01"
      />,
    );

    await waitFor(() => expect(occupant('GigabitEthernet1/0/45')).toHaveTextContent(here));
    expect(fetchSessionClients).toHaveBeenCalledWith('hospital');
    expect(occupant('GigabitEthernet1/0/46')).toHaveTextContent('Free');
    expect(occupant('GigabitEthernet1/0/47')).toHaveTextContent(`Held for ${absent}`);
    // SW02's ports, and the client on one of them, belong to SW02's panel.
    expect(
      within(screen.getByTestId('device-attachment-pool')).getAllByRole('listitem'),
    ).toHaveLength(3);
    expect(screen.getByTestId('device-attachment-port-GigabitEthernet1/0/45')).toHaveTextContent(
      'VLAN 210',
    );
  });

  it('moves only the clients on this device, to any free port of the pool', async () => {
    fetchSessionClients.mockResolvedValue([
      client(here, 'MED-ACC-SW01', 'GigabitEthernet1/0/45'),
      client(there, 'MED-ACC-SW02', 'GigabitEthernet1/0/46'),
    ]);
    render(<DeviceAttachmentPool sessionId="hospital" pool={pool()} device="MED-ACC-SW01" />);

    const mac = await screen.findByTestId('attached-client-move-mac');
    expect(
      within(mac)
        .getAllByRole('option')
        .map((option) => option.textContent),
    ).toEqual([here]);
    // The other switch's occupied port is taken; its free one is offered.
    expect(
      within(screen.getByTestId('attached-client-move-port'))
        .getAllByRole('option')
        .map((option) => option.textContent),
    ).toEqual([
      'Choose a free port',
      'MED-ACC-SW01 GigabitEthernet1/0/46',
      'MED-ACC-SW01 GigabitEthernet1/0/47',
      'MED-ACC-SW02 GigabitEthernet1/0/45',
    ]);
  });

  it('offers no move when no client is on this device', async () => {
    fetchSessionClients.mockResolvedValue([client(there, 'MED-ACC-SW02', 'GigabitEthernet1/0/46')]);
    render(<DeviceAttachmentPool sessionId="hospital" pool={pool()} device="MED-ACC-SW01" />);

    expect(await screen.findAllByText('Free')).toHaveLength(3);
    expect(screen.queryByTestId('attached-client-move')).not.toBeInTheDocument();
  });

  it('says so when the clients cannot be read', async () => {
    fetchSessionClients.mockRejectedValue(new Error('daemon unavailable'));
    render(<DeviceAttachmentPool sessionId="hospital" pool={pool()} device="MED-ACC-SW01" />);

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Could not load attached clients: daemon unavailable',
    );
  });
});
