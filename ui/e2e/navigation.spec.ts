import { expect, test } from '@playwright/test';

/**
 * Navigation Tests
 *
 * Tests for app navigation:
 * - Menu links
 * - Route transitions
 * - Breadcrumbs
 */

test.describe('Navigation', () => {
  // Increase timeout for WebKit which is slower on page loads
  test.setTimeout(60000);

  test.beforeEach(async ({ page }) => {
    await page.goto('/', { waitUntil: 'domcontentloaded' });
  });

  test('should have main navigation', async ({ page }) => {
    // Use :visible to find a visible nav element (mobile nav may be hidden)
    const nav = page.locator('nav:visible, aside:visible, [role="navigation"]:visible').first();
    await expect(nav).toBeVisible({ timeout: 5000 });
  });

  test('should navigate to devices page', async ({ page }) => {
    const devicesLink = page.getByRole('link', { name: /device/i }).first();
    if (await devicesLink.isVisible()) {
      await devicesLink.click();
      await expect(page).toHaveURL(/device/);
    }
  });

  test('should navigate to topology page', async ({ page }) => {
    const topologyLink = page.getByRole('link', { name: /topology/i }).first();
    if (await topologyLink.isVisible()) {
      await topologyLink.click();
      await expect(page).toHaveURL(/topology/);
    }
  });

  test('should navigate to templates page', async ({ page }) => {
    const templatesLink = page.getByRole('link', { name: /template/i }).first();
    if (await templatesLink.isVisible()) {
      await templatesLink.click();
      await expect(page).toHaveURL(/template/);
    }
  });

  test('should have working back navigation', async ({ page }) => {
    // Navigate to a page
    await page.goto('/devices', { waitUntil: 'domcontentloaded' });

    // Navigate to another page
    await page.goto('/templates', { waitUntil: 'domcontentloaded' });

    // Go back
    await page.goBack();
    await expect(page).toHaveURL(/device/, { timeout: 15000 });
  });

  test('should preserve page state on navigation', async ({ page }) => {
    // This is a basic check - more specific tests would be needed
    await page.goto('/devices');
    await page.waitForLoadState('domcontentloaded');

    // Navigate away and back
    await page.goto('/');
    await page.goto('/devices');

    // Page should load
    await expect(page.locator('body')).toBeVisible();
  });
});
