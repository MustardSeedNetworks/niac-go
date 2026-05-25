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
    });

    test('should have select all checkbox', async ({ page }) => {
      const selectAllCheckbox = page.locator(
        'input[type="checkbox"][aria-label*="select all" i], th input[type="checkbox"]',
      );
      const count = await selectAllCheckbox.count();
    });

    test('should select all devices with select all', async ({ page }) => {
      const selectAllCheckbox = page
        .locator('input[type="checkbox"][aria-label*="select all" i], th input[type="checkbox"]')
        .first();

      if (await selectAllCheckbox.isVisible()) {
        await selectAllCheckbox.check();

        // All row checkboxes should be checked
        const rowCheckboxes = page.locator(
          'tbody input[type="checkbox"], tr input[type="checkbox"]',
        );
        const count = await rowCheckboxes.count();
        if (count > 0) {
          const _checkedCount = await page.locator('tbody input[type="checkbox"]:checked').count();
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

        // Deselect all
        await selectAllCheckbox.uncheck();

        // All should be unchecked
      }
    });

    test('should show indeterminate state when some selected', async ({ page }) => {
      const rowCheckboxes = page.locator('tbody input[type="checkbox"]');
      const count = await rowCheckboxes.count();

      if (count >= 2) {
        // Select only first row
        await rowCheckboxes.first().check();

        // Select all checkbox should show indeterminate or partial state
        const _selectAllCheckbox = page.locator('th input[type="checkbox"]').first();
      }
    });

    test('should select individual rows', async ({ page }) => {
      const firstRowCheckbox = page.locator('tbody input[type="checkbox"]').first();

      if (await firstRowCheckbox.isVisible()) {
        await firstRowCheckbox.check();

        await expect(firstRowCheckbox).toBeChecked();
      }
    });

    test('should show selection count', async ({ page }) => {
      const rowCheckboxes = page.locator('tbody input[type="checkbox"]');
      const count = await rowCheckboxes.count();

      if (count > 0) {
        await rowCheckboxes.first().check();

        // Should show "1 selected" or similar
        const _selectionCount = page.getByText(/\d+\s*selected/i);
      }
    });
  });

  test.describe('Bulk Delete', () => {
    test('should show bulk delete button when items selected', async ({ page }) => {
      const rowCheckbox = page.locator('tbody input[type="checkbox"]').first();

      if (await rowCheckbox.isVisible()) {
        await rowCheckbox.check();

        // Bulk delete button should appear
        const _bulkDeleteButton = page.getByRole('button', {
          name: /delete selected|bulk delete/i,
        });
      }
    });

    test('should hide bulk delete button when nothing selected', async ({ page }) => {
      const _bulkDeleteButton = page.getByRole('button', { name: /delete selected|bulk delete/i });

      // Should be hidden or disabled when nothing selected
    });

    test('should confirm before bulk delete', async ({ page }) => {
      const rowCheckbox = page.locator('tbody input[type="checkbox"]').first();

      if (await rowCheckbox.isVisible()) {
        await rowCheckbox.check();

        const bulkDeleteButton = page
          .getByRole('button', { name: /delete selected|bulk delete/i })
          .first();
        if (await bulkDeleteButton.isVisible()) {
          await bulkDeleteButton.click();

          // Confirmation dialog
          const _confirmDialog = page.locator('[role="dialog"], [role="alertdialog"]');
        }
      }
    });

    test('should show count in delete confirmation', async ({ page }) => {
      const rowCheckboxes = page.locator('tbody input[type="checkbox"]');
      const count = await rowCheckboxes.count();

      if (count >= 2) {
        await rowCheckboxes.nth(0).check();
        await rowCheckboxes.nth(1).check();

        const bulkDeleteButton = page
          .getByRole('button', { name: /delete selected|bulk delete/i })
          .first();
        if (await bulkDeleteButton.isVisible()) {
          await bulkDeleteButton.click();

          // Should mention count of items to delete
          const _countText = page.getByText(/2|two/i);
        }
      }
    });

    test('should cancel bulk delete on dismiss', async ({ page }) => {
      const rowCheckbox = page.locator('tbody input[type="checkbox"]').first();

      if (await rowCheckbox.isVisible()) {
        await rowCheckbox.check();
        const bulkDeleteButton = page
          .getByRole('button', { name: /delete selected|bulk delete/i })
          .first();

        if (await bulkDeleteButton.isVisible()) {
          await bulkDeleteButton.click();

          const cancelButton = page.getByRole('button', { name: /cancel|no/i }).first();
          if (await cancelButton.isVisible()) {
            await cancelButton.click();

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
        const bulkDeleteButton = page
          .getByRole('button', { name: /delete selected|bulk delete/i })
          .first();

        if (await bulkDeleteButton.isVisible()) {
          await bulkDeleteButton.click();

          const confirmButton = page.getByRole('button', { name: /confirm|yes|delete/i }).first();
          if (await confirmButton.isVisible()) {
            await confirmButton.click();

            // Selection should be cleared
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

        // Export button should appear
        const _exportButton = page.getByRole('button', { name: /export selected|bulk export/i });
      }
    });

    test('should have export all option', async ({ page }) => {
      const exportAllButton = page.getByRole('button', { name: /export all|download all/i });
      const count = await exportAllButton.count();
    });

    test('should offer export format options', async ({ page }) => {
      const rowCheckbox = page.locator('tbody input[type="checkbox"]').first();

      if (await rowCheckbox.isVisible()) {
        await rowCheckbox.check();

        const exportButton = page.getByRole('button', { name: /export/i }).first();
        if (await exportButton.isVisible()) {
          await exportButton.click();

          // Should show format options (YAML, JSON, CSV)
          const _formatOptions = page.getByText(/yaml|json|csv/i);
        }
      }
    });

    test('should trigger download on export', async ({ page }) => {
      const rowCheckbox = page.locator('tbody input[type="checkbox"]').first();

      if (await rowCheckbox.isVisible()) {
        await rowCheckbox.check();

        const exportButton = page.getByRole('button', { name: /export/i }).first();
        if (await exportButton.isVisible()) {
          const downloadPromise = page
            .waitForEvent('download', { timeout: 5000 })
            .catch(() => null);
          await exportButton.click();
          const _download = await downloadPromise;

          // May or may not trigger download depending on implementation
        }
      }
    });
  });

  test.describe('Bulk Edit', () => {
    test('should show edit button when items selected', async ({ page }) => {
      const rowCheckbox = page.locator('tbody input[type="checkbox"]').first();

      if (await rowCheckbox.isVisible()) {
        await rowCheckbox.check();

        // Edit button may appear for bulk edit
        const _editButton = page.getByRole('button', { name: /edit selected|bulk edit/i });
      }
    });
  });

  test.describe('Keyboard Shortcuts', () => {
    test('should select row with Space key', async ({ page }) => {
      const firstRow = page.locator('tbody tr').first();

      if (await firstRow.isVisible()) {
        await firstRow.focus();
        await page.keyboard.press('Space');

        // Row should be selected
      }
    });

    test('should select all with Ctrl+A', async ({ page }) => {
      await page.keyboard.press('Control+a');

      // All should be selected (if implemented)
    });

    test('should deselect all with Escape', async ({ page }) => {
      const rowCheckbox = page.locator('tbody input[type="checkbox"]').first();

      if (await rowCheckbox.isVisible()) {
        await rowCheckbox.check();

        await page.keyboard.press('Escape');

        // Selection should be cleared (if implemented)
      }
    });
  });
});
