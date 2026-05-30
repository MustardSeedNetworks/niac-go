import { expect, test } from '@playwright/test';

/**
 * Navigation Tests
 *
 * Cleanup PR-NAC5 pared this file from 6 tests down to 2.
 *
 * The three "should navigate to <page>" tests (devices/topology/
 * templates) used `getByRole('link', { name: /<word>/i }).first()`
 * wrapped in `if (visible) { click; assertUrl; }`. Two problems:
 *
 *  1. The niac sidebar uses <button> for nav, not <a>/<link> — so
 *     getByRole('link', {...}) never matched. The if-visible gate
 *     turned that into a silent pass for every test, every CI run.
 *
 *  2. The `/template/i` regex would have hit a stale route anyway:
 *     ui/src/App.tsx redirects /templates → /runtime via
 *     <Navigate replace>. The test was triple-broken (wrong tag,
 *     wrong route, gate suppressed the failure).
 *
 * The "should preserve page state on navigation" test had zero
 * assertions — just three goto() calls and a comment that read
 * "Page should load". Pure no-op.
 *
 * Two surviving tests assert real contracts:
 *  - Main navigation surface is in the DOM and visible at `/`.
 *  - Back-button history navigation works (goto A → goto B → back
 *    lands on A).
 *
 * Per-page navigation coverage is implicit in the smoke spec
 * (which visits each route) and in the per-page specs
 * (dashboard, devices, topology-page, automation-page, etc.) that
 * already exist. Re-implementing it here under stable selectors
 * needs sidebar-nav-* testids on each NavItemButton, which is a
 * separate feature task.
 */

test.describe('Navigation', () => {
  test.setTimeout(60000);

  test.beforeEach(async ({ page }) => {
    await page.goto('/', { waitUntil: 'domcontentloaded' });
  });

  test('main navigation surface is visible at /', async ({ page }) => {
    const nav = page.locator('nav:visible, aside:visible, [role="navigation"]:visible').first();
    await expect(nav).toBeVisible({ timeout: 5000 });
  });

  test('browser back button restores the previous route', async ({ page }) => {
    await page.goto('/devices', { waitUntil: 'domcontentloaded' });
    await page.goto('/topology', { waitUntil: 'domcontentloaded' });
    await page.goBack();
    await expect(page).toHaveURL(/devices/, { timeout: 15000 });
  });
});
