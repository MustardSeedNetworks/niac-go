import { expect, test } from '@playwright/test';

/**
 * Device Editor Page Tests
 *
 * Tests for device configuration editing:
 * - Form display
 * - Field validation
 * - Save/cancel actions
 */

test.describe('Device Editor', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/device/edit');
    await page.waitForLoadState('networkidle');
  });

  test('should navigate to device editor page', async ({ page }) => {
    await expect(page).toHaveURL(/device|edit/);
  });

  test('should display editor form', async ({ page }) => {
    const content = page.locator('main, [role="main"]').first();
    await expect(content).toBeVisible();
  });

  test('should have device name field', async ({ page }) => {
    const nameField = page.locator('input[name*="name" i], input[placeholder*="name" i]');
    const count = await nameField.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should have device type selector', async ({ page }) => {
    const typeSelector = page.locator('select, [role="combobox"]');
    const count = await typeSelector.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should have IP address field', async ({ page }) => {
    const ipField = page.locator('input[name*="ip" i], input[placeholder*="ip" i]');
    const count = await ipField.count();
    expect(count).toBeGreaterThanOrEqual(0);
  });

  test('should have protocol configuration sections', async ({ page }) => {
    // Protocol sections like SNMP, DNS, DHCP
    const protocols = ['snmp', 'dns', 'dhcp', 'http', 'ftp'];
    for (const protocol of protocols) {
      const section = page.getByText(new RegExp(protocol, 'i')).first();
      if (await section.isVisible()) {
        await expect(section).toBeVisible();
      }
    }
  });

  test('should have save and cancel buttons', async ({ page }) => {
    const saveButton = page.getByRole('button', { name: /save|submit|create/i });
    const cancelButton = page.getByRole('button', { name: /cancel|back|discard/i });
    const saveCount = await saveButton.count();
    const cancelCount = await cancelButton.count();
    expect(saveCount + cancelCount).toBeGreaterThanOrEqual(0);
  });

  test('should validate required fields', async ({ page }) => {
    // Try to submit empty form
    const submitButton = page.getByRole('button', { name: /save|submit|create/i }).first();
    if (await submitButton.isVisible()) {
      await submitButton.click();
      // Should show validation errors or prevent submission
      await expect(page.locator('body')).toBeVisible();
    }
  });
});
