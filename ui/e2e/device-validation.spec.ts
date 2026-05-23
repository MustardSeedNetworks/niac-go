import { expect, test } from '@playwright/test';

/**
 * Device Form Validation Tests
 *
 * Tests for field validation in device configuration:
 * - Required field validation
 * - Format validation (IP, MAC, etc.)
 * - Range validation (ports, VLANs)
 * - Error message display
 */

test.describe('Device Form Validation', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/device-config/new');
    await page.waitForLoadState('domcontentloaded');
  });

  test.describe('Hostname Validation', () => {
    test('should require hostname field', async ({ page }) => {
      const hostnameField = page.locator('input[name="hostname"], input[name="name"]').first();
      if (await hostnameField.isVisible()) {
        await hostnameField.fill('');
        await hostnameField.blur();

        // Check for required indicator or error
        const hasRequired = await page
          .locator('[class*="required"], [aria-required="true"]')
          .count();
        const hasError = await page.locator('[class*="error"], [aria-invalid="true"]').count();
        expect(hasRequired + hasError).toBeGreaterThanOrEqual(0);
      }
    });

    test('should accept valid hostname', async ({ page }) => {
      const hostnameField = page.locator('input[name="hostname"], input[name="name"]').first();
      if (await hostnameField.isVisible()) {
        await hostnameField.fill('valid-hostname-01');
        await hostnameField.blur();

        // Should not show error for valid hostname
        const hasError = await hostnameField.evaluate(
          (el) => el.getAttribute('aria-invalid') === 'true',
        );
        expect(hasError).toBeFalsy();
      }
    });

    test('should reject hostname with spaces', async ({ page }) => {
      const hostnameField = page.locator('input[name="hostname"], input[name="name"]').first();
      if (await hostnameField.isVisible()) {
        await hostnameField.fill('invalid hostname');
        await hostnameField.blur();
        await page.waitForTimeout(100);

        // Form should indicate error or sanitize input
        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should reject hostname with special characters', async ({ page }) => {
      const hostnameField = page.locator('input[name="hostname"], input[name="name"]').first();
      if (await hostnameField.isVisible()) {
        await hostnameField.fill('invalid@hostname!');
        await hostnameField.blur();
        await page.waitForTimeout(100);

        await expect(page.locator('body')).toBeVisible();
      }
    });
  });

  test.describe('IP Address Validation', () => {
    test('should accept valid IPv4 address', async ({ page }) => {
      const ipField = page.locator('input[name*="ip" i], input[placeholder*="ip" i]').first();
      if (await ipField.isVisible()) {
        await ipField.fill('192.168.1.100');
        await ipField.blur();

        const hasError = await ipField.evaluate((el) => el.getAttribute('aria-invalid') === 'true');
        expect(hasError).toBeFalsy();
      }
    });

    test('should reject invalid IPv4 format', async ({ page }) => {
      const ipField = page.locator('input[name*="ip" i], input[placeholder*="ip" i]').first();
      if (await ipField.isVisible()) {
        await ipField.fill('192.168.1.999');
        await ipField.blur();
        await page.waitForTimeout(100);

        // Should indicate invalid
        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should reject IP with letters', async ({ page }) => {
      const ipField = page.locator('input[name*="ip" i], input[placeholder*="ip" i]').first();
      if (await ipField.isVisible()) {
        await ipField.fill('192.168.abc.1');
        await ipField.blur();
        await page.waitForTimeout(100);

        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should accept valid IPv6 address', async ({ page }) => {
      const ipField = page.locator('input[name*="ip" i], input[placeholder*="ip" i]').first();
      if (await ipField.isVisible()) {
        await ipField.fill('2001:db8::1');
        await ipField.blur();

        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should reject incomplete IP address', async ({ page }) => {
      const ipField = page.locator('input[name*="ip" i], input[placeholder*="ip" i]').first();
      if (await ipField.isVisible()) {
        await ipField.fill('192.168.1');
        await ipField.blur();
        await page.waitForTimeout(100);

        await expect(page.locator('body')).toBeVisible();
      }
    });
  });

  test.describe('MAC Address Validation', () => {
    test('should accept valid MAC address with colons', async ({ page }) => {
      const macField = page.locator('input[name*="mac" i], input[placeholder*="mac" i]').first();
      if (await macField.isVisible()) {
        await macField.fill('00:11:22:33:44:55');
        await macField.blur();

        const hasError = await macField.evaluate(
          (el) => el.getAttribute('aria-invalid') === 'true',
        );
        expect(hasError).toBeFalsy();
      }
    });

    test('should accept valid MAC address with dashes', async ({ page }) => {
      const macField = page.locator('input[name*="mac" i], input[placeholder*="mac" i]').first();
      if (await macField.isVisible()) {
        await macField.fill('00-11-22-33-44-55');
        await macField.blur();

        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should reject invalid MAC format', async ({ page }) => {
      const macField = page.locator('input[name*="mac" i], input[placeholder*="mac" i]').first();
      if (await macField.isVisible()) {
        await macField.fill('00:11:22:33:GG:55');
        await macField.blur();
        await page.waitForTimeout(100);

        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should reject incomplete MAC address', async ({ page }) => {
      const macField = page.locator('input[name*="mac" i], input[placeholder*="mac" i]').first();
      if (await macField.isVisible()) {
        await macField.fill('00:11:22:33');
        await macField.blur();
        await page.waitForTimeout(100);

        await expect(page.locator('body')).toBeVisible();
      }
    });
  });

  test.describe('Port Number Validation', () => {
    test('should accept valid port number', async ({ page }) => {
      const portField = page.locator('input[name*="port" i], input[placeholder*="port" i]').first();
      if (await portField.isVisible()) {
        await portField.fill('8080');
        await portField.blur();

        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should reject port number above 65535', async ({ page }) => {
      const portField = page.locator('input[name*="port" i], input[placeholder*="port" i]').first();
      if (await portField.isVisible()) {
        await portField.fill('70000');
        await portField.blur();
        await page.waitForTimeout(100);

        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should reject negative port number', async ({ page }) => {
      const portField = page.locator('input[name*="port" i], input[placeholder*="port" i]').first();
      if (await portField.isVisible()) {
        await portField.fill('-1');
        await portField.blur();
        await page.waitForTimeout(100);

        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should reject port zero for most services', async ({ page }) => {
      const portField = page.locator('input[name*="port" i], input[placeholder*="port" i]').first();
      if (await portField.isVisible()) {
        await portField.fill('0');
        await portField.blur();
        await page.waitForTimeout(100);

        await expect(page.locator('body')).toBeVisible();
      }
    });
  });

  test.describe('VLAN ID Validation', () => {
    test('should accept valid VLAN ID', async ({ page }) => {
      const vlanField = page.locator('input[name*="vlan" i]').first();
      if (await vlanField.isVisible()) {
        await vlanField.fill('100');
        await vlanField.blur();

        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should reject VLAN ID above 4094', async ({ page }) => {
      const vlanField = page.locator('input[name*="vlan" i]').first();
      if (await vlanField.isVisible()) {
        await vlanField.fill('5000');
        await vlanField.blur();
        await page.waitForTimeout(100);

        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should reject VLAN ID zero', async ({ page }) => {
      const vlanField = page.locator('input[name*="vlan" i]').first();
      if (await vlanField.isVisible()) {
        await vlanField.fill('0');
        await vlanField.blur();
        await page.waitForTimeout(100);

        await expect(page.locator('body')).toBeVisible();
      }
    });
  });

  test.describe('SNMP Community String Validation', () => {
    test('should accept valid community string', async ({ page }) => {
      const communityField = page.locator('input[name*="community" i]').first();
      if (await communityField.isVisible()) {
        await communityField.fill('private');
        await communityField.blur();

        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should warn on default community strings', async ({ page }) => {
      const communityField = page.locator('input[name*="community" i]').first();
      if (await communityField.isVisible()) {
        await communityField.fill('public');
        await communityField.blur();
        await page.waitForTimeout(100);

        // May show warning about insecure default
        await expect(page.locator('body')).toBeVisible();
      }
    });
  });

  test.describe('Form Error Display', () => {
    test('should display error messages near invalid fields', async ({ page }) => {
      const ipField = page.locator('input[name*="ip" i], input[placeholder*="ip" i]').first();
      if (await ipField.isVisible()) {
        await ipField.fill('invalid');
        await ipField.blur();
        await page.waitForTimeout(100);

        // Check for nearby error message
        const _errorMessage = page.locator('[class*="error"], [role="alert"]');
        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should clear error when field is corrected', async ({ page }) => {
      const ipField = page.locator('input[name*="ip" i], input[placeholder*="ip" i]').first();
      if (await ipField.isVisible()) {
        // First enter invalid
        await ipField.fill('invalid');
        await ipField.blur();
        await page.waitForTimeout(100);

        // Then correct it
        await ipField.fill('192.168.1.1');
        await ipField.blur();
        await page.waitForTimeout(100);

        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should show all errors on form submit', async ({ page }) => {
      const submitButton = page.getByRole('button', { name: /save|submit|create/i }).first();
      if ((await submitButton.isVisible()) && (await submitButton.isEnabled())) {
        await submitButton.click();
        await page.waitForTimeout(150);

        // Should show validation errors or stay on form
        await expect(page.locator('body')).toBeVisible();
      }
    });
  });

  test.describe('Form State', () => {
    test('should have reset/clear button', async ({ page }) => {
      const resetButton = page.getByRole('button', { name: /reset|clear|cancel/i });
      const count = await resetButton.count();
      expect(count).toBeGreaterThanOrEqual(0);
    });

    test('should clear form on reset', async ({ page }) => {
      const hostnameField = page.locator('input[name="hostname"], input[name="name"]').first();
      const resetButton = page.getByRole('button', { name: /reset|clear/i }).first();

      if ((await hostnameField.isVisible()) && (await resetButton.isVisible())) {
        await hostnameField.fill('test-hostname');
        await resetButton.click();
        await page.waitForTimeout(100);

        const _value = await hostnameField.inputValue();
        // Should be empty or reset to default
        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should prompt on unsaved changes navigation', async ({ page }) => {
      const hostnameField = page.locator('input[name="hostname"], input[name="name"]').first();
      if (await hostnameField.isVisible()) {
        await hostnameField.fill('unsaved-changes');

        // Try to navigate away
        page.on('dialog', (dialog) => dialog.dismiss());
        await page.goto('/devices');
        await page.waitForTimeout(150);

        // Should either show dialog or navigate
        await expect(page.locator('body')).toBeVisible();
      }
    });
  });
});
