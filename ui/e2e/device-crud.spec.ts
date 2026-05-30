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
 * Anything richer (validation errors, save round-trips, clone, delete
 * confirmation, search filtering) needs a real fake backend or seeded
 * data — better suited to a Vitest unit on the form component or a
 * fullstack tier spec. Adding deterministic mocks here is out of
 * scope for the cleanup pass.
 */

test.describe('Device CRUD', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/devices');
    await page.waitForLoadState('domcontentloaded');
  });

  test('Add Device button routes to /device-config/', async ({ page }) => {
    const addButton = page.getByTestId('device-add');
    await expect(addButton).toBeVisible();
    await addButton.click();
    await expect(page).toHaveURL(/device-config/);
  });

  test('/device-config/new renders text inputs', async ({ page }) => {
    await page.goto('/device-config/new');
    await expect(page.locator('input[type="text"]').first()).toBeVisible();
    expect(await page.locator('input[type="text"]').count()).toBeGreaterThan(0);
  });
});
