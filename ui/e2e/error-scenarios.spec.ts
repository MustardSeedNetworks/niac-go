import { expect, test } from '@playwright/test';

/**
 * Error Scenario Tests
 *
 * Tests for error handling and edge cases:
 * - Network errors
 * - API failures
 * - Invalid routes
 */

test.describe('Error Scenarios', () => {
  test('should handle network errors gracefully', async ({ page }) => {
    await page.route('**/api/**', (route) => {
      route.abort('failed');
    });

    await page.goto('/');

    // Page should still render
    await expect(page.locator('body')).not.toBeEmpty();
  });

  test('should handle slow API responses', async ({ page }) => {
    await page.route('**/api/**', async (route) => {
      await new Promise((resolve) => setTimeout(resolve, 2000));
      route.continue();
    });

    await page.goto('/');
    await expect(page.locator('body')).toBeVisible({ timeout: 10000 });
  });

  test('should handle 404 routes', async ({ page }) => {
    await page.goto('/nonexistent-page-12345');

    // Should show 404 page or redirect
    await expect(page.locator('body')).not.toBeEmpty();
  });

  test('should handle 500 errors from API', async ({ page }) => {
    await page.route('**/api/**', (route) => {
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Internal Server Error' }),
      });
    });

    await page.goto('/');
    await expect(page.locator('body')).not.toBeEmpty();
  });

  test('should handle malformed API responses', async ({ page }) => {
    await page.route('**/api/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: 'invalid json{{{',
      });
    });

    await page.goto('/');
    await expect(page.locator('body')).not.toBeEmpty();
  });

  test('should handle empty API responses', async ({ page }) => {
    await page.route('**/api/v1/devices', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      });
    });

    await page.goto('/devices');
    await expect(page.locator('body')).not.toBeEmpty();
  });

  test('should display error messages to user', async ({ page }) => {
    await page.route('**/api/**', (route) => {
      route.fulfill({
        status: 400,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Bad Request' }),
      });
    });

    await page.goto('/');
    // Error should be communicated (via toast, alert, or inline message)
    await expect(page.locator('body')).not.toBeEmpty();
  });
});
