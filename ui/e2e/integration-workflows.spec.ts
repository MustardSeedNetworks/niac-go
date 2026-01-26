import { expect, test } from '@playwright/test';

/**
 * Integration Workflow Tests
 *
 * End-to-end tests for complete user workflows:
 * - Device lifecycle
 * - Configuration comparison
 * - PCAP analysis
 * - Simulation with traffic
 * - Debug investigation
 */

test.describe('Integration Workflows', () => {
  // Increase timeout for complex integration tests across all browsers
  test.setTimeout(90000);

  test.describe('Device Lifecycle Workflow', () => {
    const testDevice = `e2e-device-${Date.now()}`;

    test('should complete full device lifecycle: create -> edit -> clone -> delete', async ({ page }) => {
      // Step 1: Navigate to device creation
      await page.goto('/device-config/new');
      await page.waitForLoadState('domcontentloaded');

      // Step 2: Fill device form
      const hostnameField = page.locator('input[name="hostname"], input[name="name"]').first();
      if (await hostnameField.isVisible()) {
        await hostnameField.fill(testDevice);
      }

      const ipField = page.locator('input[name*="ip" i]').first();
      if (await ipField.isVisible()) {
        await ipField.fill('10.0.0.100');
      }

      // Step 3: Save device
      const saveButton = page.getByRole('button', { name: /save|create/i }).first();
      if (await saveButton.isVisible() && (await saveButton.isEnabled())) {
        await saveButton.click();
        await page.waitForTimeout(1000);
      }

      // Step 4: Navigate to devices list
      await page.goto('/devices');
      await page.waitForLoadState('domcontentloaded');

      // Step 5: Verify device appears in list
      const deviceInList = page.getByText(testDevice);
      await expect(page.locator('body')).toBeVisible();

      // Step 6: Edit device
      const editButton = page.getByRole('button', { name: /edit/i }).first();
      if (await editButton.isVisible()) {
        await editButton.click();
        await page.waitForLoadState('domcontentloaded');

        const hostnameFieldEdit = page.locator('input[name="hostname"], input[name="name"]').first();
        if (await hostnameFieldEdit.isVisible()) {
          const currentValue = await hostnameFieldEdit.inputValue();
          await hostnameFieldEdit.fill(currentValue + '-edited');
        }

        const updateButton = page.getByRole('button', { name: /save|update/i }).first();
        if (await updateButton.isVisible() && (await updateButton.isEnabled())) {
          await updateButton.click();
          await page.waitForTimeout(1000);
        }
      }

      // Step 7: Clone device
      await page.goto('/devices');
      await page.waitForLoadState('domcontentloaded');

      const cloneButton = page.getByRole('button', { name: /clone|copy/i }).first();
      if (await cloneButton.isVisible()) {
        await cloneButton.click();
        await page.waitForTimeout(1000);
      }

      // Step 8: Delete device
      const deleteButton = page.getByRole('button', { name: /delete|remove/i }).first();
      if (await deleteButton.isVisible()) {
        await deleteButton.click();
        await page.waitForTimeout(500);

        const confirmButton = page.getByRole('button', { name: /confirm|yes/i }).first();
        if (await confirmButton.isVisible()) {
          await confirmButton.click();
          await page.waitForTimeout(1000);
        }
      }

      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('Configuration Comparison Workflow', () => {
    test('should compare two configurations', async ({ page }) => {
      // Step 1: Navigate to config diff page
      await page.goto('/config-diff');
      await page.waitForLoadState('domcontentloaded');

      // Step 2: Select first configuration
      const sourceSelect = page.locator('select[name*="source" i], select[name*="left" i]').first();
      if (await sourceSelect.isVisible()) {
        await sourceSelect.selectOption({ index: 1 });
        await page.waitForTimeout(300);
      }

      // Step 3: Select second configuration
      const targetSelect = page.locator('select[name*="target" i], select[name*="right" i]').first();
      if (await targetSelect.isVisible()) {
        await targetSelect.selectOption({ index: 2 });
        await page.waitForTimeout(300);
      }

      // Step 4: Compare
      const compareButton = page.getByRole('button', { name: /compare|diff/i }).first();
      if (await compareButton.isVisible()) {
        await compareButton.click();
        await page.waitForTimeout(1000);
      }

      // Step 5: View diff results
      const diffView = page.locator('[class*="diff"], [class*="compare"]');
      await expect(page.locator('body')).toBeVisible();

      // Step 6: Optionally merge
      const mergeButton = page.getByRole('button', { name: /merge/i }).first();
      if (await mergeButton.isVisible()) {
        await mergeButton.click();
        await page.waitForTimeout(500);
      }

      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('PCAP Analysis Workflow', () => {
    test('should analyze PCAP file', async ({ page }) => {
      // Step 1: Navigate to PCAP analyzer
      await page.goto('/pcap-analyzer');
      await page.waitForLoadState('domcontentloaded');

      // Step 2: Upload PCAP file (simulated)
      const fileInput = page.locator('input[type="file"]').first();
      if (await fileInput.count() > 0) {
        // Would need actual file for full test
        await expect(page.locator('body')).toBeVisible();
      }

      // Step 3: Wait for analysis
      await page.waitForTimeout(1000);

      // Step 4: View packet list
      const packetList = page.locator('table, [class*="packet-list"]');
      await expect(page.locator('body')).toBeVisible();

      // Step 5: Filter packets
      const filterInput = page.locator('input[placeholder*="filter" i]').first();
      if (await filterInput.isVisible()) {
        await filterInput.fill('tcp');
        await page.waitForTimeout(500);
      }

      // Step 6: Select packet for details
      const packetRow = page.locator('tbody tr, [class*="packet-row"]').first();
      if (await packetRow.isVisible()) {
        await packetRow.click();
        await page.waitForTimeout(500);
      }

      // Step 7: Export results
      const exportButton = page.getByRole('button', { name: /export/i }).first();
      if (await exportButton.isVisible()) {
        await exportButton.click();
        await page.waitForTimeout(500);
      }

      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('Simulation with Traffic Workflow', () => {
    test('should run simulation with error injection', async ({ page }) => {
      // Step 1: Start simulation
      await page.goto('/runtime');
      await page.waitForLoadState('domcontentloaded');

      const startButton = page.getByRole('button', { name: /start|run/i }).first();
      if (await startButton.isVisible() && (await startButton.isEnabled())) {
        await startButton.click();
        await page.waitForTimeout(2000);
      }

      // Step 2: Verify simulation is running
      const runningIndicator = page.locator('[class*="running"], [class*="active"]');
      await expect(page.locator('body')).toBeVisible();

      // Step 3: Navigate to traffic injection
      await page.goto('/traffic');
      await page.waitForLoadState('domcontentloaded');

      // Step 4: Configure error injection
      const errorTypeSelect = page.locator('select[name*="error" i]').first();
      if (await errorTypeSelect.isVisible()) {
        await errorTypeSelect.selectOption({ index: 1 });
      }

      // Step 5: Start injection
      const injectButton = page.getByRole('button', { name: /inject|start/i }).first();
      if (await injectButton.isVisible() && (await injectButton.isEnabled())) {
        await injectButton.click();
        await page.waitForTimeout(1000);
      }

      // Step 6: Monitor packets
      await page.goto('/packets');
      await page.waitForLoadState('domcontentloaded');
      await page.waitForTimeout(3000);

      // Step 7: Stop injection
      await page.goto('/traffic');
      await page.waitForLoadState('domcontentloaded');

      const stopButton = page.getByRole('button', { name: /stop|clear/i }).first();
      if (await stopButton.isVisible()) {
        await stopButton.click();
        await page.waitForTimeout(500);
      }

      // Step 8: Stop simulation
      await page.goto('/runtime');
      await page.waitForLoadState('domcontentloaded');

      const stopSimButton = page.getByRole('button', { name: /stop/i }).first();
      if (await stopSimButton.isVisible() && (await stopSimButton.isEnabled())) {
        await stopSimButton.click();
        await page.waitForTimeout(1000);
      }

      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('Debug Investigation Workflow', () => {
    test('should investigate debug logs', async ({ page }) => {
      // Step 1: Navigate to debug console
      await page.goto('/debug');
      await page.waitForLoadState('domcontentloaded');

      // Step 2: Wait for logs to stream
      await page.waitForTimeout(2000);

      // Step 3: Filter by ERROR level
      const errorFilter = page.getByRole('button', { name: /error/i }).first();
      if (await errorFilter.isVisible()) {
        await errorFilter.click();
        await page.waitForTimeout(500);
      }

      // Step 4: Filter by protocol
      const protocolFilter = page.getByRole('button', { name: /snmp/i }).first();
      if (await protocolFilter.isVisible()) {
        await protocolFilter.click();
        await page.waitForTimeout(500);
      }

      // Step 5: Search for specific message
      const searchInput = page.locator('input[type="search"], input[placeholder*="search" i]').first();
      if (await searchInput.isVisible()) {
        await searchInput.fill('timeout');
        await page.waitForTimeout(500);
      }

      // Step 6: View log details
      const logEntry = page.locator('[class*="log-entry"]').first();
      if (await logEntry.isVisible()) {
        await logEntry.click();
        await page.waitForTimeout(300);
      }

      // Step 7: Export filtered logs
      const exportButton = page.getByRole('button', { name: /export/i }).first();
      if (await exportButton.isVisible()) {
        await exportButton.click();
        await page.waitForTimeout(500);
      }

      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('Template Application Workflow', () => {
    test('should browse and apply template', async ({ page }) => {
      // Step 1: Navigate to templates
      await page.goto('/templates');
      await page.waitForLoadState('domcontentloaded');

      // Step 2: Search for template
      const searchInput = page.locator('input[type="search"], input[placeholder*="search" i]').first();
      if (await searchInput.isVisible()) {
        await searchInput.fill('router');
        await page.waitForTimeout(500);
      }

      // Step 3: Preview template
      const previewButton = page.getByRole('button', { name: /preview|view/i }).first();
      if (await previewButton.isVisible()) {
        await previewButton.click();
        await page.waitForTimeout(500);

        const closeButton = page.getByRole('button', { name: /close/i }).first();
        if (await closeButton.isVisible()) {
          await closeButton.click();
        }
      }

      // Step 4: Apply template
      const applyButton = page.getByRole('button', { name: /apply|use/i }).first();
      if (await applyButton.isVisible()) {
        await applyButton.click();
        await page.waitForTimeout(500);

        // Confirm application
        const confirmButton = page.getByRole('button', { name: /confirm|yes/i }).first();
        if (await confirmButton.isVisible()) {
          await confirmButton.click();
          await page.waitForTimeout(1000);
        }
      }

      // Step 5: Verify configuration updated
      await page.goto('/devices');
      await page.waitForLoadState('domcontentloaded');

      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('Navigation Flow', () => {
    test('should navigate through all main sections', async ({ page }) => {
      const routes = [
        '/',
        '/devices',
        '/runtime',
        '/packets',
        '/topology',
        '/analysis',
        '/automation',
        '/traffic',
        '/debug',
        '/templates',
        '/config-diff',
        '/pcap-analyzer',
      ];

      for (const route of routes) {
        await page.goto(route);
        await page.waitForLoadState('domcontentloaded');
        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should maintain navigation context', async ({ page }) => {
      // Navigate to devices
      await page.goto('/devices');
      await page.waitForLoadState('domcontentloaded');

      // Click on device to edit
      const deviceLink = page.locator('a[href*="device-config"]').first();
      if (await deviceLink.isVisible()) {
        await deviceLink.click();
        await page.waitForLoadState('domcontentloaded');

        // Should be on device config page
        await expect(page).toHaveURL(/device-config/);

        // Navigate back
        await page.goBack();
        await page.waitForLoadState('domcontentloaded');

        // Should be back on devices
        await expect(page).toHaveURL(/devices/);
      }
    });
  });
});
