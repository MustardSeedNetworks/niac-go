import { expect, test } from '@playwright/test';

/**
 * AP-4 slice 1. Several testers can share one scenario, each on its own pool
 * port (AP-2). The runtime page has to say who is plugged in and where, so an
 * operator can tell two cables into two ports from two MACs on one segment.
 *
 * The dry-run E2E daemon captures no wire, so it observes no client; the
 * clients read is stubbed with what a running pool session reports. Placement
 * itself is asserted on the wire by AP-2's wiretest.
 */

const running = {
  running: true,
  sessionId: 'hospital',
  interface: 'e2e-dry-run0',
  deviceCount: 12,
  uptimeSeconds: 42,
};

const seen = (mac: string, device: string, port: string) => ({
  mac,
  device,
  interface: port,
  firstSeen: '2026-09-26T08:00:00Z',
  lastSeen: new Date().toISOString(),
  expireAt: '2026-09-26T08:05:00Z',
  frames: 10,
});

const first = '00:c0:17:00:00:01';
const second = '00:c0:17:00:00:02';

test('lists each attached client at its own device and port and follows a move', async ({
  page,
}) => {
  let clients = [
    seen(first, 'MED-ACC-SW01', 'GigabitEthernet1/0/45'),
    seen(second, 'MED-ACC-SW01', 'GigabitEthernet1/0/46'),
  ];
  const clientReads: string[] = [];

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
    if (resource === 'clients') {
      clientReads.push(url.pathname);
      await route.fulfill({ json: clients });
      return;
    }
    await route.fulfill({ json: resource === 'topology' ? { nodes: [], edges: [] } : [] });
  });

  await page.goto('/runtime');
  const card = page.getByTestId('attached-clients');
  await expect(card).toBeVisible();

  const firstRow = card.getByTestId(`attached-client-${first}`);
  const secondRow = card.getByTestId(`attached-client-${second}`);
  await expect(firstRow.getByTestId('attached-client-device')).toHaveText('MED-ACC-SW01');
  await expect(firstRow.getByTestId('attached-client-port')).toHaveText('GigabitEthernet1/0/45');
  await expect(secondRow.getByTestId('attached-client-device')).toHaveText('MED-ACC-SW01');
  await expect(secondRow.getByTestId('attached-client-port')).toHaveText('GigabitEthernet1/0/46');
  expect(clientReads.every((path) => path === '/api/v1/sessions/hospital/clients')).toBe(true);

  // The session re-places one tester on another switch; the next poll must
  // show it there and leave the other where it was.
  clients = [
    seen(first, 'MED-ACC-SW02', 'GigabitEthernet1/0/45'),
    seen(second, 'MED-ACC-SW01', 'GigabitEthernet1/0/46'),
  ];
  await expect(firstRow.getByTestId('attached-client-device')).toHaveText('MED-ACC-SW02', {
    timeout: 15000,
  });
  await expect(firstRow.getByTestId('attached-client-port')).toHaveText('GigabitEthernet1/0/45');
  await expect(secondRow.getByTestId('attached-client-device')).toHaveText('MED-ACC-SW01');
  await expect(secondRow.getByTestId('attached-client-port')).toHaveText('GigabitEthernet1/0/46');
});

const pooled = {
  ...running,
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
          device: 'MED-ACC-SW01',
          network: 'clinical',
          ports: [
            { device: 'MED-ACC-SW01', interface: 'GigabitEthernet1/0/45', network: 'clinical' },
            { device: 'MED-ACC-SW01', interface: 'GigabitEthernet1/0/46', network: 'clinical' },
            { device: 'MED-ACC-SW02', interface: 'GigabitEthernet1/0/45', network: 'clinical' },
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

test('re-pins one client to a free pool port and leaves the other where it was', async ({
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
    await route.fulfill({ json: { ...pooled, sessions: [pooled] } });
  });
  await page.route('**/api/v1/sessions/*/**', async (route) => {
    const url = new URL(route.request().url());
    const resource = url.pathname.split('/').pop();
    if (resource === 'pins' && route.request().method() === 'POST') {
      const pin = route.request().postDataJSON();
      pins.push({ path: url.pathname, pin });
      // The daemon restarts the session with the pinned client on its new
      // port. The stub keeps the other client where it was; a real restart
      // may re-place an unpinned client until AP-6 reloads in place.
      clients = [
        seen(first, pin.device, pin.interface),
        seen(second, 'MED-ACC-SW01', 'GigabitEthernet1/0/46'),
      ];
      await route.fulfill({ json: pin });
      return;
    }
    if (resource === 'clients') {
      await route.fulfill({ json: clients });
      return;
    }
    await route.fulfill({ json: resource === 'topology' ? { nodes: [], edges: [] } : [] });
  });

  await page.goto('/runtime');
  const move = page.getByTestId('attached-client-move');
  await expect(move).toBeVisible();

  // The free ports are the pool less the two the clients hold.
  const portPicker = move.getByTestId('attached-client-move-port');
  await expect(portPicker.getByRole('option')).toHaveText([
    'Choose a free port',
    'MED-ACC-SW02 GigabitEthernet1/0/45',
  ]);
  await move.getByTestId('attached-client-move-mac').selectOption(first);
  await portPicker.selectOption({ label: 'MED-ACC-SW02 GigabitEthernet1/0/45' });
  await move.getByTestId('attached-client-move-submit').click();

  await expect(move.getByRole('status')).toHaveText(
    `Pinned ${first} to MED-ACC-SW02 GigabitEthernet1/0/45. The scenario restarted.`,
  );
  expect(pins).toEqual([
    {
      path: '/api/v1/sessions/hospital/pins',
      pin: { mac: first, device: 'MED-ACC-SW02', interface: 'GigabitEthernet1/0/45' },
    },
  ]);

  const card = page.getByTestId('attached-clients');
  const firstRow = card.getByTestId(`attached-client-${first}`);
  const secondRow = card.getByTestId(`attached-client-${second}`);
  await expect(firstRow.getByTestId('attached-client-device')).toHaveText('MED-ACC-SW02');
  await expect(firstRow.getByTestId('attached-client-port')).toHaveText('GigabitEthernet1/0/45');
  await expect(secondRow.getByTestId('attached-client-device')).toHaveText('MED-ACC-SW01');
  await expect(secondRow.getByTestId('attached-client-port')).toHaveText('GigabitEthernet1/0/46');
});
