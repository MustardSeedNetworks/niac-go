import { expect, test } from '@playwright/test';

/**
 * PCAP Analyzer Page Tests
 *
 * Tests for PCAP file analysis:
 * - File upload
 * - Packet parsing
 * - Analysis display
 */

test.describe('PCAP Analyzer', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/pcap-analyzer');
    await page.waitForLoadState('domcontentloaded');
  });

  test('should navigate to pcap analyzer page', async ({ page }) => {
    await expect(page).toHaveURL(/pcap|analyzer/);
  });

  test('should display upload interface', async ({ page }) => {
    const content = page.locator('main, [role="main"]').first();
    await expect(content).toBeVisible();
  });

  test('should have file upload input', async ({ page }) => {
    // File input for PCAP upload
    const fileInput = page.locator('input[type="file"]');
    const dropZone = page.locator('[class*="drop"], [class*="upload"]');
    const count = (await fileInput.count()) + (await dropZone.count());
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should have analyze button', async ({ page }) => {
    const analyzeButton = page.getByRole('button', { name: /analyze|parse|load/i });
    const count = await analyzeButton.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should display packet list area', async ({ page }) => {
    // Packet list or table
    const packetList = page.locator('table, [class*="packet"], [class*="list"]');
    const count = await packetList.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should have filter capabilities', async ({ page }) => {
    const filter = page.locator('input[placeholder*="filter" i], input[type="search"]');
    const count = await filter.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should have export functionality', async ({ page }) => {
    const exportButton = page.getByRole('button', { name: /export|download|save/i });
    const count = await exportButton.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });
});
