import { expect, test } from '@playwright/test';

/**
 * Device Editor Page Tests
 *
 * Cleanup PR-NAC5 pared this file from 8 tests down to 1. The
 * dropped tests were:
 *
 *  - "should display editor form" (asserted only that <main> was
 *    visible — true on every routed page)
 *  - "should have device name field" / "should have device type
 *    selector" / "should have IP address field" / "should have
 *    save and cancel buttons" — each obtained a count() and
 *    discarded it; NO assertion (same shape as the deleted
 *    config-workflow stubs).
 *  - "should have protocol configuration sections" — wrapped each
 *    iteration in `if (visible) { assert visible }` (tautological).
 *  - "should validate required fields" — `if submitButton visible,
 *    expect disabled` — gated the only assertion behind a
 *    visibility check, so the test passed silently whenever the
 *    regex selector `getByRole({name: /save|submit|create/i})`
 *    missed (a strict-mode-and-i18n-fragile pattern).
 *
 * The surviving test asserts the route registration: visiting
 * /device-config/new keeps us on /device-config (no silent redirect
 * to /devices or /).
 *
 * Real form-render coverage already lives in device-crud.spec.ts
 * post-PR-NAC2 ("/device-config/new renders text inputs"). Real
 * validation coverage needs a deterministic backend.
 */

test.describe('Device Editor', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/api/v1/**', async (route) => {
      const url = new URL(route.request().url());

      if (url.pathname === '/api/v1/auth/scope') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ scope: 'admin' }),
        });
        return;
      }

      if (url.pathname.startsWith('/api/v1/device-schemas/')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            type: 'switch',
            label: 'Switch',
            visible_sections: ['basic', 'interfaces'],
          }),
        });
        return;
      }

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({}),
      });
    });
    await page.goto('/device-config/new');
    await page.waitForLoadState('domcontentloaded');
  });

  test('/device-config/new stays on /device-config (no silent redirect)', async ({ page }) => {
    await expect(page).toHaveURL(/device-config/);
  });
});

/**
 * NOTE: this describe block stubs `**\/api/v1/**` with page.route() and fulfills
 * 201 unconditionally, so it asserts the shape the client *sends* and never
 * exercises the server that would reject it. That is exactly how D11 shipped:
 * the payload below was snake_cased, every real POST returned 400, and this
 * test stayed green. Contract coverage lives in
 * ui/src/api/wire-casing.contract.test.ts and the Go handler tests; treat this
 * spec as UI wiring only.
 */
test.describe('Device Editor API wiring', () => {
  test('creates a device with interface speed, duplex, and status metadata', async ({ page }) => {
    let createPayload: Record<string, unknown> | undefined;

    await page.route('**/api/v1/**', async (route) => {
      const request = route.request();
      const url = new URL(request.url());

      if (url.pathname === '/api/v1/csrf-token') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ token: 'test-csrf-token' }),
        });
        return;
      }

      if (url.pathname === '/api/v1/auth/scope') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ scope: 'admin' }),
        });
        return;
      }

      if (url.pathname === '/api/v1/files' && url.searchParams.get('kind') === 'walks') {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify([]),
        });
        return;
      }

      if (url.pathname.startsWith('/api/v1/device-schemas/')) {
        await route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({
            type: 'switch',
            label: 'Switch',
            visible_sections: ['basic', 'interfaces'],
          }),
        });
        return;
      }

      if (url.pathname === '/api/v1/config/devices' && request.method() === 'POST') {
        createPayload = request.postDataJSON() as Record<string, unknown>;
        await route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            success: true,
            device: createPayload,
          }),
        });
        return;
      }

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({}),
      });
    });

    await page.goto('/device-config/new');
    await expect(page.getByLabel('Hostname')).toBeEnabled();
    await page.getByLabel('Hostname').fill('edge-switch-01');
    await page.getByLabel('MAC Address').fill('02:00:00:00:10:01');
    await page.getByLabel('Primary IP Address').fill('192.0.2.11');

    await page.getByRole('button', { name: /interfaces section, collapsed/i }).click();
    await page.getByTestId('add-interface').click();
    await page.getByLabel('Interface 1 name').fill('Ethernet1/1');
    await page.getByLabel('Interface 1 speed Mbps').fill('10000');
    await page.getByLabel('Interface 1 duplex').selectOption('full');
    await page.getByLabel('Interface 1 admin status').selectOption('up');
    await page.getByLabel('Interface 1 oper status').selectOption('down');
    await page.getByLabel('Interface 1 description').fill('uplink to core');

    await page.getByTestId('device-editor-save').click();

    await expect(page.getByRole('alert')).toContainText('Device created successfully');
    expect(createPayload).toMatchObject({
      hostname: 'edge-switch-01',
      mac: '02:00:00:00:10:01',
      ip: '192.0.2.11',
      interfaceDetails: [
        {
          name: 'Ethernet1/1',
          speed: 10000,
          duplex: 'full',
          adminStatus: 'up',
          operStatus: 'down',
          description: 'uplink to core',
        },
      ],
    });
  });
});
