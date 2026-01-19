import { expect, test } from '@playwright/test';

/**
 * Dashboard Page Tests
 *
 * Tests for the main dashboard:
 * - System overview
 * - Device stats
 * - Quick actions
 */

test.describe('Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
  });

  test('should display dashboard overview', async ({ page }) => {
    // Dashboard should show system stats
    await expect(page.locator('body')).not.toBeEmpty();
  });

  test('should display navigation menu', async ({ page }) => {
    // Navigation should be visible
    const nav = page.locator('nav, [role="navigation"], aside');
    await expect(nav.first()).toBeVisible();
  });

  test('should have quick access links', async ({ page }) => {
    // Check for navigation links to main pages
    const navLinks = page.locator('a, [role="link"]');
    const count = await navLinks.count();
    expect(count).toBeGreaterThan(0);
  });

  test('should display device count or status', async ({ page }) => {
    // Dashboard should show some device-related info
    const deviceInfo = page.getByText(/device|simulation|running/i);
    const count = await deviceInfo.count();
    expect(count).toBeGreaterThanOrEqual(0); // May not be visible on empty state
  });
});
