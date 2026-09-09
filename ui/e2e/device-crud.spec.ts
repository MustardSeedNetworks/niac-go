import { expect, test } from '@playwright/test';

/**
 * Device CRUD Workflow Tests
 *
 * Cleanup PR-NAC2 pared this file from 244 lines / 14 tests down to
 * 2 contract-driven tests. The original suite was a textbook example
 * of `if (await x.isVisible()) { do stuff }` testing: 12 of the 14
 * tests had no hard assertions at all — they wrapped every action in
 * a visibility gate so the test passed whether the button existed or
 * not. See PR-NAC2 description for the full breakdown.
 *
 * The two surviving tests assert real merge-gate contracts:
 *  - The Add Device button (data-testid="device-add", landed in
 *    PR-N4) routes to /device-config/.
 *  - The /device-config/new form actually renders text inputs.
 *
 * These checks declare an empty loaded configuration through an inventory
 * fixture. The genuinely absent-configuration path is exercised against
 * an isolated daemon in first-run.acceptance.ts.
 */

test.describe('Device CRUD', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/api/v1/config/devices', (route) =>
      route.fulfill({ json: { devices: [], totalCount: 0, configurationLoaded: true } }),
    );
    // `/devices` is the read-only Running Devices live view; the
    // Device Library (the page hosting <DeviceListHeader> + the Add
    // Device button) lives at `/device-config`. The original 14-test
    // file went to `/devices` and never tripped this because every
    // assertion was gated by `if-visible` (see PR-NAC2 description).
    await page.goto('/device-config');
    await page.waitForLoadState('domcontentloaded');
  });

  test('Add Device button routes to /device-config/', async ({ page }) => {
    const addButton = page.getByTestId('device-add');
    await expect(addButton).toBeVisible();
    await addButton.click();
    await expect(page).toHaveURL(/\/device-config\/new$/);
  });

  test('/device-config/new renders text inputs', async ({ page }) => {
    await page.goto('/device-config/new');
    await expect(page.locator('input[type="text"]').first()).toBeVisible();
    expect(await page.locator('input[type="text"]').count()).toBeGreaterThan(0);
  });
});
