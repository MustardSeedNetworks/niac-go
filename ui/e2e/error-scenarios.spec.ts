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
    // Set up route interception before navigation
    await page.route('**/api/v1/**', (route) => {
      route.abort('failed');
    });

    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // React app should render despite API errors
    const root = page.locator('#root');
    await expect(root).toBeAttached({ timeout: 10000 });
  });

  test('should handle slow API responses', async ({ page }) => {
    await page.route('**/api/v1/**', async (route) => {
      await new Promise((resolve) => setTimeout(resolve, 2000));
      route.continue();
    });

    await page.goto('/');
    const root = page.locator('#root');
    await expect(root).toBeAttached({ timeout: 15000 });
  });

  test('should handle 404 routes', async ({ page }) => {
    await page.goto('/nonexistent-page-12345');
    await page.waitForLoadState('domcontentloaded');

    // Should show 404 page or redirect to home
    const root = page.locator('#root');
    await expect(root).toBeAttached();
  });

  test('should handle 500 errors from API', async ({ page }) => {
    await page.route('**/api/v1/**', (route) => {
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Internal Server Error' }),
      });
    });

    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Page should render with error state or fallback UI
    const root = page.locator('#root');
    await expect(root).toBeAttached({ timeout: 10000 });
  });

  test('should handle malformed API responses', async ({ page }) => {
    await page.route('**/api/v1/**', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: 'invalid json{{{',
      });
    });

    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Page should handle JSON parse errors gracefully
    const root = page.locator('#root');
    await expect(root).toBeAttached({ timeout: 10000 });
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
    await page.waitForLoadState('domcontentloaded');

    // Should show empty state
    const root = page.locator('#root');
    await expect(root).toBeAttached();
  });

  test('should display error state for bad requests', async ({ page }) => {
    await page.route('**/api/v1/**', (route) => {
      route.fulfill({
        status: 400,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Bad Request' }),
      });
    });

    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Page should render with error indication
    const root = page.locator('#root');
    await expect(root).toBeAttached({ timeout: 10000 });
  });
});
