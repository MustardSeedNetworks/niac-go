import { expect, test } from '@playwright/test';

/**
 * Devices Page (Running Devices) Tests
 *
 * Cleanup PR-NAC5 pared this file from 5 tests down to 1. The
 * dropped tests were the same shape as the deleted device-crud
 * stubs (see PR-NAC2): four of five tests either discarded the
 * count() result, asserted only that <main> was visible, or wrapped
 * the assertion in `if (await x.isVisible()) { await expect(x).
 * toBeVisible(); }` — tautological (assert-visible after confirming-
 * visible). They contributed no merge-gate signal.
 *
 * The surviving test asserts the route registration: visiting
 * /devices keeps us on /devices (no silent redirect to /dashboard
 * or /). The richer interactions (Device list rendering, search/
 * filter, type filtering) need a deterministic backend or fixture
 * data — out of scope for the cleanup pass.
 */

test.describe('Devices', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/devices');
    await page.waitForLoadState('domcontentloaded');
  });

  test('/devices stays on /devices (no silent redirect)', async ({ page }) => {
    await expect(page).toHaveURL(/devices/);
  });
});
