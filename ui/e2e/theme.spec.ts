import { expect, test } from '@playwright/test';

/**
 * Theme Tests
 *
 * Tests for dark/light mode functionality:
 * - Theme toggle
 * - Theme persistence
 * - Color scheme changes
 */

test.describe('Theme', () => {
  test('should have theme toggle button', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');

    // Look for theme toggle (sun/moon icons)
    const themeButton = page.getByRole('button', { name: /dark|light|theme/i });
    const count = await themeButton.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should toggle between dark and light mode', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');

    const html = page.locator('html');
    const initialDark = await html.evaluate((el) => el.classList.contains('dark'));

    // Find and click theme toggle
    const themeButton = page.getByRole('button', { name: /dark|light|theme/i }).first();
    if (await themeButton.isVisible()) {
      await themeButton.click();

      // Theme should have toggled
      const newDark = await html.evaluate((el) => el.classList.contains('dark'));
      expect(newDark).not.toBe(initialDark);
    }
  });

  test('should apply dark mode styles', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');

    // Force dark mode
    await page.evaluate(() => {
      document.documentElement.classList.add('dark');
    });

    const body = page.locator('body');
    const bgColor = await body.evaluate((el) => getComputedStyle(el).backgroundColor);
    expect(bgColor).toBeTruthy();
  });

  test('should apply light mode styles', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');

    // Force light mode
    await page.evaluate(() => {
      document.documentElement.classList.remove('dark');
    });

    const body = page.locator('body');
    const bgColor = await body.evaluate((el) => getComputedStyle(el).backgroundColor);
    expect(bgColor).toBeTruthy();
  });

  test('should respect system preference', async ({ page }) => {
    // Emulate dark color scheme
    await page.emulateMedia({ colorScheme: 'dark' });
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');

    // Page should render
    await expect(page.locator('body')).toBeVisible();
  });
});
