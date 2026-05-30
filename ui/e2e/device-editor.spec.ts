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
    await page.goto('/device-config/new');
    await page.waitForLoadState('domcontentloaded');
  });

  test('/device-config/new stays on /device-config (no silent redirect)', async ({ page }) => {
    await expect(page).toHaveURL(/device-config/);
  });
});
