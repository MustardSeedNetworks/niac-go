import { expect, test } from '@playwright/test';

test.describe('daemon-served GUI', () => {
  test('serves the SPA and reads live API data from the Go daemon', async ({ page }) => {
    const apiResponses: Array<{ url: string; status: number }> = [];
    page.on('response', (response) => {
      if (response.url().includes('/api/v1/')) {
        apiResponses.push({ url: response.url(), status: response.status() });
      }
    });

    const version = await page.request.get('/api/v1/version');
    expect(version.ok()).toBe(true);

    await page.goto('/');

    await expect(page).toHaveTitle(/NIAC/i);
    await expect(page.getByRole('navigation')).toBeVisible();
    await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();

    await expect
      .poll(() =>
        apiResponses.some(
          (response) => response.url.includes('/api/v1/version') && response.status === 200,
        ),
      )
      .toBe(true);
  });

  test('accepts state-changing API calls with the browser CSRF token', async ({ request }) => {
    const csrf = await request.get('/api/v1/csrf-token');
    expect(csrf.ok()).toBe(true);
    const { token } = (await csrf.json()) as { token: string };
    expect(token).toBeTruthy();

    const missingToken = await request.put('/api/v1/alerts', {
      data: { packets_threshold: 2500, webhook_url: '' },
    });
    expect(missingToken.status()).toBe(403);

    const update = await request.put('/api/v1/alerts', {
      headers: { 'X-Csrf-Token': token },
      data: { packets_threshold: 2500, webhook_url: '' },
    });
    expect(update.ok()).toBe(true);

    const updated = (await update.json()) as { packets_threshold: number };
    expect(updated.packets_threshold).toBe(2500);
  });
});
