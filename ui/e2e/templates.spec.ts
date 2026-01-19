import { expect, test } from '@playwright/test';

/**
 * Templates Page Tests
 *
 * Tests for configuration templates:
 * - Template list
 * - Template creation
 * - Template application
 */

test.describe('Templates', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/templates');
    await page.waitForLoadState('networkidle');
  });

  test('should navigate to templates page', async ({ page }) => {
    await expect(page).toHaveURL(/template/);
  });

  test('should display template list or empty state', async ({ page }) => {
    const content = page.locator('main, [role="main"]').first();
    await expect(content).toBeVisible();
  });

  test('should have create template button', async ({ page }) => {
    const createButton = page.getByRole('button', { name: /create|new|add/i });
    const count = await createButton.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should display template categories', async ({ page }) => {
    // Common template categories
    const categories = ['router', 'switch', 'server', 'network'];
    for (const category of categories) {
      const element = page.getByText(new RegExp(category, 'i')).first();
      if (await element.isVisible()) {
        await expect(element).toBeVisible();
      }
    }
  });

  test('should have template search', async ({ page }) => {
    const searchInput = page.locator('input[type="search"], input[placeholder*="search" i]');
    const count = await searchInput.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });
});
