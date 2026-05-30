import { expect, test } from '@playwright/test';

/**
 * Automation Page (/automation) E2E
 *
 * Covers the alert / automation policy surface:
 * - Page renders with "Alert policy" heading
 * - Page lands on /automation route
 */

test.describe('Automation Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/automation');
    await page.waitForLoadState('domcontentloaded');
  });

  test('should render the Alert policy heading', async ({ page }) => {
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 10000,
    });
  });

  test('should land on the /automation route', async ({ page }) => {
    await expect(page).toHaveURL(/\/automation$/);
  });

  test('should render alert configuration form fields', async ({ page }) => {
    // The page exposes threshold + webhook fields; at least one form input
    // should be present.
    const input = page.locator('input[type="number"], input[type="url"], input[type="text"]');
    await expect(input.first()).toBeVisible({ timeout: 5000 });
  });
});
