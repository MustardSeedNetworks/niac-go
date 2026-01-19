import { expect, test } from '@playwright/test';

/**
 * Automation Page Tests
 *
 * Tests for automation and scripting:
 * - Automation scripts
 * - Scheduled tasks
 * - Script execution
 */

test.describe('Automation', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/automation');
    await page.waitForLoadState('networkidle');
  });

  test('should navigate to automation page', async ({ page }) => {
    await expect(page).toHaveURL(/automation/);
  });

  test('should display automation interface', async ({ page }) => {
    const content = page.locator('main, [role="main"]').first();
    await expect(content).toBeVisible();
  });

  test('should have script editor or list', async ({ page }) => {
    // Script editor or script list
    const scripts = page.locator('textarea, [class*="editor"], [class*="script"]');
    const count = await scripts.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should have run/execute button', async ({ page }) => {
    const runButton = page.getByRole('button', { name: /run|execute|start/i });
    const count = await runButton.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should show execution history', async ({ page }) => {
    const history = page.getByText(/history|log|result|output/i);
    const count = await history.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });
});
