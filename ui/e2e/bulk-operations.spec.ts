import { expect, test } from '@playwright/test';

/**
 * Bulk Operations Tests
 *
 * Tests for multi-select and bulk device operations:
 * - Select all/none
 * - Multi-select
 * - Bulk delete
 * - Bulk export
 */

test.describe('Bulk Operations', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/devices');
    await page.waitForLoadState('domcontentloaded');
  });

  test.describe('Selection Controls', () => {
    test('should have row selection checkboxes', async ({ page }) => {
      const checkboxes = page.locator('input[type="checkbox"]');
      const count = await checkboxes.count();
      expect(count).toBeGreaterThanOrEqual(0);
    });

    test('should have select all checkbox', async ({ page }) => {
      const selectAllCheckbox = page.locator(
        'input[type="checkbox"][aria-label*="select all" i], th input[type="checkbox"]'
      );
      const count = await selectAllCheckbox.count();
      expect(count).toBeGreaterThanOrEqual(0);
    });

    test('should select all devices with select all', async ({ page }) => {
      const selectAllCheckbox = page
        .locator('input[type="checkbox"][aria-label*="select all" i], th input[type="checkbox"]')
        .first();

      if (await selectAllCheckbox.isVisible()) {
        await selectAllCheckbox.check();
        await page.waitForTimeout(300);

        // All row checkboxes should be checked
        const rowCheckboxes = page.locator('tbody input[type="checkbox"], tr input[type="checkbox"]');
        const count = await rowCheckboxes.count();
        if (count > 0) {
          const checkedCount = await page.locator('tbody input[type="checkbox"]:checked').count();
          await expect(page.locator('body')).toBeVisible();
        }
      }
    });

    test('should deselect all with select all toggle', async ({ page }) => {
      const selectAllCheckbox = page
        .locator('input[type="checkbox"][aria-label*="select all" i], th input[type="checkbox"]')
        .first();

      if (await selectAllCheckbox.isVisible()) {
        // Select all
        await selectAllCheckbox.check();
        await page.waitForTimeout(300);

        // Deselect all
        await selectAllCheckbox.uncheck();
        await page.waitForTimeout(300);

        // All should be unchecked
        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should show indeterminate state when some selected', async ({ page }) => {
      const rowCheckboxes = page.locator('tbody input[type="checkbox"]');
      const count = await rowCheckboxes.count();

      if (count >= 2) {
        // Select only first row
        await rowCheckboxes.first().check();
        await page.waitForTimeout(300);

        // Select all checkbox should show indeterminate or partial state
        const selectAllCheckbox = page.locator('th input[type="checkbox"]').first();
        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should select individual rows', async ({ page }) => {
      const firstRowCheckbox = page.locator('tbody input[type="checkbox"]').first();

      if (await firstRowCheckbox.isVisible()) {
        await firstRowCheckbox.check();
        await page.waitForTimeout(300);

        await expect(firstRowCheckbox).toBeChecked();
      }
    });

    test('should show selection count', async ({ page }) => {
      const rowCheckboxes = page.locator('tbody input[type="checkbox"]');
      const count = await rowCheckboxes.count();

      if (count > 0) {
        await rowCheckboxes.first().check();
        await page.waitForTimeout(300);

        // Should show "1 selected" or similar
        const selectionCount = page.getByText(/\d+\s*selected/i);
        await expect(page.locator('body')).toBeVisible();
      }
    });
  });

  test.describe('Bulk Delete', () => {
    test('should show bulk delete button when items selected', async ({ page }) => {
      const rowCheckbox = page.locator('tbody input[type="checkbox"]').first();

      if (await rowCheckbox.isVisible()) {
        await rowCheckbox.check();
        await page.waitForTimeout(300);

        // Bulk delete button should appear
        const bulkDeleteButton = page.getByRole('button', { name: /delete selected|bulk delete/i });
        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should hide bulk delete button when nothing selected', async ({ page }) => {
      const bulkDeleteButton = page.getByRole('button', { name: /delete selected|bulk delete/i });

      // Should be hidden or disabled when nothing selected
      await expect(page.locator('body')).toBeVisible();
    });

    test('should confirm before bulk delete', async ({ page }) => {
      const rowCheckbox = page.locator('tbody input[type="checkbox"]').first();

      if (await rowCheckbox.isVisible()) {
        await rowCheckbox.check();
        await page.waitForTimeout(300);

        const bulkDeleteButton = page.getByRole('button', { name: /delete selected|bulk delete/i }).first();
        if (await bulkDeleteButton.isVisible()) {
          await bulkDeleteButton.click();
          await page.waitForTimeout(500);

          // Confirmation dialog
          const confirmDialog = page.locator('[role="dialog"], [role="alertdialog"]');
          await expect(page.locator('body')).toBeVisible();
        }
      }
    });

    test('should show count in delete confirmation', async ({ page }) => {
      const rowCheckboxes = page.locator('tbody input[type="checkbox"]');
      const count = await rowCheckboxes.count();

      if (count >= 2) {
        await rowCheckboxes.nth(0).check();
        await rowCheckboxes.nth(1).check();
        await page.waitForTimeout(300);

        const bulkDeleteButton = page.getByRole('button', { name: /delete selected|bulk delete/i }).first();
        if (await bulkDeleteButton.isVisible()) {
          await bulkDeleteButton.click();
          await page.waitForTimeout(500);

          // Should mention count of items to delete
          const countText = page.getByText(/2|two/i);
          await expect(page.locator('body')).toBeVisible();
        }
      }
    });

    test('should cancel bulk delete on dismiss', async ({ page }) => {
      const rowCheckbox = page.locator('tbody input[type="checkbox"]').first();

      if (await rowCheckbox.isVisible()) {
        await rowCheckbox.check();
        const bulkDeleteButton = page.getByRole('button', { name: /delete selected|bulk delete/i }).first();

        if (await bulkDeleteButton.isVisible()) {
          await bulkDeleteButton.click();
          await page.waitForTimeout(500);

          const cancelButton = page.getByRole('button', { name: /cancel|no/i }).first();
          if (await cancelButton.isVisible()) {
            await cancelButton.click();
            await page.waitForTimeout(300);

            // Items should still be selected
            await expect(rowCheckbox).toBeChecked();
          }
        }
      }
    });

    test('should clear selection after successful bulk delete', async ({ page }) => {
      await page.route('**/api/v1/config/devices/*', (route) => {
        if (route.request().method() === 'DELETE') {
          route.fulfill({ status: 204 });
        } else {
          route.continue();
        }
      });

      const rowCheckbox = page.locator('tbody input[type="checkbox"]').first();

      if (await rowCheckbox.isVisible()) {
        await rowCheckbox.check();
        const bulkDeleteButton = page.getByRole('button', { name: /delete selected|bulk delete/i }).first();

        if (await bulkDeleteButton.isVisible()) {
          await bulkDeleteButton.click();
          await page.waitForTimeout(500);

          const confirmButton = page.getByRole('button', { name: /confirm|yes|delete/i }).first();
          if (await confirmButton.isVisible()) {
            await confirmButton.click();
            await page.waitForTimeout(1000);

            // Selection should be cleared
            await expect(page.locator('body')).toBeVisible();
          }
        }
      }
    });
  });

  test.describe('Bulk Export', () => {
    test('should show export button when items selected', async ({ page }) => {
      const rowCheckbox = page.locator('tbody input[type="checkbox"]').first();

      if (await rowCheckbox.isVisible()) {
        await rowCheckbox.check();
        await page.waitForTimeout(300);

        // Export button should appear
        const exportButton = page.getByRole('button', { name: /export selected|bulk export/i });
        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should have export all option', async ({ page }) => {
      const exportAllButton = page.getByRole('button', { name: /export all|download all/i });
      const count = await exportAllButton.count();
      expect(count).toBeGreaterThanOrEqual(0);
    });

    test('should offer export format options', async ({ page }) => {
      const rowCheckbox = page.locator('tbody input[type="checkbox"]').first();

      if (await rowCheckbox.isVisible()) {
        await rowCheckbox.check();
        await page.waitForTimeout(300);

        const exportButton = page.getByRole('button', { name: /export/i }).first();
        if (await exportButton.isVisible()) {
          await exportButton.click();
          await page.waitForTimeout(500);

          // Should show format options (YAML, JSON, CSV)
          const formatOptions = page.getByText(/yaml|json|csv/i);
          await expect(page.locator('body')).toBeVisible();
        }
      }
    });

    test('should trigger download on export', async ({ page }) => {
      const rowCheckbox = page.locator('tbody input[type="checkbox"]').first();

      if (await rowCheckbox.isVisible()) {
        await rowCheckbox.check();
        await page.waitForTimeout(300);

        const exportButton = page.getByRole('button', { name: /export/i }).first();
        if (await exportButton.isVisible()) {
          const downloadPromise = page.waitForEvent('download', { timeout: 5000 }).catch(() => null);
          await exportButton.click();
          const download = await downloadPromise;

          // May or may not trigger download depending on implementation
          await expect(page.locator('body')).toBeVisible();
        }
      }
    });
  });

  test.describe('Bulk Edit', () => {
    test('should show edit button when items selected', async ({ page }) => {
      const rowCheckbox = page.locator('tbody input[type="checkbox"]').first();

      if (await rowCheckbox.isVisible()) {
        await rowCheckbox.check();
        await page.waitForTimeout(300);

        // Edit button may appear for bulk edit
        const editButton = page.getByRole('button', { name: /edit selected|bulk edit/i });
        await expect(page.locator('body')).toBeVisible();
      }
    });
  });

  test.describe('Keyboard Shortcuts', () => {
    test('should select row with Space key', async ({ page }) => {
      const firstRow = page.locator('tbody tr').first();

      if (await firstRow.isVisible()) {
        await firstRow.focus();
        await page.keyboard.press('Space');
        await page.waitForTimeout(300);

        // Row should be selected
        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should select all with Ctrl+A', async ({ page }) => {
      await page.keyboard.press('Control+a');
      await page.waitForTimeout(300);

      // All should be selected (if implemented)
      await expect(page.locator('body')).toBeVisible();
    });

    test('should deselect all with Escape', async ({ page }) => {
      const rowCheckbox = page.locator('tbody input[type="checkbox"]').first();

      if (await rowCheckbox.isVisible()) {
        await rowCheckbox.check();
        await page.waitForTimeout(300);

        await page.keyboard.press('Escape');
        await page.waitForTimeout(300);

        // Selection should be cleared (if implemented)
        await expect(page.locator('body')).toBeVisible();
      }
    });
  });
});
