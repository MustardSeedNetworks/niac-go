import { expect, test } from '@playwright/test';

/**
 * Debug Console Page (/debug) E2E
 *
 * Covers the live-log debug surface:
 * - Page renders the "Debug Console" heading
 * - A reconnect / connection-status control is present
 */

test.describe('Debug Console Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/debug');
    await page.waitForLoadState('domcontentloaded');
  });

  test('should render the Debug Console heading', async ({ page }) => {
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 10000,
    });
  });

  test('should land on the /debug route', async ({ page }) => {
    await expect(page).toHaveURL(/\/debug$/);
  });

  test('should expose a reconnect / connection status control', async ({ page }) => {
    const statusButton = page.getByTestId('debug-connection-status');
    await expect(statusButton).toBeVisible();
    await statusButton.focus();
    await page.keyboard.press('Shift+Tab');
    await page.keyboard.press('Tab');
    await expect(statusButton).toBeFocused();
    await expect(statusButton).toHaveAccessibleDescription(
      /Connected to log stream|Click to reconnect/,
    );
    await expect(
      page.getByRole('tooltip').filter({ hasText: /Connected to log stream|Click to reconnect/ }),
    ).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(statusButton).toBeFocused();
  });
});

// niac-go#2298: the first streamed line used to grow the log viewer and push
// the debug-level panel out from under a hovering pointer.
test('the first log line does not move the debug-level panel', async ({ page }) => {
  let release = () => {};
  const firstLine = new Promise<void>((resolve) => {
    release = resolve;
  });
  let streams = 0;
  let subscribed = () => {};
  const streamOpened = new Promise<void>((resolve) => {
    subscribed = resolve;
  });
  await page.route('**/api/v1/stream/logs', async (route) => {
    streams += 1;
    subscribed();
    // Later reconnects stay pending so only one line ever arrives.
    if (streams > 1) return;
    await firstLine;
    await route.fulfill({
      contentType: 'text/event-stream',
      body: 'data: {"level":"info","message":"first streamed line"}\n\n',
    });
  });
  await page.goto('/debug');
  await page.getByTestId('debug-level-toggle').click();
  const panel = page.getByTestId('debug-level-control');
  await expect(panel).toBeVisible();
  await streamOpened;
  const before = await panel.boundingBox();

  release();
  await expect(page.getByRole('log').getByText('first streamed line')).toBeVisible();
  expect(await panel.boundingBox()).toEqual(before);
});
