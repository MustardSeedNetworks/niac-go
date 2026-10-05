import { expect, test } from '@playwright/test';
import { disableAnimations } from './helpers/auth';

/**
 * AP-4 slice 4. The topology page is the primary attachment surface: click a
 * switch, see who is plugged into its tester ports, and move one of them.
 *
 * The dry-run E2E daemon captures no wire, so it observes no client; the
 * session status, graph and clients read are stubbed with what a running pool
 * session reports. Placement itself is asserted on the wire by AP-2's
 * wiretest.
 */

const port = (device: string, iface: string) => ({
  device,
  interface: iface,
  vlan: 210,
  network: 'clinical',
});

const running = {
  running: true,
  sessionId: 'hospital',
  interface: 'e2e-dry-run0',
  deviceCount: 2,
  uptimeSeconds: 42,
  fabric: {
    topology: {
      binding: {
        attachment: 'cyberscope',
        interface: 'e2e-dry-run0',
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
          network: 'clinical',
          ports: [
            port('MED-ACC-SW01', 'GigabitEthernet1/0/45'),
            port('MED-ACC-SW01', 'GigabitEthernet1/0/46'),
            port('MED-ACC-SW02', 'GigabitEthernet1/0/45'),
          ],
        },
      ],
    },
    forwarded: 0,
    drops: 0,
    received: 0,
    transmitted: 0,
  },
};

const DEVICES = [
  { name: 'MED-ACC-SW01', type: 'switch', ips: ['10.60.1.11'], protocols: ['lldp', 'snmp'] },
  { name: 'MED-ACC-SW02', type: 'switch', ips: ['10.60.1.12'], protocols: ['lldp', 'snmp'] },
];

const TOPOLOGY = {
  nodes: DEVICES.map((device) => ({ name: device.name, type: device.type })),
  links: [{ source: 'MED-ACC-SW01', target: 'MED-ACC-SW02', label: 'Te1/1/1', discovered: false }],
};

const seen = (mac: string, device: string, iface: string) => ({
  mac,
  device,
  interface: iface,
  firstSeen: '2026-09-26T08:00:00Z',
  lastSeen: new Date().toISOString(),
  expireAt: '2026-09-26T08:05:00Z',
  frames: 10,
});

const first = '00:c0:17:00:00:01';
const second = '00:c0:17:00:00:02';

test.beforeEach(async ({ page }) => {
  await disableAnimations(page);
});

test('a switch shows who is on its tester ports and moves one client to another switch', async ({
  page,
}) => {
  let clients = [
    seen(first, 'MED-ACC-SW01', 'GigabitEthernet1/0/45'),
    seen(second, 'MED-ACC-SW01', 'GigabitEthernet1/0/46'),
  ];
  const pins: unknown[] = [];

  await page.route('**/api/v1/simulation', async (route) => {
    if (route.request().method() !== 'GET') {
      await route.fallback();
      return;
    }
    await route.fulfill({ json: { ...running, sessions: [running] } });
  });
  await page.route('**/api/v1/sessions/*/**', async (route) => {
    const url = new URL(route.request().url());
    const resource = url.pathname.split('/').pop();
    if (resource === 'pins' && route.request().method() === 'POST') {
      const pin = route.request().postDataJSON();
      pins.push({ path: url.pathname, pin });
      // The daemon moves the pinned client to its new port on the running
      // session, and every other client stays where it was.
      clients = [seen(first, pin.device, pin.interface), clients[1]];
      await route.fulfill({ json: pin });
      return;
    }
    const bodies: Record<string, unknown> = {
      clients,
      topology: TOPOLOGY,
      devices: DEVICES,
      stats: {},
    };
    await route.fulfill({ json: bodies[resource ?? ''] ?? [] });
  });

  await page.goto('/topology');
  const nodes = page.getByTestId('topology-device-node');
  await expect(nodes).toHaveCount(DEVICES.length);
  await expect(nodes.first()).toBeVisible();

  await page.getByTestId('rf__node-MED-ACC-SW01').click();
  const pool = page.getByTestId('device-attachment-pool');
  await expect(pool).toBeVisible();
  const occupant = (iface: string) =>
    pool
      .getByTestId(`device-attachment-port-${iface}`)
      .getByTestId('device-attachment-port-occupant');
  await expect(occupant('GigabitEthernet1/0/45')).toHaveText(first);
  await expect(occupant('GigabitEthernet1/0/46')).toHaveText(second);

  // Both SW01 ports are held, so the one free port is on the other switch.
  const move = pool.getByTestId('attached-client-move');
  const portPicker = move.getByTestId('attached-client-move-port');
  await expect(portPicker.getByRole('option')).toHaveText([
    'Choose a free port',
    'MED-ACC-SW02 GigabitEthernet1/0/45',
  ]);
  await move.getByTestId('attached-client-move-mac').selectOption(first);
  await portPicker.selectOption({ label: 'MED-ACC-SW02 GigabitEthernet1/0/45' });
  await move.getByTestId('attached-client-move-submit').click();

  await expect(move.getByRole('status')).toHaveText(
    `Pinned ${first} to MED-ACC-SW02 GigabitEthernet1/0/45.`,
  );
  expect(pins).toEqual([
    {
      path: '/api/v1/sessions/hospital/pins',
      pin: { mac: first, device: 'MED-ACC-SW02', interface: 'GigabitEthernet1/0/45' },
    },
  ]);
  await expect(occupant('GigabitEthernet1/0/45')).toHaveText('Available');
  await expect(occupant('GigabitEthernet1/0/46')).toHaveText(second);

  await page.getByTestId('rf__node-MED-ACC-SW02').click();
  await expect(occupant('GigabitEthernet1/0/45')).toHaveText(first);
});
