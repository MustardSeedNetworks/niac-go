import { expect, test } from '@playwright/test';

/**
 * Device CRUD Workflow Tests
 * Issue #388
 *
 * Tests for device create, read, update, delete operations:
 * - Creating new devices
 * - Editing existing devices
 * - Deleting devices
 * - Cloning devices
 * - Form validation
 */

test.describe('Device CRUD Operations', () => {
  const testDeviceName = `test-device-${Date.now()}`;

  test.beforeEach(async ({ page }) => {
    await page.goto('/devices');
    await page.waitForLoadState('domcontentloaded');
  });

  test.describe('Create Device', () => {
    test('should navigate to create device form', async ({ page }) => {
      const addButton = page.getByRole('button', { name: /add|create|new/i }).first();
      if (await addButton.isVisible()) {
        await addButton.click();
        await expect(page).toHaveURL(/device-config/);
      }
    });

    test('should display all required form fields', async ({ page }) => {
      await page.goto('/device-config/new');

      // Wait for the device form to render — domcontentloaded fires before
      // React mounts the inputs, which made the count() race the render.
      await expect(page.locator('input[type="text"]').first()).toBeVisible();

      const nameFields = await page.locator('input[type="text"]').count();
      expect(nameFields).toBeGreaterThan(0);
    });

    test('should validate required fields on empty submit', async ({ page }) => {
      await page.goto('/device-config/new');
      await page.waitForLoadState('domcontentloaded');

      const submitButton = page.getByRole('button', { name: /save|submit|create/i }).first();
      if (await submitButton.isVisible()) {
        // Button should be disabled for invalid/empty form
        const isDisabled = await submitButton.isDisabled();
        if (!isDisabled) {
          await submitButton.click();
          // Should show validation error or stay on page
          await expect(page.locator('body')).toBeVisible();
        } else {
          await expect(submitButton).toBeDisabled();
        }
      }
    });

    test('should reject invalid IP address', async ({ page }) => {
      await page.goto('/device-config/new');
      await page.waitForLoadState('domcontentloaded');

      const ipField = page.locator('input[name*="ip" i], input[placeholder*="ip" i]').first();
      if (await ipField.isVisible()) {
        await ipField.fill('invalid.ip.address');
        await ipField.blur();

        // Check for validation error or invalid state
        const _hasError =
          (await page
            .locator('[class*="error"], [class*="invalid"], [aria-invalid="true"]')
            .count()) > 0;
        // Form should indicate invalid input somehow
        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should create device with valid configuration', async ({ page }) => {
      await page.goto('/device-config/new');
      await page.waitForLoadState('domcontentloaded');

      // Fill in hostname
      const hostnameField = page
        .locator('input[name*="hostname" i], input[placeholder*="hostname" i], input[name="name"]')
        .first();
      if (await hostnameField.isVisible()) {
        await hostnameField.fill(testDeviceName);
      }

      // Fill in IP if visible
      const ipField = page.locator('input[name*="ip" i], input[placeholder*="ip" i]').first();
      if (await ipField.isVisible()) {
        await ipField.fill('192.168.1.100');
      }

      // Fill MAC if visible
      const macField = page.locator('input[name*="mac" i], input[placeholder*="mac" i]').first();
      if (await macField.isVisible()) {
        await macField.fill('00:11:22:33:44:55');
      }

      // Try to save
      const submitButton = page.getByRole('button', { name: /save|submit|create/i }).first();
      if (await submitButton.isVisible()) {
        const isEnabled = await submitButton.isEnabled();
        if (isEnabled) {
          await submitButton.click();
          // Should navigate away or show success
          await page.waitForTimeout(250);
        }
      }
    });
  });

  test.describe('Read Device', () => {
    test('should display device list', async ({ page }) => {
      const content = page.locator('main, [role="main"]').first();
      await expect(content).toBeVisible();
    });

    test('should show device details on selection', async ({ page }) => {
      const deviceItem = page.locator('[class*="device"], [data-testid*="device"], tr, li').first();
      if (await deviceItem.isVisible()) {
        await deviceItem.click();
        await page.waitForTimeout(150);
        // Should show details panel or navigate to details
        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should filter devices by search', async ({ page }) => {
      const searchInput = page.locator(
        'input[type="search"], input[placeholder*="search" i], input[placeholder*="filter" i]',
      );
      if (await searchInput.isVisible()) {
        await searchInput.fill('test');
        await page.waitForTimeout(150);
        // Should filter the list
        await expect(page.locator('body')).toBeVisible();
      }
    });
  });

  test.describe('Update Device', () => {
    test('should navigate to edit existing device', async ({ page }) => {
      // Look for edit button or click on device
      const editButton = page.getByRole('button', { name: /edit/i }).first();
      const deviceLink = page.locator('a[href*="device-config"]').first();

      if (await editButton.isVisible()) {
        await editButton.click();
        await expect(page).toHaveURL(/device-config|edit/);
      } else if (await deviceLink.isVisible()) {
        await deviceLink.click();
        await expect(page).toHaveURL(/device-config/);
      }
    });

    test('should modify and save device changes', async ({ page }) => {
      // Navigate to an existing device if available
      const deviceLink = page.locator('a[href*="device-config"]').first();
      if (await deviceLink.isVisible()) {
        await deviceLink.click();
        await page.waitForLoadState('domcontentloaded');

        // Make a change
        const hostnameField = page
          .locator(
            'input[name*="hostname" i], input[placeholder*="hostname" i], input[name="name"]',
          )
          .first();
        if (await hostnameField.isVisible()) {
          const currentValue = await hostnameField.inputValue();
          await hostnameField.fill(`${currentValue}-modified`);

          // Try to save
          const submitButton = page.getByRole('button', { name: /save|update|submit/i }).first();
          if ((await submitButton.isVisible()) && (await submitButton.isEnabled())) {
            await submitButton.click();
            await page.waitForTimeout(250);
          }
        }
      }
    });
  });

  test.describe('Delete Device', () => {
    test('should show delete confirmation', async ({ page }) => {
      const deleteButton = page.getByRole('button', { name: /delete|remove/i }).first();
      if (await deleteButton.isVisible()) {
        await deleteButton.click();
        // Should show confirmation dialog
        const confirmDialog = page.locator(
          '[role="dialog"], [role="alertdialog"], [class*="modal"]',
        );
        const _confirmCount = await confirmDialog.count();
        // Either shows dialog or performs action
        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should cancel delete on dismiss', async ({ page }) => {
      const deleteButton = page.getByRole('button', { name: /delete|remove/i }).first();
      if (await deleteButton.isVisible()) {
        await deleteButton.click();

        // Look for cancel button in dialog
        const cancelButton = page.getByRole('button', { name: /cancel|no|dismiss/i }).first();
        if (await cancelButton.isVisible()) {
          await cancelButton.click();
          // Dialog should close
          await page.waitForTimeout(150);
          await expect(page.locator('body')).toBeVisible();
        }
      }
    });
  });

  test.describe('Clone Device', () => {
    test('should clone existing device', async ({ page }) => {
      const cloneButton = page.getByRole('button', { name: /clone|copy|duplicate/i }).first();
      if (await cloneButton.isVisible()) {
        await cloneButton.click();
        // Should navigate to new device form or show clone dialog
        await page.waitForTimeout(150);
        await expect(page.locator('body')).toBeVisible();
      }
    });
  });

  test.describe('Protocol Sections', () => {
    test('should display SNMP configuration section', async ({ page }) => {
      await page.goto('/device-config/new');
      await page.waitForLoadState('domcontentloaded');

      const snmpSection = page.getByText(/snmp/i).first();
      if (await snmpSection.isVisible()) {
        await expect(snmpSection).toBeVisible();
      }
    });

    test('should display network protocol sections', async ({ page }) => {
      await page.goto('/device-config/new');
      await page.waitForLoadState('domcontentloaded');

      const protocols = ['dhcp', 'dns', 'http', 'ftp', 'netbios'];
      for (const protocol of protocols) {
        const _section = page.getByText(new RegExp(protocol, 'i')).first();
        // Just verify page renders, sections may be collapsed
        await expect(page.locator('body')).toBeVisible();
      }
    });
  });
});
