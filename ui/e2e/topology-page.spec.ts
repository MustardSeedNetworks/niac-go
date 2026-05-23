import { expect, test } from '@playwright/test';

/**
 * Topology Page (/topology) E2E
 *
 * Covers the network topology visualization surface:
 * - Page renders the "Network Topology" heading
 * - Page lands on /topology route
 */

test.describe('Topology Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/topology');
    await page.waitForLoadState('domcontentloaded');
  });

  test('should render the Network Topology heading', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /network topology/i })).toBeVisible({
      timeout: 10000,
    });
  });

  test('should land on the /topology route', async ({ page }) => {
    await expect(page).toHaveURL(/\/topology$/);
  });
});
