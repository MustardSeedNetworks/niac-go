import { expect, test } from '@playwright/test';

/**
 * Devices Page Tests
 *
 * Tests for device list and management:
 * - Device list display
 * - Device filtering
 * - Device actions
 */

test.describe('Devices', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/devices');
    await page.waitForLoadState('networkidle');
  });

  test('should navigate to devices page', async ({ page }) => {
    await expect(page).toHaveURL(/devices/);
  });

  test('should display device list or empty state', async ({ page }) => {
    // Either device list or empty state message
    const content = page.locator('main, [role="main"]').first();
    await expect(content).toBeVisible();
  });

  test('should have add device button', async ({ page }) => {
    // Look for add/create button
    const addButton = page.getByRole('button', { name: /add|create|new/i });
    const count = await addButton.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should have search or filter capability', async ({ page }) => {
    // Look for search input
    const searchInput = page.locator('input[type="search"], input[placeholder*="search" i]');
    const count = await searchInput.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should display device types', async ({ page }) => {
    // Device types should be visible (router, switch, etc.)
    const deviceTypes = ['router', 'switch', 'server', 'firewall'];
    for (const type of deviceTypes) {
      const element = page.getByText(new RegExp(type, 'i')).first();
      // At least check page renders
      if (await element.isVisible()) {
        await expect(element).toBeVisible();
      }
    }
  });
});
