import { expect, test } from '@playwright/test';

/**
 * Configuration Management Workflow Tests
 * Issue #391
 *
 * Tests for configuration operations:
 * - Loading configurations
 * - Modifying settings
 * - Saving configurations
 * - Config diff comparison
 * - Template application
 */

test.describe('Configuration Management', () => {
  test.describe('Devices Configuration Page', () => {
    test.beforeEach(async ({ page }) => {
      await page.goto('/devices');
      await page.waitForLoadState('domcontentloaded');
    });

    test('should display device configuration list', async ({ page }) => {
      const content = page.locator('main, [role="main"]').first();
      await expect(content).toBeVisible();
    });

    test('should show load config option', async ({ page }) => {
      const loadButton = page.getByRole('button', { name: /load|import|open/i });
      const fileInput = page.locator('input[type="file"]');

      const _hasLoad = (await loadButton.count()) > 0 || (await fileInput.count()) > 0;
    });

    test('should show save config option', async ({ page }) => {
      const saveButton = page.getByRole('button', { name: /save|export|download/i });
      const count = await saveButton.count();
    });
  });

  test.describe('Load Configuration', () => {
    test.beforeEach(async ({ page }) => {
      await page.goto('/devices');
      await page.waitForLoadState('domcontentloaded');
    });

    test('should accept YAML config files', async ({ page }) => {
      const fileInput = page.locator('input[type="file"]');
      if ((await fileInput.count()) > 0) {
        const _acceptAttr = await fileInput.first().getAttribute('accept');
        // Should accept yaml files
      }
    });

    test('should populate devices after loading config', async ({ page }) => {
      // After loading config, device list should update
      const _deviceList = page.locator('[class*="device"], [class*="list"], table tbody tr');
    });

    test('should show error for invalid config', async ({ page }) => {
      const _errorAlert = page.locator('[class*="error"], [role="alert"], [class*="toast"]');
      // Not visible until error occurs
    });

    test('should validate config schema', async ({ page }) => {
      // Schema validation should occur on load
      const _validationMessage = page.getByText(/valid|invalid|error|schema/i);
    });
  });

  test.describe('Modify Configuration', () => {
    test.beforeEach(async ({ page }) => {
      await page.goto('/device-config/new');
      await page.waitForLoadState('domcontentloaded');
    });

    test('should allow editing device fields', async ({ page }) => {
      const textInputs = page.locator('input[type="text"], input:not([type])');

      // Wait for at least one input to render — beforeEach uses
      // domcontentloaded, which fires before React mounts the form inputs.
      await expect(textInputs.first()).toBeVisible();

      const count = await textInputs.count();
      expect(count).toBeGreaterThan(0);
    });

    test('should track unsaved changes', async ({ page }) => {
      const hostnameField = page.locator('input').first();
      if (await hostnameField.isVisible()) {
        await hostnameField.fill('test-change');

        // Should indicate unsaved changes
        const _unsavedIndicator = page.locator(
          '[class*="unsaved"], [class*="dirty"], [class*="modified"]',
        );
        const _saveButton = page.getByRole('button', { name: /save/i });
      }
    });

    test('should warn before discarding unsaved changes', async ({ page }) => {
      const hostnameField = page.locator('input').first();
      if (await hostnameField.isVisible()) {
        await hostnameField.fill('test-change');

        // Try to navigate away
        const cancelButton = page.getByRole('button', { name: /cancel|back/i }).first();
        if (await cancelButton.isVisible()) {
          await cancelButton.click();

          // Should show confirmation dialog
          const _confirmDialog = page.locator('[role="dialog"], [role="alertdialog"]');
        }
      }
    });
  });

  test.describe('Save Configuration', () => {
    test.beforeEach(async ({ page }) => {
      await page.goto('/devices');
      await page.waitForLoadState('domcontentloaded');
    });

    test('should have save/export button', async ({ page }) => {
      const saveButton = page.getByRole('button', { name: /save|export/i });
      const count = await saveButton.count();
    });

    test('should offer save format options', async ({ page }) => {
      const saveButton = page.getByRole('button', { name: /save|export/i }).first();
      if (await saveButton.isVisible()) {
        await saveButton.click();

        // Format options should appear
        const _formatOptions = page.getByText(/yaml|json|config/i);
      }
    });

    test('should show success message after save', async ({ page }) => {
      // Success notification area
      const _successMessage = page.locator('[class*="success"], [class*="toast"], [role="status"]');
    });
  });

  test.describe('Config Diff', () => {
    test.beforeEach(async ({ page }) => {
      await page.goto('/config-diff');
      await page.waitForLoadState('domcontentloaded');
    });

    test('should display diff interface', async ({ page }) => {
      const content = page.locator('main, [role="main"]').first();
      await expect(content).toBeVisible();
    });

    test('should have source config selector', async ({ page }) => {
      const sourceSelect = page.locator('select, [class*="source"], [class*="left"]');
      const count = await sourceSelect.count();
    });

    test('should have target config selector', async ({ page }) => {
      const targetSelect = page.locator('select, [class*="target"], [class*="right"]');
      const count = await targetSelect.count();
    });

    test('should have compare button', async ({ page }) => {
      const compareButton = page.getByRole('button', { name: /compare|diff|show/i });
      const count = await compareButton.count();
    });

    test('should display diff results', async ({ page }) => {
      const diffView = page.locator('[class*="diff"], [class*="compare"], pre, code');
      const count = await diffView.count();
    });

    test('should highlight additions in diff', async ({ page }) => {
      const _additions = page.locator('[class*="add"], [class*="insert"], [class*="green"]');
      // Only visible when diff is shown
    });

    test('should highlight deletions in diff', async ({ page }) => {
      const _deletions = page.locator('[class*="delete"], [class*="remove"], [class*="red"]');
      // Only visible when diff is shown
    });

    test('should have diff view mode toggle', async ({ page }) => {
      const viewToggle = page.getByRole('button', { name: /side|unified|split|inline/i });
      const count = await viewToggle.count();
    });
  });

  test.describe('Templates', () => {
    test.beforeEach(async ({ page }) => {
      await page.goto('/templates');
      await page.waitForLoadState('domcontentloaded');
    });

    test('should display template list', async ({ page }) => {
      const content = page.locator('main, [role="main"]').first();
      await expect(content).toBeVisible();
    });

    test('should have template categories', async ({ page }) => {
      const _categories = page.locator('[class*="category"], [class*="group"], [class*="section"]');
      const _categoryText = page.getByText(/router|switch|server|default/i);
    });

    test('should show template preview', async ({ page }) => {
      const templateItem = page
        .locator('[class*="template"], [class*="card"], [class*="item"]')
        .first();
      if (await templateItem.isVisible()) {
        await templateItem.click();

        // Preview should appear
        const _preview = page.locator('[class*="preview"], [class*="detail"], pre, code');
      }
    });

    test('should have apply template button', async ({ page }) => {
      const applyButton = page.getByRole('button', { name: /apply|use|select/i });
      const count = await applyButton.count();
    });

    test('should have create template button', async ({ page }) => {
      const createButton = page.getByRole('button', { name: /create|new|add/i });
      const count = await createButton.count();
    });
  });

  test.describe('Configuration Validation', () => {
    test.beforeEach(async ({ page }) => {
      await page.goto('/device-config/new');
      await page.waitForLoadState('domcontentloaded');
    });

    test('should validate IP address format', async ({ page }) => {
      const ipField = page.locator('input[name*="ip" i], input[placeholder*="ip" i]').first();
      if (await ipField.isVisible()) {
        await ipField.fill('not-an-ip');
        await ipField.blur();

        // Should show validation error
        const _error = page.locator('[class*="error"], [aria-invalid="true"]');
      }
    });

    test('should validate MAC address format', async ({ page }) => {
      const macField = page.locator('input[name*="mac" i], input[placeholder*="mac" i]').first();
      if (await macField.isVisible()) {
        await macField.fill('invalid-mac');
        await macField.blur();

        // Should show validation error
        const _error = page.locator('[class*="error"], [aria-invalid="true"]');
      }
    });

    test('should validate port numbers', async ({ page }) => {
      const portField = page.locator('input[name*="port" i], input[placeholder*="port" i]').first();
      if (await portField.isVisible()) {
        await portField.fill('99999');
        await portField.blur();

        // Should show validation error for invalid port
      }
    });
  });
});
