import { expect, test } from '@playwright/test';

/**
 * Visual Regression Tests
 *
 * Captures screenshots of key pages for visual comparison.
 * Run: npm run test:visual
 * Update baselines: npm run test:visual:update
 */

test.describe('Visual Regression - Pages', () => {
  test.beforeEach(async ({ page }) => {
    // Wait for fonts and styles to load
    await page.waitForLoadState('domcontentloaded');
  });

  test('dashboard page', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(1000); // Allow animations to settle
    // Mask dynamic elements that change with SSE
    await expect(page).toHaveScreenshot('dashboard.png', {
      fullPage: true,
      mask: [page.locator('[class*="live"], [class*="counter"], [class*="timestamp"]')],
    });
  });

  test('devices page', async ({ page }) => {
    await page.goto('/devices');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(1000);
    await expect(page).toHaveScreenshot('devices.png', {
      fullPage: true,
      maxDiffPixelRatio: 0.05, // Allow 5% difference for dynamic content
    });
  });

  test('runtime page', async ({ page }) => {
    await page.goto('/runtime');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(1000);
    // Mask dynamic SSE content
    await expect(page).toHaveScreenshot('runtime.png', {
      fullPage: true,
      mask: [
        page.locator('[class*="live"], [class*="counter"], [class*="timestamp"], [class*="stat"]'),
      ],
    });
  });

  test('packets page', async ({ page }) => {
    await page.goto('/packets');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(1000);
    // Mask dynamic packet stream content
    await expect(page).toHaveScreenshot('packets.png', {
      fullPage: true,
      mask: [
        page.locator(
          '[class*="packet"], [class*="stream"], [class*="counter"], [class*="timestamp"]',
        ),
      ],
    });
  });

  test('pcap analyzer page', async ({ page }) => {
    await page.goto('/packets');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot('pcap-analyzer.png', {
      fullPage: true,
    });
  });

  test('debug page', async ({ page }) => {
    await page.goto('/debug');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(1000);
    await expect(page).toHaveScreenshot('debug.png', {
      fullPage: true,
      mask: [page.locator('[class*="log"], [class*="console"], [class*="timestamp"]')],
    });
  });

  test('device config page', async ({ page }) => {
    await page.goto('/device-config/new');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(500);
    await expect(page).toHaveScreenshot('device-config.png', {
      fullPage: true,
    });
  });
});

test.describe('Visual Regression - Components', () => {
  test('navigation sidebar', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(500);

    const nav = page.locator('nav:visible, [role="navigation"]:visible').first();
    if (await nav.isVisible()) {
      await expect(nav).toHaveScreenshot('navigation.png');
    }
  });

  test('device list table', async ({ page }) => {
    await page.goto('/devices');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(500);

    const table = page.locator('table, [role="table"]').first();
    if (await table.isVisible()) {
      await expect(table).toHaveScreenshot('device-table.png');
    }
  });

  test('packet stream area', async ({ page }) => {
    await page.goto('/packets');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(500);

    const streamArea = page.locator('main, [role="main"]').first();
    if (await streamArea.isVisible()) {
      await expect(streamArea).toHaveScreenshot('packet-stream.png');
    }
  });
});

test.describe('Visual Regression - Empty States', () => {
  test('empty device list', async ({ page }) => {
    await page.goto('/devices');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(500);

    const emptyState = page.locator('[class*="empty"], [class*="no-data"]').first();
    if (await emptyState.isVisible()) {
      await expect(emptyState).toHaveScreenshot('empty-devices.png');
    }
  });

  test('empty packets view', async ({ page }) => {
    await page.goto('/packets');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(500);

    const emptyState = page.locator('[class*="empty"], [class*="waiting"]').first();
    if (await emptyState.isVisible()) {
      await expect(emptyState).toHaveScreenshot('empty-packets.png');
    }
  });
});

test.describe('Visual Regression - Forms', () => {
  test('device form fields', async ({ page }) => {
    await page.goto('/device-config/new');
    await page.waitForLoadState('domcontentloaded');
    await page.waitForTimeout(500);

    const form = page.locator('form').first();
    if (await form.isVisible()) {
      await expect(form).toHaveScreenshot('device-form.png');
    }
  });
});
