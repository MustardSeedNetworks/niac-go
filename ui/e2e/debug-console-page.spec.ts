import { expect, test } from '@playwright/test';

/**
 * Debug Console Page (/debug) E2E
 *
 * Covers the live-log debug surface:
 * - Page renders the "Debug Console" heading
 * - A reconnect / connection-status control is present
 */

test.describe('Debug Console Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/debug');
    await page.waitForLoadState('domcontentloaded');
  });

  test('should render the Debug Console heading', async ({ page }) => {
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 10000,
    });
  });

  test('should land on the /debug route', async ({ page }) => {
    await expect(page).toHaveURL(/\/debug$/);
  });

  test('should expose a reconnect / connection status control', async ({ page }) => {
    const statusButton = page.getByTestId('debug-connection-status');
    await expect(statusButton).toBeVisible();
    await statusButton.focus();
    await page.keyboard.press('Shift+Tab');
    await page.keyboard.press('Tab');
    await expect(statusButton).toBeFocused();
    await expect(statusButton).toHaveAccessibleDescription(
      /Connected to log stream|Click to reconnect/,
    );
    await expect(
      page.getByRole('tooltip').filter({ hasText: /Connected to log stream|Click to reconnect/ }),
    ).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(statusButton).toBeFocused();
  });
});
