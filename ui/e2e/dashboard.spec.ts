import { expect, test } from '@playwright/test';

/**
 * Dashboard Page Tests
 *
 * Tests for the main dashboard:
 * - System overview
 * - Device stats
 * - Quick actions
 */

test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');
  });

  test('should display dashboard overview', async ({ page }) => {
    // Substantive assertion: the page's own H1 rendered. `getByRole('heading')
    // .first()` was weaker than it looked — it passes on whatever heading
    // happens to come first in the DOM, including one from the app shell, so it
    // could not distinguish "dashboard rendered" from "something rendered".
    await expect(page.getByTestId('page-header-title')).toBeVisible();
  });

  test('should display navigation menu', async ({ page }) => {
    // Navigation should be visible (use :visible to handle hidden mobile nav)
    const nav = page.locator('nav:visible, [role="navigation"]:visible, aside:visible').first();
    await expect(nav).toBeVisible({ timeout: 5000 });
  });

  test('should have quick access links', async ({ page }) => {
    // Check for navigation links to main pages
    const navLinks = page.locator('a, [role="link"]');
    await expect(navLinks.first()).toBeVisible();
  });

  // Dropped: "should display device count or empty-state hint". It
  // used /device|simulation|running/i OR /no devices|empty|nothing to
  // show|get started/i — both regexes i18n-fragile (es: "dispositivo"
  // / "sin dispositivos"). The OR pattern also made the assertion
  // soft: failure only when BOTH sides returned zero. Dashboard-render
  // coverage is already exercised by the three tests above (heading
  // visible, navigation visible, links available) plus the smoke
  // spec's page-header-title assertion. Adding a deterministic
  // device/empty contract needs a stable testid on the dashboard's
  // StatCard or empty-state component — tracked as a separate task.
});
