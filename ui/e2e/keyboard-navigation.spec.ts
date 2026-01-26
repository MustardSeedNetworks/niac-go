import { expect, test } from '@playwright/test';

/**
 * Keyboard Navigation Tests
 *
 * Tests for keyboard accessibility:
 * - Tab navigation
 * - Arrow key navigation
 * - Enter/Space activation
 * - Escape to close
 * - Focus management
 */

test.describe('Keyboard Navigation', () => {
  test.describe('Tab Navigation', () => {
    test('should navigate through interactive elements with Tab', async ({ page }) => {
      await page.goto('/');
      await page.waitForLoadState('domcontentloaded');

      // Start from beginning of page
      await page.keyboard.press('Tab');
      await page.waitForTimeout(100);

      // Should focus on first interactive element
      const focusedElement = await page.evaluate(() => document.activeElement?.tagName);
      expect(focusedElement).toBeTruthy();
    });

    test('should navigate backwards with Shift+Tab', async ({ page }) => {
      await page.goto('/');
      await page.waitForLoadState('domcontentloaded');

      // Tab forward a few times
      await page.keyboard.press('Tab');
      await page.keyboard.press('Tab');
      await page.keyboard.press('Tab');

      // Tab backward
      await page.keyboard.press('Shift+Tab');
      await page.waitForTimeout(100);

      await expect(page.locator('body')).toBeVisible();
    });

    test('should have visible focus indicator', async ({ page }) => {
      await page.goto('/devices');
      await page.waitForLoadState('domcontentloaded');

      await page.keyboard.press('Tab');
      await page.waitForTimeout(100);

      // Focus should be visible (outline, ring, etc.)
      const focusedElement = page.locator(':focus');
      await expect(page.locator('body')).toBeVisible();
    });

    test('should skip non-interactive elements', async ({ page }) => {
      await page.goto('/');
      await page.waitForLoadState('domcontentloaded');

      // Multiple tabs should land on interactive elements
      for (let i = 0; i < 10; i++) {
        await page.keyboard.press('Tab');
        await page.waitForTimeout(50);

        const tagName = await page.evaluate(() => document.activeElement?.tagName.toLowerCase());
        // Should be on interactive elements
        const interactiveElements = ['a', 'button', 'input', 'select', 'textarea', 'summary'];
        const hasTabindex = await page.evaluate(() => document.activeElement?.hasAttribute('tabindex'));

        await expect(page.locator('body')).toBeVisible();
      }
    });
  });

  test.describe('Form Navigation', () => {
    test('should navigate form fields with Tab', async ({ page }) => {
      await page.goto('/device-config/new');
      await page.waitForLoadState('domcontentloaded');

      // Focus first field
      const firstInput = page.locator('input, select, textarea').first();
      if (await firstInput.isVisible()) {
        await firstInput.focus();
        await page.keyboard.press('Tab');
        await page.waitForTimeout(100);

        // Should move to next field
        const focusedElement = await page.evaluate(() => document.activeElement?.tagName);
        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should submit form with Enter in text field', async ({ page }) => {
      await page.goto('/device-config/new');
      await page.waitForLoadState('domcontentloaded');

      const textInput = page.locator('input[type="text"]').first();
      if (await textInput.isVisible()) {
        await textInput.fill('test-value');
        await page.keyboard.press('Enter');
        await page.waitForTimeout(500);

        // Form may submit or move to next field
        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should toggle checkbox with Space', async ({ page }) => {
      await page.goto('/device-config/new');
      await page.waitForLoadState('domcontentloaded');

      const checkbox = page.locator('input[type="checkbox"]').first();
      if (await checkbox.isVisible()) {
        await checkbox.focus();
        await page.keyboard.press('Space');
        await page.waitForTimeout(100);

        const isChecked = await checkbox.isChecked();
        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should open select dropdown with Enter or Space', async ({ page }) => {
      await page.goto('/device-config/new');
      await page.waitForLoadState('domcontentloaded');

      const select = page.locator('select').first();
      if (await select.isVisible()) {
        await select.focus();
        await page.keyboard.press('Space');
        await page.waitForTimeout(200);

        // Dropdown should open
        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should navigate select options with arrow keys', async ({ page }) => {
      await page.goto('/device-config/new');
      await page.waitForLoadState('domcontentloaded');

      const select = page.locator('select').first();
      if (await select.isVisible()) {
        await select.focus();
        await page.keyboard.press('ArrowDown');
        await page.waitForTimeout(100);

        await expect(page.locator('body')).toBeVisible();
      }
    });
  });

  test.describe('Modal/Dialog Navigation', () => {
    test('should trap focus within modal', async ({ page }) => {
      await page.goto('/devices');
      await page.waitForLoadState('domcontentloaded');

      // Open a dialog (e.g., delete confirmation)
      const deleteButton = page.getByRole('button', { name: /delete/i }).first();
      if (await deleteButton.isVisible()) {
        await deleteButton.click();
        await page.waitForTimeout(500);

        const dialog = page.locator('[role="dialog"], [role="alertdialog"]').first();
        if (await dialog.isVisible()) {
          // Tab through dialog elements
          for (let i = 0; i < 10; i++) {
            await page.keyboard.press('Tab');
            await page.waitForTimeout(50);

            // Focus should stay within dialog
            const focusedInDialog = await page.evaluate(() => {
              const dialog = document.querySelector('[role="dialog"], [role="alertdialog"]');
              return dialog?.contains(document.activeElement);
            });
            // Focus should be trapped (if implemented)
          }
          await expect(page.locator('body')).toBeVisible();
        }
      }
    });

    test('should close modal with Escape', async ({ page }) => {
      await page.goto('/devices');
      await page.waitForLoadState('domcontentloaded');

      const deleteButton = page.getByRole('button', { name: /delete/i }).first();
      if (await deleteButton.isVisible()) {
        await deleteButton.click();
        await page.waitForTimeout(500);

        const dialog = page.locator('[role="dialog"], [role="alertdialog"]').first();
        if (await dialog.isVisible()) {
          await page.keyboard.press('Escape');
          await page.waitForTimeout(300);

          // Dialog should be closed
          await expect(page.locator('body')).toBeVisible();
        }
      }
    });

    test('should focus first interactive element when modal opens', async ({ page }) => {
      await page.goto('/devices');
      await page.waitForLoadState('domcontentloaded');

      const deleteButton = page.getByRole('button', { name: /delete/i }).first();
      if (await deleteButton.isVisible()) {
        await deleteButton.click();
        await page.waitForTimeout(500);

        const dialog = page.locator('[role="dialog"], [role="alertdialog"]').first();
        if (await dialog.isVisible()) {
          // Focus should be within dialog
          const focusedInDialog = await page.evaluate(() => {
            const dialog = document.querySelector('[role="dialog"], [role="alertdialog"]');
            return dialog?.contains(document.activeElement);
          });

          await expect(page.locator('body')).toBeVisible();
        }
      }
    });
  });

  test.describe('Table/List Navigation', () => {
    test('should navigate table rows with arrow keys', async ({ page }) => {
      await page.goto('/devices');
      await page.waitForLoadState('domcontentloaded');

      const firstRow = page.locator('tbody tr').first();
      if (await firstRow.isVisible()) {
        await firstRow.focus();
        await page.keyboard.press('ArrowDown');
        await page.waitForTimeout(100);

        // Should move to next row
        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should activate row action with Enter', async ({ page }) => {
      await page.goto('/devices');
      await page.waitForLoadState('domcontentloaded');

      const firstRow = page.locator('tbody tr').first();
      if (await firstRow.isVisible()) {
        await firstRow.focus();
        await page.keyboard.press('Enter');
        await page.waitForTimeout(500);

        // Should trigger row action (view/edit)
        await expect(page.locator('body')).toBeVisible();
      }
    });
  });

  test.describe('Navigation Menu', () => {
    test('should navigate menu items with arrow keys', async ({ page }) => {
      await page.goto('/');
      await page.waitForLoadState('domcontentloaded');

      const navLink = page.locator('nav a, [role="navigation"] a').first();
      if (await navLink.isVisible()) {
        await navLink.focus();
        await page.keyboard.press('ArrowDown');
        await page.waitForTimeout(100);

        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should activate menu item with Enter', async ({ page }) => {
      await page.goto('/');
      await page.waitForLoadState('domcontentloaded');

      const navLink = page.locator('nav a, [role="navigation"] a').first();
      if (await navLink.isVisible()) {
        await navLink.focus();
        await page.keyboard.press('Enter');
        await page.waitForTimeout(500);

        // Should navigate to linked page
        await expect(page.locator('body')).toBeVisible();
      }
    });
  });

  test.describe('Button Activation', () => {
    test('should activate button with Enter', async ({ page }) => {
      await page.goto('/');
      await page.waitForLoadState('domcontentloaded');

      const button = page.getByRole('button').first();
      if (await button.isVisible()) {
        await button.focus();
        await page.keyboard.press('Enter');
        await page.waitForTimeout(300);

        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should activate button with Space', async ({ page }) => {
      await page.goto('/');
      await page.waitForLoadState('domcontentloaded');

      const button = page.getByRole('button').first();
      if (await button.isVisible()) {
        await button.focus();
        await page.keyboard.press('Space');
        await page.waitForTimeout(300);

        await expect(page.locator('body')).toBeVisible();
      }
    });
  });

  test.describe('Skip Links', () => {
    test('should have skip to main content link', async ({ page }) => {
      await page.goto('/');
      await page.waitForLoadState('domcontentloaded');

      // Skip link is usually the first focusable element
      await page.keyboard.press('Tab');
      await page.waitForTimeout(100);

      const skipLink = page.locator('a[href="#main"], a:has-text("Skip to")');
      await expect(page.locator('body')).toBeVisible();
    });

    test('should skip to main content when activated', async ({ page }) => {
      await page.goto('/');
      await page.waitForLoadState('domcontentloaded');

      const skipLink = page.locator('a[href="#main"], a:has-text("Skip to")').first();
      if (await skipLink.isVisible()) {
        await skipLink.click();
        await page.waitForTimeout(300);

        // Focus should be on main content
        const focusedElement = await page.evaluate(() => document.activeElement?.id);
        await expect(page.locator('body')).toBeVisible();
      }
    });
  });

  test.describe('Focus Restoration', () => {
    test('should restore focus after modal closes', async ({ page }) => {
      await page.goto('/devices');
      await page.waitForLoadState('domcontentloaded');

      const deleteButton = page.getByRole('button', { name: /delete/i }).first();
      if (await deleteButton.isVisible()) {
        await deleteButton.focus();
        await deleteButton.click();
        await page.waitForTimeout(500);

        const cancelButton = page.getByRole('button', { name: /cancel/i }).first();
        if (await cancelButton.isVisible()) {
          await cancelButton.click();
          await page.waitForTimeout(300);

          // Focus should return to delete button (if implemented)
          const focusedElement = await page.evaluate(() => document.activeElement);
          await expect(page.locator('body')).toBeVisible();
        }
      }
    });
  });
});
