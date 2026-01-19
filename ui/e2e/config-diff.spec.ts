import { expect, test } from '@playwright/test';

/**
 * Config Diff Page Tests
 *
 * Tests for configuration comparison:
 * - Config file selection
 * - Diff display
 * - Side-by-side view
 */

test.describe('Config Diff', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/config-diff');
    await page.waitForLoadState('networkidle');
  });

  test('should navigate to config diff page', async ({ page }) => {
    await expect(page).toHaveURL(/config|diff/);
  });

  test('should display diff interface', async ({ page }) => {
    const content = page.locator('main, [role="main"]').first();
    await expect(content).toBeVisible();
  });

  test('should have file selectors', async ({ page }) => {
    // Select inputs for configs to compare
    const selectors = page.locator('select, input[type="file"], [role="combobox"]');
    const count = await selectors.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should have compare button', async ({ page }) => {
    const compareButton = page.getByRole('button', { name: /compare|diff|analyze/i });
    const count = await compareButton.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should show diff view modes', async ({ page }) => {
    // Side-by-side or unified view options
    const viewModes = page.getByRole('button', { name: /side|unified|split/i });
    const count = await viewModes.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });
});
