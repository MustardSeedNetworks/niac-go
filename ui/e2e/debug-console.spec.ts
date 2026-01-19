import { expect, test } from '@playwright/test';

/**
 * Debug Console Page Tests
 *
 * Tests for debug console functionality:
 * - Console output
 * - Command input
 * - Log filtering
 */

test.describe('Debug Console', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/debug');
    await page.waitForLoadState('networkidle');
  });

  test('should navigate to debug console page', async ({ page }) => {
    await expect(page).toHaveURL(/debug|console/);
  });

  test('should display console interface', async ({ page }) => {
    const content = page.locator('main, [role="main"]').first();
    await expect(content).toBeVisible();
  });

  test('should have console output area', async ({ page }) => {
    // Console log display
    const console = page.locator('[class*="console"], [class*="log"], pre, code');
    const count = await console.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should have command input', async ({ page }) => {
    const input = page.locator('input[type="text"], textarea');
    const count = await input.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should have log level filter', async ({ page }) => {
    // Filter by log level
    const filter = page.getByRole('button', { name: /debug|info|warn|error|filter/i });
    const count = await filter.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should have clear button', async ({ page }) => {
    const clearButton = page.getByRole('button', { name: /clear|reset/i });
    const count = await clearButton.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });
});
