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
    await page.waitForLoadState('domcontentloaded');
  });

  test('should display dashboard overview', async ({ page }) => {
    // Substantive assertion: the page renders at least one heading.
    // The previous `expect(body).not.toBeEmpty()` was tautological — body
    // is never empty for a rendered SPA shell.
    await expect(page.getByRole('heading').first()).toBeVisible();
  });

  test('should display navigation menu', async ({ page }) => {
    // Navigation should be visible (use :visible to handle hidden mobile nav)
    const nav = page.locator('nav:visible, [role="navigation"]:visible, aside:visible').first();
    await expect(nav).toBeVisible({ timeout: 5000 });
  });

  test('should have quick access links', async ({ page }) => {
    // Check for navigation links to main pages
    const navLinks = page.locator('a, [role="link"]');
    const count = await navLinks.count();
    expect(count).toBeGreaterThan(0);
  });

  test('should display device count or empty-state hint', async ({ page }) => {
    // Either devices are listed OR the empty-state copy is shown. The old
    // `expect(count) >= 0` form was tautological — counts can never be
    // negative, so the assertion always passed even on a completely blank
    // dashboard.
    const populated = page.getByText(/device|simulation|running/i);
    const empty = page.getByText(/no devices|empty|nothing to show|get started/i);
    const [populatedCount, emptyCount] = await Promise.all([populated.count(), empty.count()]);
    expect(
      populatedCount + emptyCount,
      'dashboard must show either device info or an empty-state hint',
    ).toBeGreaterThan(0);
  });
});
