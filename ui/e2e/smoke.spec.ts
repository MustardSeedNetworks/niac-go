import { test, expect } from '@playwright/test';

/**
 * Smoke Tests
 *
 * Basic sanity checks to verify the application loads correctly.
 * These tests should be fast and run on every PR.
 */

test.describe('Smoke Tests', () => {
  test('homepage loads successfully', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveTitle(/NIAC/i);
  });

  test('navigation is visible', async ({ page }) => {
    await page.goto('/');
    // Wait for page to fully load, then check for any visible navigation
    await page.waitForLoadState('domcontentloaded');
    const visibleNav = page.locator('nav:visible, [role="navigation"]:visible').first();
    await expect(visibleNav).toBeVisible({ timeout: 5000 });
  });

  test('no console errors on load', async ({ page }) => {
    const errors: string[] = [];
    page.on('console', (msg) => {
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });

    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Filter out expected errors (favicon, 404, 503 when no simulation running)
    const criticalErrors = errors.filter(
      (e) =>
        !e.includes('favicon') &&
        !e.includes('404') &&
        !e.includes('503') &&
        !e.includes('Service Unavailable')
    );
    expect(criticalErrors).toHaveLength(0);
  });
});
