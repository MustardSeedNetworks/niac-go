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
    await expect(page.getByRole('heading', { name: /^debug console$/i })).toBeVisible({
      timeout: 10000,
    });
  });

  test('should land on the /debug route', async ({ page }) => {
    await expect(page).toHaveURL(/\/debug$/);
  });

  test('should expose a reconnect / connection status control', async ({ page }) => {
    // The header has a button whose title reflects either
    // "Connected to log stream" or "Click to reconnect" — both reachable
    // via the title attribute.
    const statusButton = page.locator('button[title*="log stream"], button[title*="reconnect"]');
    await expect(statusButton.first()).toBeVisible({ timeout: 5000 });
  });
});
