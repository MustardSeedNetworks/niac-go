import { expect, test } from '@playwright/test';
import { withIsolatedDaemon } from './isolated-daemon';

/**
 * AP-4: the move works however the tester's NIC is bound. The re-pin restarts
 * the session from its own start request, so a restart that dropped the mode or
 * the wire VLAN would land the session on another binding, or be refused by
 * the policy that admitted it.
 *
 * Everything but the clients read is the real dry-run daemon: the dry run
 * captures no wire, so it observes no client.
 */

const scenario = `networks:
  - name: med-data
    subnet: 10.51.210.0/24
    virtual_vlan: 210
attachments:
  - name: cyberscope
    at:
      device: MED-ACC-SW01
      ports:
        - GigabitEthernet1/0/20
        - GigabitEthernet1/0/21
        - GigabitEthernet1/0/22
    pins:
      - mac: "00:c0:17:00:00:02"
        device: MED-ACC-SW01
        interface: GigabitEthernet1/0/22
devices:
  - name: MED-ACC-SW01
    type: switch
    mac: "02:00:00:00:00:01"
    interfaces:
      - name: Vlan210
        network: med-data
        address: 10.51.210.21/24
      - name: GigabitEthernet1/0/20
        vlans: [210]
      - name: GigabitEthernet1/0/21
        vlans: [210]
      - name: GigabitEthernet1/0/22
        vlans: [210]
`;

const moving = '00:c0:17:00:00:01';
const staying = '00:c0:17:00:00:02';

const seen = (mac: string, iface: string) => ({
  mac,
  device: 'MED-ACC-SW01',
  interface: iface,
  firstSeen: '2026-09-30T08:00:00Z',
  lastSeen: new Date().toISOString(),
  expireAt: '2099-01-01T00:00:00Z',
  frames: 10,
});

interface Pin {
  mac: string;
  device: string;
  interface: string;
}

interface Session {
  sessionId: string;
  running: boolean;
  attachmentMode?: string;
  physicalVlan?: number;
  fabric?: {
    topology?: {
      binding?: { mode?: string; wireTagged?: boolean };
      attachments?: { name: string; pins?: Pin[] }[];
    };
  };
}

const bindings = [
  { mode: 'direct', policy: 'e2e-dry-run0=direct', start: {}, vlan: undefined, tagged: false },
  {
    mode: 'access',
    policy: 'e2e-dry-run0=access:200',
    start: { accessVlan: 200 },
    vlan: 200,
    tagged: false,
  },
  {
    mode: 'trunk',
    policy: 'e2e-dry-run0=trunk:200,201',
    start: { accessVlan: 201 },
    vlan: 201,
    tagged: true,
  },
] as const;

for (const binding of bindings) {
  test(`re-pins a client on a ${binding.mode} binding and keeps the binding`, async ({
    page,
    request,
  }) => {
    await withIsolatedDaemon(
      request,
      async (baseURL) => {
        const session = async (): Promise<Session> => {
          const status = await request.get(`${baseURL}/api/v1/simulation`);
          expect(status.ok()).toBe(true);
          const [only] = ((await status.json()) as { sessions: Session[] }).sessions;
          return only;
        };
        const csrf = await request.get(`${baseURL}/api/v1/csrf-token`);
        const { token } = (await csrf.json()) as { token: string };
        const start = await request.post(`${baseURL}/api/v1/simulation`, {
          headers: { 'X-Csrf-Token': token },
          data: {
            // A trunk carries one session per tag, so it has to be named.
            sessionId: 'hospital',
            interface: 'e2e-dry-run0',
            attachment: 'cyberscope',
            attachmentMode: binding.mode,
            ...binding.start,
            configData: scenario,
          },
        });
        expect(start.ok(), await start.text()).toBe(true);
        const before = await session();
        expect(before.attachmentMode).toBe(binding.mode);
        expect(before.physicalVlan).toBe(binding.vlan);
        expect(before.fabric?.topology?.binding?.wireTagged ?? false).toBe(binding.tagged);

        await page.route('**/api/v1/sessions/*/clients', (route) =>
          route.fulfill({
            json: [seen(moving, 'GigabitEthernet1/0/20'), seen(staying, 'GigabitEthernet1/0/22')],
          }),
        );
        await page.goto(`${baseURL}/runtime`);
        const move = page.getByTestId('attached-client-move');
        const portPicker = move.getByTestId('attached-client-move-port');
        // The pool less the port each client holds.
        await expect(portPicker.getByRole('option')).toHaveText([
          'Choose a free port',
          'MED-ACC-SW01 GigabitEthernet1/0/21',
        ]);
        await move.getByTestId('attached-client-move-mac').selectOption(moving);
        await portPicker.selectOption({ label: 'MED-ACC-SW01 GigabitEthernet1/0/21' });
        await move.getByTestId('attached-client-move-submit').click();
        await expect(move.getByRole('status')).toHaveText(
          `Pinned ${moving} to MED-ACC-SW01 GigabitEthernet1/0/21. The scenario restarted.`,
        );

        const after = await session();
        expect(after.running).toBe(true);
        expect(after.sessionId).toBe('hospital');
        expect(after.attachmentMode).toBe(binding.mode);
        expect(after.physicalVlan).toBe(binding.vlan);
        expect(after.fabric?.topology?.binding?.wireTagged ?? false).toBe(binding.tagged);
        const pins = after.fabric?.topology?.attachments?.find(
          (attachment) => attachment.name === 'cyberscope',
        )?.pins;
        expect(pins).toEqual([
          { mac: staying, device: 'MED-ACC-SW01', interface: 'GigabitEthernet1/0/22' },
          { mac: moving, device: 'MED-ACC-SW01', interface: 'GigabitEthernet1/0/21' },
        ]);
      },
      binding.policy,
    );
  });
}
