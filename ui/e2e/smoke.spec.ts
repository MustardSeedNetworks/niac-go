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
    // Adjust selector based on actual navigation structure
    const nav = page.locator('nav, [role="navigation"]');
    await expect(nav).toBeVisible();
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

    // Filter out expected errors (e.g., missing favicon)
    const criticalErrors = errors.filter(
      (e) => !e.includes('favicon') && !e.includes('404')
    );
    expect(criticalErrors).toHaveLength(0);
  });
});
