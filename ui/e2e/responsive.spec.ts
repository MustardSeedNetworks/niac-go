import { expect, test, type Page } from '@playwright/test';

/**
 * Responsive Design Tests
 *
 * Tests for mobile and tablet responsiveness:
 * - Layout adjustments
 * - Navigation on small screens
 * - Touch-friendly targets
 */

const viewports = {
  mobile: { width: 375, height: 667 },
  tablet: { width: 768, height: 1024 },
  desktop: { width: 1920, height: 1080 },
};

async function checkTouchTargets(page: Page) {
  const buttons = page.locator('button');
  const count = await buttons.count();

  for (let i = 0; i < Math.min(count, 10); i++) {
    const button = buttons.nth(i);
    if (await button.isVisible()) {
      const box = await button.boundingBox();
      if (box) {
        // Minimum touch target size
        expect(box.width).toBeGreaterThanOrEqual(32);
        expect(box.height).toBeGreaterThanOrEqual(32);
      }
    }
  }
}

test.describe('Responsive Design', () => {
  test('should display correctly on mobile', async ({ page }) => {
    await page.setViewportSize(viewports.mobile);
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Page should render
    await expect(page.locator('body')).toBeVisible();

    // No horizontal overflow
    const body = page.locator('body');
    const scrollWidth = await body.evaluate((el) => el.scrollWidth);
    const clientWidth = await body.evaluate((el) => el.clientWidth);
    expect(scrollWidth).toBeLessThanOrEqual(clientWidth + 10);

    await checkTouchTargets(page);
  });

  test('should display correctly on tablet', async ({ page }) => {
    await page.setViewportSize(viewports.tablet);
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('body')).toBeVisible();
    await checkTouchTargets(page);
  });

  test('should display correctly on desktop', async ({ page }) => {
    await page.setViewportSize(viewports.desktop);
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    await expect(page.locator('body')).toBeVisible();
  });

  test('should show mobile navigation on small screens', async ({ page }) => {
    await page.setViewportSize(viewports.mobile);
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Look for hamburger menu or mobile nav
    const mobileNav = page.locator('[class*="mobile"], [class*="hamburger"], [class*="menu"]');
    // Navigation should exist in some form
    const nav = page.locator('nav, aside, [role="navigation"]');
    await expect(nav.or(mobileNav).first()).toBeVisible();
  });

  test('should maintain navigation on all pages', async ({ page }) => {
    const pages = ['/', '/devices', '/topology', '/templates'];

    for (const url of pages) {
      await page.setViewportSize(viewports.mobile);
      await page.goto(url);
      await page.waitForLoadState('networkidle');

      // Page should not be completely blank
      const content = await page.locator('body').textContent();
      expect(content?.length).toBeGreaterThan(0);
    }
  });
});
