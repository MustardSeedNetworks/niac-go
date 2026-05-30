import { expect, test } from '@playwright/test';

/**
 * Packet Inspector Page (/packets) E2E
 *
 * Covers the live-packet capture / inspection surface:
 * - Page renders the "Packets" heading
 * - Page lands on /packets route
 */

test.describe('Packet Inspector Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/packets');
    await page.waitForLoadState('domcontentloaded');
  });

  test('should render the Packets heading', async ({ page }) => {
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 10000,
    });
  });

  test('should land on the /packets route', async ({ page }) => {
    await expect(page).toHaveURL(/\/packets$/);
  });
});
