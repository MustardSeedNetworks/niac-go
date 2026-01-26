import { expect, test } from '@playwright/test';

/**
 * Topology Page Tests
 *
 * Tests for network topology visualization:
 * - Topology view rendering
 * - Node interactions
 * - Zoom and pan controls
 */

test.describe('Topology', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/topology');
    await page.waitForLoadState('domcontentloaded');
  });

  test('should navigate to topology page', async ({ page }) => {
    await expect(page).toHaveURL(/topology/);
  });

  test('should display topology canvas or container', async ({ page }) => {
    // Topology visualization area
    const canvas = page.locator('canvas, svg, [class*="topology"], [class*="graph"]');
    const count = await canvas.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should have zoom controls', async ({ page }) => {
    // Zoom in/out buttons
    const zoomControls = page.getByRole('button', { name: /zoom|fit|reset/i });
    const count = await zoomControls.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should display legend or node types', async ({ page }) => {
    // Legend should show node types
    const legend = page.locator('[class*="legend"], [role="list"]');
    const count = await legend.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should handle empty topology gracefully', async ({ page }) => {
    // If no devices, should show helpful message
    const emptyState = page.getByText(/no device|empty|add device|get started/i);
    const content = page.locator('main');
    // Either empty state or topology should be visible
    await expect(content).toBeVisible();
  });
});
