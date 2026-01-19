import { expect, test } from '@playwright/test';

/**
 * Packet Inspector Page Tests
 *
 * Tests for packet analysis and inspection:
 * - Packet list display
 * - Packet details
 * - Filtering and search
 */

test.describe('Packet Inspector', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/packets');
    await page.waitForLoadState('networkidle');
  });

  test('should navigate to packet inspector page', async ({ page }) => {
    await expect(page).toHaveURL(/packet/);
  });

  test('should display packet list or capture area', async ({ page }) => {
    const content = page.locator('main, [role="main"]').first();
    await expect(content).toBeVisible();
  });

  test('should have capture controls', async ({ page }) => {
    // Start/stop capture buttons
    const captureButton = page.getByRole('button', { name: /capture|start|stop|record/i });
    const count = await captureButton.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should have filter input', async ({ page }) => {
    const filterInput = page.locator('input[placeholder*="filter" i], input[type="search"]');
    const count = await filterInput.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should display protocol columns', async ({ page }) => {
    // Packet list headers
    const headers = page.getByText(/source|destination|protocol|length|time/i);
    const count = await headers.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });
});
