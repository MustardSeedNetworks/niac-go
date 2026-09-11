import { expect, test } from '@playwright/test';

/**
 * AP-0. The wizard's attachment field was free text defaulting to the literal
 * `tester`, while every generated pack names its attachment `cyberscope`, and
 * the mode and VLAN were defaults the operator's --attachment-policy set was
 * never asked about. Out of the box that produced `unknown_attachment` with
 * nothing on screen naming the valid choice.
 *
 * This walks a generated pack through to a start and types no part of the
 * binding.
 */

const packRequest = {
  sites: [{ code: 'MED', octet: 51, location: 'Regional Medical Center' }],
  counts: {
    siteWanRouters: 1,
    firewalls: 1,
    coreSwitches: 1,
    distributionSwitches: 2,
    accessSwitches: 1,
    serverSwitches: 1,
    accessPointsPerAccess: 1,
    workstationsPerAccess: 1,
    wirelessControllers: 0,
  },
  domain: 'care.example',
  snmpCommunity: 'NetAllyDemo',
  attachmentName: 'cyberscope',
};

const manifest = {
  deviceCount: 5,
  networkCount: 3,
  linkCount: 4,
  deviceNamesSha256: 'devices',
  networksSha256: 'networks',
  linksSha256: 'links',
};

const content = `networks:
  - name: lab-transit
    subnet: 10.10.200.0/24
attachments:
  - name: cyberscope
    connect: lab-transit
devices:
  - name: MED-CORE-SW01
    type: switch
    mac: 02:00:00:00:00:01
`;

test('starts a generated pack without typing any part of the binding', async ({ page }) => {
  let startPayload: Record<string, unknown> | null = null;

  await page.route('**/api/v1/license', (route) =>
    route.fulfill({
      json: {
        tier: 'Pro',
        isActivated: true,
        isTrialMode: false,
        trialDaysRemaining: 0,
        features: [
          {
            id: 'config_templates',
            label: 'Configuration templates',
            description: 'Create simulations from reusable configuration templates.',
            granted: true,
          },
        ],
        licenseEnforced: true,
      },
    }),
  );
  await page.route('**/api/v1/simulation', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({ json: { running: false, interface: '', deviceCount: 0 } });
      return;
    }
    startPayload = route.request().postDataJSON() as Record<string, unknown>;
    await route.fulfill({
      json: {
        sessionId: 'scenario-200',
        running: true,
        interface: 'lo0',
        deviceCount: 5,
        uptimeSeconds: 0,
      },
    });
  });
  await page.route('**/api/v1/interfaces?filter=usable', (route) =>
    route.fulfill({
      json: {
        interfaces: [{ name: 'lo0', addresses: ['127.0.0.1'], isUp: true, isLoopback: true }],
      },
    }),
  );
  await page.route('**/api/v1/templates', (route) => route.fulfill({ json: [] }));
  await page.route('**/api/v1/library/networks', (route) => route.fulfill({ json: [] }));
  await page.route('**/api/v1/scenario/packs', (route) =>
    route.fulfill({
      json: [
        {
          id: 'hospital',
          version: '1.3.0',
          manifestVersion: 1,
          name: 'Hospital network',
          description: 'Acute-care site.',
          mapPurpose: 'presentation',
          request: packRequest,
          manifest,
        },
      ],
    }),
  );
  await page.route('**/api/v1/scenario/generate', (route) =>
    route.fulfill({ json: { content, manifest } }),
  );
  await page.route('**/api/v1/library/drafts', (route) =>
    route.fulfill({
      status: 201,
      json: {
        name: 'hospital',
        content,
        format: 'yaml',
        revision: 'revision-1',
        modifiedAt: '2026-09-11T12:00:00Z',
        sizeBytes: content.length,
      },
    }),
  );
  // The operator approved one access binding and one trunk binding on lo0.
  await page.route('**/api/v1/attachment-policies', (route) =>
    route.fulfill({
      json: {
        policies: [
          { interface: 'lo0', mode: 'access', accessVlan: 200 },
          { interface: 'lo0', mode: 'trunk', allowedVlans: [200, 210] },
        ],
      },
    }),
  );
  await page.route('**/api/v1/simulation/attachments', (route) =>
    route.fulfill({ json: { routed: true, attachments: ['cyberscope'] } }),
  );
  await page.route('**/api/v1/simulation/preflight', (route) =>
    route.fulfill({
      json: {
        safe: true,
        topology: {
          binding: {
            attachment: 'cyberscope',
            interface: 'lo0',
            mode: 'access',
            physicalVlan: 200,
            network: 'lab-transit',
            wireTagged: false,
          },
          networks: [{ name: 'lab-transit', prefix: '10.10.200.0/24' }],
          interfaces: [],
          routes: [],
          dhcpScopes: [],
        },
        diagnostics: [],
      },
    }),
  );

  await page.goto('/new-simulation');
  await page.getByTestId('wizard-interface-select').selectOption('lo0');
  await page.getByTestId('scenario-pack-hospital').click();
  await expect(page.getByTestId('fleet-domain')).toHaveValue('care.example');
  await page.getByTestId('wizard-select-fleet').click();
  for (let step = 0; step < 5; step += 1) {
    await page.getByTestId('wizard-next-button').click();
  }

  // The scenario's own attachment name, offered rather than typed.
  const attachment = page.getByTestId('wizard-attachment-name');
  await expect(attachment).toHaveValue('cyberscope');
  await expect(attachment.locator('option')).toHaveText(['cyberscope']);

  // Only the operator's approved modes and VLANs.
  await expect(page.getByTestId('wizard-attachment-mode').locator('option')).toHaveCount(2);
  await expect(page.getByTestId('wizard-access-vlan')).toHaveValue('200');

  await page.getByTestId('wizard-preflight-check').click();
  await page.getByTestId('wizard-preflight-start').click();

  await expect
    .poll(() => startPayload)
    .toMatchObject({ attachment: 'cyberscope', attachmentMode: 'access', accessVlan: 200 });
});
