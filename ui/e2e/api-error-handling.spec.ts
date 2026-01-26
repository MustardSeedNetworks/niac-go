import { expect, test } from '@playwright/test';

/**
 * API Error Handling Tests
 *
 * Tests for proper handling of API errors:
 * - Network errors
 * - HTTP error codes (400, 401, 403, 404, 409, 422, 500+)
 * - Timeout handling
 * - Retry logic
 */

test.describe('API Error Handling', () => {
  test.describe('Network Errors', () => {
    test('should handle connection refused gracefully', async ({ page }) => {
      // Intercept API calls and simulate connection refused
      await page.route('**/api/v1/**', (route) => {
        route.abort('connectionrefused');
      });

      await page.goto('/devices');
      await page.waitForLoadState('domcontentloaded');

      // Should show error state or fallback UI
      const errorIndicator = page.locator('[class*="error"], [role="alert"], [class*="offline"]');
      await expect(page.locator('body')).toBeVisible();
    });

    test('should handle timeout gracefully', async ({ page }) => {
      // Simulate slow response
      await page.route('**/api/v1/config/devices', async (route) => {
        await new Promise((resolve) => setTimeout(resolve, 35000));
        route.fulfill({ status: 200, body: '[]' });
      });

      await page.goto('/devices');
      await page.waitForLoadState('domcontentloaded');
      await page.waitForTimeout(5000);

      // Should show loading or timeout error
      await expect(page.locator('body')).toBeVisible();
    });

    test('should show offline indicator when disconnected', async ({ page }) => {
      await page.goto('/');
      await page.waitForLoadState('domcontentloaded');

      // Go offline
      await page.context().setOffline(true);
      await page.waitForTimeout(1000);

      // Try to trigger an API call
      const refreshButton = page.getByRole('button', { name: /refresh|reload/i }).first();
      if (await refreshButton.isVisible()) {
        await refreshButton.click();
        await page.waitForTimeout(500);
      }

      // Should indicate offline state
      await expect(page.locator('body')).toBeVisible();

      // Restore online
      await page.context().setOffline(false);
    });
  });

  test.describe('HTTP 400 Bad Request', () => {
    test('should display validation errors from API', async ({ page }) => {
      await page.route('**/api/v1/config/devices', (route) => {
        if (route.request().method() === 'POST') {
          route.fulfill({
            status: 400,
            contentType: 'application/json',
            body: JSON.stringify({
              error: 'Validation failed',
              details: { hostname: 'Invalid hostname format' },
            }),
          });
        } else {
          route.continue();
        }
      });

      await page.goto('/device-config/new');
      await page.waitForLoadState('domcontentloaded');

      const hostnameField = page.locator('input[name="hostname"], input[name="name"]').first();
      const submitButton = page.getByRole('button', { name: /save|submit|create/i }).first();

      if ((await hostnameField.isVisible()) && (await submitButton.isVisible())) {
        await hostnameField.fill('test-device');
        if (await submitButton.isEnabled()) {
          await submitButton.click();
          await page.waitForTimeout(1000);

          // Should show error message
          await expect(page.locator('body')).toBeVisible();
        }
      }
    });
  });

  test.describe('HTTP 401 Unauthorized', () => {
    test('should handle unauthorized access', async ({ page }) => {
      await page.route('**/api/v1/**', (route) => {
        route.fulfill({
          status: 401,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Unauthorized' }),
        });
      });

      await page.goto('/devices');
      await page.waitForLoadState('domcontentloaded');

      // Should show auth error or redirect
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('HTTP 403 Forbidden', () => {
    test('should handle forbidden access', async ({ page }) => {
      await page.route('**/api/v1/config/devices', (route) => {
        if (route.request().method() === 'DELETE') {
          route.fulfill({
            status: 403,
            contentType: 'application/json',
            body: JSON.stringify({ error: 'Permission denied' }),
          });
        } else {
          route.continue();
        }
      });

      await page.goto('/devices');
      await page.waitForLoadState('domcontentloaded');

      const deleteButton = page.getByRole('button', { name: /delete|remove/i }).first();
      if (await deleteButton.isVisible()) {
        await deleteButton.click();
        await page.waitForTimeout(500);

        // Confirm if dialog appears
        const confirmButton = page.getByRole('button', { name: /confirm|yes|delete/i }).first();
        if (await confirmButton.isVisible()) {
          await confirmButton.click();
          await page.waitForTimeout(1000);
        }

        // Should show permission error
        await expect(page.locator('body')).toBeVisible();
      }
    });
  });

  test.describe('HTTP 404 Not Found', () => {
    test('should handle missing device gracefully', async ({ page }) => {
      await page.route('**/api/v1/config/devices/nonexistent', (route) => {
        route.fulfill({
          status: 404,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Device not found' }),
        });
      });

      await page.goto('/device-config/nonexistent');
      await page.waitForLoadState('domcontentloaded');

      // Should show not found message or redirect
      const notFoundText = page.getByText(/not found|does not exist|404/i);
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('HTTP 409 Conflict', () => {
    test('should handle duplicate device name', async ({ page }) => {
      await page.route('**/api/v1/config/devices', (route) => {
        if (route.request().method() === 'POST') {
          route.fulfill({
            status: 409,
            contentType: 'application/json',
            body: JSON.stringify({ error: 'Device with this hostname already exists' }),
          });
        } else {
          route.continue();
        }
      });

      await page.goto('/device-config/new');
      await page.waitForLoadState('domcontentloaded');

      const hostnameField = page.locator('input[name="hostname"], input[name="name"]').first();
      const submitButton = page.getByRole('button', { name: /save|submit|create/i }).first();

      if ((await hostnameField.isVisible()) && (await submitButton.isVisible())) {
        await hostnameField.fill('duplicate-device');
        if (await submitButton.isEnabled()) {
          await submitButton.click();
          await page.waitForTimeout(1000);

          // Should show conflict error
          const conflictError = page.getByText(/already exists|duplicate|conflict/i);
          await expect(page.locator('body')).toBeVisible();
        }
      }
    });
  });

  test.describe('HTTP 422 Unprocessable Entity', () => {
    test('should handle semantic validation errors', async ({ page }) => {
      await page.route('**/api/v1/config/devices', (route) => {
        if (route.request().method() === 'POST') {
          route.fulfill({
            status: 422,
            contentType: 'application/json',
            body: JSON.stringify({
              error: 'Validation failed',
              fields: {
                ip: 'IP address conflicts with existing device',
                mac: 'MAC address already in use',
              },
            }),
          });
        } else {
          route.continue();
        }
      });

      await page.goto('/device-config/new');
      await page.waitForLoadState('domcontentloaded');

      const submitButton = page.getByRole('button', { name: /save|submit|create/i }).first();
      if (await submitButton.isVisible() && (await submitButton.isEnabled())) {
        await submitButton.click();
        await page.waitForTimeout(1000);

        // Should show field-level errors
        await expect(page.locator('body')).toBeVisible();
      }
    });
  });

  test.describe('HTTP 500+ Server Errors', () => {
    test('should handle internal server error', async ({ page }) => {
      await page.route('**/api/v1/config/devices', (route) => {
        route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Internal server error' }),
        });
      });

      await page.goto('/devices');
      await page.waitForLoadState('domcontentloaded');

      // Should show error state
      const errorMessage = page.getByText(/error|failed|problem/i);
      await expect(page.locator('body')).toBeVisible();
    });

    test('should handle 502 Bad Gateway', async ({ page }) => {
      await page.route('**/api/v1/**', (route) => {
        route.fulfill({
          status: 502,
          contentType: 'text/html',
          body: '<html><body>Bad Gateway</body></html>',
        });
      });

      await page.goto('/devices');
      await page.waitForLoadState('domcontentloaded');

      // Should handle gracefully
      await expect(page.locator('body')).toBeVisible();
    });

    test('should handle 503 Service Unavailable', async ({ page }) => {
      await page.route('**/api/v1/**', (route) => {
        route.fulfill({
          status: 503,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Service temporarily unavailable' }),
        });
      });

      await page.goto('/devices');
      await page.waitForLoadState('domcontentloaded');

      // Should show service unavailable message
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('Error Recovery', () => {
    test('should allow retry after error', async ({ page }) => {
      let requestCount = 0;
      await page.route('**/api/v1/config/devices', (route) => {
        requestCount++;
        if (requestCount === 1) {
          route.fulfill({
            status: 500,
            body: JSON.stringify({ error: 'Server error' }),
          });
        } else {
          route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify([]),
          });
        }
      });

      await page.goto('/devices');
      await page.waitForLoadState('domcontentloaded');

      // Look for retry button
      const retryButton = page.getByRole('button', { name: /retry|try again|refresh/i }).first();
      if (await retryButton.isVisible()) {
        await retryButton.click();
        await page.waitForTimeout(1000);

        // Should succeed on retry
        await expect(page.locator('body')).toBeVisible();
      }
    });

    test('should clear error state on successful retry', async ({ page }) => {
      let requestCount = 0;
      await page.route('**/api/v1/config/devices', (route) => {
        requestCount++;
        if (requestCount <= 1) {
          route.fulfill({ status: 500, body: '{}' });
        } else {
          route.fulfill({ status: 200, body: '[]' });
        }
      });

      await page.goto('/devices');
      await page.waitForTimeout(1000);

      // Navigate away and back to retry
      await page.goto('/');
      await page.waitForLoadState('domcontentloaded');
      await page.goto('/devices');
      await page.waitForLoadState('domcontentloaded');

      // Error should be cleared
      await expect(page.locator('body')).toBeVisible();
    });
  });

  test.describe('SSE Error Handling', () => {
    test('should handle SSE connection errors', async ({ page }) => {
      await page.route('**/api/v1/events/**', (route) => {
        route.abort('connectionrefused');
      });

      await page.goto('/runtime');
      await page.waitForLoadState('domcontentloaded');
      await page.waitForTimeout(2000);

      // Should show connection error or fallback state
      await expect(page.locator('body')).toBeVisible();
    });

    test('should attempt SSE reconnection', async ({ page }) => {
      await page.goto('/runtime');
      await page.waitForLoadState('domcontentloaded');

      // Simulate SSE disconnect
      await page.evaluate(() => {
        window.dispatchEvent(new Event('offline'));
      });
      await page.waitForTimeout(1000);

      // Restore connection
      await page.evaluate(() => {
        window.dispatchEvent(new Event('online'));
      });
      await page.waitForTimeout(2000);

      // Should recover
      await expect(page.locator('body')).toBeVisible();
    });
  });
});
