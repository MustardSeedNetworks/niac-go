import { expect, test } from '@playwright/test';

/**
 * Packet Inspector Page (/packets) E2E
 *
 * Covers the live-packet capture / inspection surface:
 * - Page renders the "Packets" heading
 * - Page lands on /packets route
 */

test.describe('Packet Inspector Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/packets');
    await page.waitForLoadState('domcontentloaded');
  });

  test('should render the Packets heading', async ({ page }) => {
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 10000,
    });
  });

  test('should land on the /packets route', async ({ page }) => {
    await expect(page).toHaveURL(/\/packets$/);
  });
});

/**
 * U7: stopping a standalone capture ends the only source of live frames on
 * the page, so it must ask first. Both cases stub the capture as running —
 * the E2E daemon has no interface it may sniff.
 */
test.describe('Packet Inspector Page — stop capture confirmation', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/api/v1/capture', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({ json: { running: true, interface: 'eth0', filter: '' } });
        return;
      }
      await route.fulfill({ json: { status: 'stopped' } });
    });
    await page.goto('/packets');
    await page.waitForLoadState('domcontentloaded');
  });

  test('asks before stopping, and backing out leaves the capture running', async ({ page }) => {
    const stops: string[] = [];
    page.on('request', (request) => {
      if (request.url().includes('/api/v1/capture') && request.method() === 'DELETE') {
        stops.push(request.url());
      }
    });

    await page.getByRole('button', { name: 'Stop capture' }).click();

    const dialog = page.getByRole('dialog', { name: 'Stop the capture?' });
    await expect(dialog).toBeVisible();
    expect(stops).toHaveLength(0);

    await dialog.getByRole('button', { name: 'Cancel' }).click();
    await expect(dialog).toBeHidden();
    expect(stops).toHaveLength(0);
    await expect(page.getByRole('button', { name: 'Stop capture' })).toBeVisible();
  });

  test('confirming sends the stop', async ({ page }) => {
    const stopped = page.waitForRequest(
      (request) => request.url().includes('/api/v1/capture') && request.method() === 'DELETE',
    );

    await page.getByRole('button', { name: 'Stop capture' }).click();
    const dialog = page.getByRole('dialog', { name: 'Stop the capture?' });
    await dialog.getByRole('button', { name: 'Stop capture' }).click();

    await stopped;
    await expect(dialog).toBeHidden();
  });
});
