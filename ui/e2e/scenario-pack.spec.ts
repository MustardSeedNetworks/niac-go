import { expect, test } from '@playwright/test';

const request = {
  sites: [
    { code: 'MED', octet: 51, location: 'Regional Medical Center' },
    { code: 'AMB', octet: 52, location: 'Ambulatory Care Campus' },
    { code: 'CLD', octet: 53, location: 'Clinical Data Center' },
  ],
  counts: {
    siteWanRouters: 2,
    firewalls: 2,
    coreSwitches: 2,
    distributionSwitches: 4,
    accessSwitches: 12,
    serverSwitches: 2,
    accessPointsPerAccess: 3,
    workstationsPerAccess: 4,
    wirelessControllers: 2,
  },
  domain: 'care.example',
  snmpCommunity: 'NetAllyDemo',
  attachmentName: 'cyberscope',
};

const content = `devices:
  - name: MED-CORE-SW01
    type: switch
    mac: 02:00:00:00:00:01
`;
const generatedFleetContent = `${content}${'# generated fleet device metadata\n'.repeat(40_000)}`;

test('selects a versioned scenario pack and creates an editable draft', async ({ page }) => {
  let generated = false;
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
    await route.fallback();
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
          version: '1.0.0',
          manifestVersion: 1,
          name: 'Hospital network',
          description: 'Acute-care, ambulatory, and clinical data-center sites.',
          request,
          manifest: {
            deviceCount: 351,
            networkCount: 30,
            linkCount: 416,
            deviceNamesSha256: 'devices',
            networksSha256: 'networks',
            linksSha256: 'links',
          },
        },
      ],
    }),
  );
  await page.route('**/api/v1/scenario/generate', async (route) => {
    expect(route.request().method()).toBe('POST');
    expect(route.request().postDataJSON()).toEqual(request);
    generated = true;
    await route.fulfill({
      json: {
        content: generatedFleetContent,
        manifest: {
          deviceCount: 351,
          networkCount: 30,
          linkCount: 416,
          deviceNamesSha256: 'devices',
          networksSha256: 'networks',
          linksSha256: 'links',
        },
      },
    });
  });
  await page.route('**/api/v1/library/drafts', async (route) => {
    const body = route.request().postDataJSON() as { name: string; content: string };
    expect(new TextEncoder().encode(body.content).length).toBeGreaterThan(1 << 20);
    expect(body.content).toBe(generatedFleetContent);
    await route.fulfill({
      status: 201,
      json: {
        name: body.name,
        content,
        format: 'yaml',
        revision: 'revision-1',
        modifiedAt: '2026-07-29T12:00:00Z',
        sizeBytes: content.length,
      },
    });
  });

  await page.goto('/new-simulation');
  await page.getByTestId('wizard-interface-select').selectOption('lo0');
  await page.getByTestId('scenario-pack-hospital').click();
  await expect(page.getByTestId('fleet-domain')).toHaveValue('care.example');
  await page.getByTestId('wizard-select-fleet').click();
  await page.getByTestId('wizard-next-button').click();

  await expect.poll(() => generated).toBe(true);
  await expect(page.getByRole('tab', { name: 'Visual topology' })).toBeVisible();
});
