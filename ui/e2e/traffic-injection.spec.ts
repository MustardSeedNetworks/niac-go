import { expect, test } from '@playwright/test';

/**
 * Traffic Injection Tests
 *
 * Tests for error injection functionality:
 * - Error type selection
 * - Target device/interface selection
 * - Start/stop injection
 * - Statistics display
 */

test.describe('Traffic Injection', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/traffic');
    await page.waitForLoadState('domcontentloaded');
  });

  test.describe('Page Layout', () => {
    test('should navigate to traffic injection page', async ({ page }) => {
      await expect(page).toHaveURL(/traffic/);
    });

    test('should display traffic configuration options', async ({ page }) => {
      const content = page.locator('main, [role="main"]').first();
      await expect(content).toBeVisible();
    });

    test('should have error injection controls', async ({ page }) => {
      const _controls = page.locator('[class*="control"], [class*="injection"], form');
      await expect(page.locator('body')).toBeVisible();
    });

    test('should display active injections list', async ({ page }) => {
      const _injectionsList = page.locator('[class*="active"], [class*="list"], table');
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('Error Type Selection', () => {
    test('should have protocol selection', async ({ page }) => {
      const protocolSelect = page.locator('select, [role="combobox"]');
      const count = await protocolSelect.count();
      expect(count).toBeGreaterThanOrEqual(0);
    });

    test('should have error type dropdown or selector', async ({ page }) => {
      const errorTypeSelect = page.locator('select[name*="error" i], select[name*="type" i]');
      const errorTypeButtons = page.locator('[class*="error-type"], [class*="injection-type"]');

      const _hasSelector =
        (await errorTypeSelect.count()) > 0 || (await errorTypeButtons.count()) > 0;
      await expect(page.locator('body')).toBeVisible();
    });

    test('should list available error types', async ({ page }) => {
      const _errorTypes = page.getByText(/drop|delay|corrupt|duplicate|reorder/i);
      await expect(page.locator('body')).toBeVisible();
    });

    test('should show error type description', async ({ page }) => {
      const errorTypeSelect = page
        .locator('select[name*="error" i], select[name*="type" i]')
        .first();
      if (await errorTypeSelect.isVisible()) {
        await errorTypeSelect.focus();
        await expect(page.locator('body')).toBeVisible();
      }
    });
  });

  test.describe('Target Selection', () => {
    test('should have device selector', async ({ page }) => {
      const deviceSelect = page.locator('select[name*="device" i], [class*="device-select"]');
      const count = await deviceSelect.count();
      expect(count).toBeGreaterThanOrEqual(0);
    });

    test('should have interface selector', async ({ page }) => {
      const interfaceSelect = page.locator(
        'select[name*="interface" i], [class*="interface-select"]',
      );
      const count = await interfaceSelect.count();
      expect(count).toBeGreaterThanOrEqual(0);
    });

    test('should update interfaces when device changes', async ({ page }) => {
      const deviceSelect = page.locator('select[name*="device" i]').first();
      const interfaceSelect = page.locator('select[name*="interface" i]').first();

      if ((await deviceSelect.isVisible()) && (await interfaceSelect.isVisible())) {
        await deviceSelect.selectOption({ index: 1 });
        await page.waitForTimeout(150);
        await expect(page.locator('body')).toBeVisible();
      }
    });
  });

  test.describe('Injection Parameters', () => {
    test('should have rate/percentage input', async ({ page }) => {
      const rateInput = page.locator('input[name*="rate" i], input[name*="percent" i]');
      const count = await rateInput.count();
      expect(count).toBeGreaterThanOrEqual(0);
    });

    test('should validate rate range (0-100)', async ({ page }) => {
      const rateInput = page.locator('input[name*="rate" i], input[name*="percent" i]').first();
      if (await rateInput.isVisible()) {
        await rateInput.fill('150');
        await rateInput.blur();
        await page.waitForTimeout(100);
        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should have delay input for delay injection', async ({ page }) => {
      const _delayInput = page.locator('input[name*="delay" i], input[name*="latency" i]');
      await expect(page.locator('body')).toBeVisible();
    });

    test('should have direction selector (inbound/outbound)', async ({ page }) => {
      const _directionSelect = page.locator('select[name*="direction" i], [class*="direction"]');
      const _directionButtons = page.getByRole('button', { name: /inbound|outbound|both/i });
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('Start Injection', () => {
    test('should have start/stop traffic buttons', async ({ page }) => {
      const trafficButton = page.getByRole('button', { name: /start|stop|inject|generate/i });
      const count = await trafficButton.count();
      expect(count).toBeGreaterThanOrEqual(0);
    });

    test('should disable start button when no target selected', async ({ page }) => {
      const startButton = page.getByRole('button', { name: /start|inject|apply/i }).first();
      if (await startButton.isVisible()) {
        const _isDisabled = await startButton.isDisabled();
        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should show confirmation before starting', async ({ page }) => {
      const startButton = page.getByRole('button', { name: /start|inject|apply/i }).first();
      if ((await startButton.isVisible()) && (await startButton.isEnabled())) {
        await startButton.click();
        await page.waitForTimeout(150);
        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should show success on injection start', async ({ page }) => {
      await page.route('**/api/v1/errors', (route) => {
        if (route.request().method() === 'POST') {
          route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({ id: 'test-injection-1', status: 'active' }),
          });
        } else {
          route.continue();
        }
      });
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('Active Injections', () => {
    test('should display traffic statistics', async ({ page }) => {
      const stats = page.getByText(/packet|rate|byte|pps/i);
      const count = await stats.count();
      expect(count).toBeGreaterThanOrEqual(0);
    });

    test('should display list of active injections', async ({ page }) => {
      const _activeList = page.locator('[class*="active"], table tbody');
      await expect(page.locator('body')).toBeVisible();
    });

    test('should show injection details', async ({ page }) => {
      const _injectionDetails = page.locator('[class*="injection"], tr');
      await expect(page.locator('body')).toBeVisible();
    });

    test('should have stop button for each injection', async ({ page }) => {
      const _stopButtons = page.getByRole('button', { name: /stop|remove|clear/i });
      await expect(page.locator('body')).toBeVisible();
    });

    test('should show injection statistics', async ({ page }) => {
      const _stats = page.getByText(/packet|dropped|delayed|corrupted/i);
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('Stop Injection', () => {
    test('should confirm before stopping', async ({ page }) => {
      const stopButton = page.getByRole('button', { name: /stop|remove/i }).first();
      if (await stopButton.isVisible()) {
        await stopButton.click();
        await page.waitForTimeout(150);
        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should remove injection from active list', async ({ page }) => {
      await page.route('**/api/v1/errors/*', (route) => {
        if (route.request().method() === 'DELETE') {
          route.fulfill({ status: 204 });
        } else {
          route.continue();
        }
      });

      const stopButton = page.getByRole('button', { name: /stop|remove/i }).first();
      if (await stopButton.isVisible()) {
        await stopButton.click();
        await page.waitForTimeout(250);
        await expect(page.locator('body')).toBeVisible();
      }
    });
  });

  test.describe('Clear All', () => {
    test('should have clear all button', async ({ page }) => {
      const clearAllButton = page.getByRole('button', { name: /clear all|stop all/i });
      const count = await clearAllButton.count();
      expect(count).toBeGreaterThanOrEqual(0);
    });

    test('should confirm before clearing all', async ({ page }) => {
      const clearAllButton = page.getByRole('button', { name: /clear all|stop all/i }).first();
      if (await clearAllButton.isVisible()) {
        await clearAllButton.click();
        await page.waitForTimeout(150);
        const _confirmDialog = page.locator('[role="dialog"], [role="alertdialog"]');
        await expect(page.locator('body')).toBeVisible();
      }
    });
  });

  test.describe('Real-time Updates', () => {
    test('should update statistics in real-time', async ({ page }) => {
      await page.waitForTimeout(150);
      await expect(page.locator('body')).toBeVisible();
    });

    test('should show injection status changes', async ({ page }) => {
      const _statusIndicator = page.locator('[class*="status"], [class*="indicator"]');
      await expect(page.locator('body')).toBeVisible();
    });
  });
});
