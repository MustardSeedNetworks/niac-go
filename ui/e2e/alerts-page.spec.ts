import { expect, test } from '@playwright/test';

/**
 * Alerts Page (/alerts) E2E
 *
 * Covers the alert policy surface:
 * - Page renders with "Alert policy" heading
 * - Page lands on /alerts route
 */

test.describe('Alerts Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/alerts');
    await page.waitForLoadState('domcontentloaded');
  });

  test('should render the Alert policy heading', async ({ page }) => {
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 10000,
    });
  });

  test('should land on the /alerts route', async ({ page }) => {
    await expect(page).toHaveURL(/\/alerts$/);
  });

  test('should render alert configuration form fields', async ({ page }) => {
    // The page exposes threshold + webhook fields; at least one form input
    // should be present.
    const input = page.locator('input[type="number"], input[type="url"], input[type="text"]');
    await expect(input.first()).toBeVisible({ timeout: 5000 });
  });
});
