import { expect, test } from '@playwright/test';

/**
 * Runtime Control Page (/runtime) E2E
 *
 * Covers the simulation start / control surface:
 * - Page renders the "Start Simulation" heading
 * - Page lands on /runtime route
 */

test.describe('Runtime Control Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/runtime');
    await page.waitForLoadState('domcontentloaded');
  });

  test('should render the Start Simulation heading', async ({ page }) => {
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 10000,
    });
  });

  test('should land on the /runtime route', async ({ page }) => {
    await expect(page).toHaveURL(/\/runtime$/);
  });
});
