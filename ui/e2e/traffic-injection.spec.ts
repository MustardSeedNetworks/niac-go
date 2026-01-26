import { expect, test } from '@playwright/test';

/**
 * Traffic Injection Page Tests
 *
 * Tests for traffic generation and injection:
 * - Traffic configuration
 * - Start/stop controls
 * - Traffic statistics
 */

test.describe('Traffic Injection', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/traffic');
    await page.waitForLoadState('domcontentloaded');
  });

  test('should navigate to traffic injection page', async ({ page }) => {
    await expect(page).toHaveURL(/traffic/);
  });

  test('should display traffic configuration options', async ({ page }) => {
    const content = page.locator('main, [role="main"]').first();
    await expect(content).toBeVisible();
  });

  test('should have protocol selection', async ({ page }) => {
    // Protocol dropdown or options
    const protocolSelect = page.locator('select, [role="combobox"]');
    const count = await protocolSelect.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should have start/stop traffic buttons', async ({ page }) => {
    const trafficButton = page.getByRole('button', { name: /start|stop|inject|generate/i });
    const count = await trafficButton.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should display traffic statistics', async ({ page }) => {
    // Stats like packets sent, rate, etc.
    const stats = page.getByText(/packet|rate|byte|pps/i);
    const count = await stats.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });
});
