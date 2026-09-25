import { expect, test } from '@playwright/test';
import { withIsolatedDaemon } from './isolated-daemon';

test('an idle daemon is neutral rather than all-clear', async ({ page, request }) => {
  await withIsolatedDaemon(request, async (baseURL) => {
    const response = await request.get(`${baseURL}/api/v1/simulation`);
    expect(response.ok()).toBe(true);
    expect(await response.json()).toMatchObject({ running: false });
    await page.goto(`${baseURL}/runtime`);
    const rollup = page.getByTestId('status-rollup');
    await expect(rollup).toHaveAttribute('data-state', 'idle');
    await expect(rollup).toContainText('No simulation is running');
    await expect(rollup).not.toContainText('All clear');
  });
});

test('an idle UI asks nothing of the stack and reads the idle config calmly', async ({
  page,
  request,
}) => {
  await withIsolatedDaemon(request, async (baseURL) => {
    // Idle is one condition, so the config read reports it like the fault
    // catalogue does (niac-go#2192: it was a 400 beside the others' 503).
    const errors = await request.get(`${baseURL}/api/v1/errors`);
    const config = await request.get(`${baseURL}/api/v1/config`);
    expect(errors.status()).toBe(503);
    expect(config.status()).toBe(errors.status());

    const catalogueReads: string[] = [];
    page.on('request', (req) => {
      if (new URL(req.url()).pathname === '/api/v1/errors') catalogueReads.push(req.url());
    });
    await page.goto(`${baseURL}/devices`);
    await expect(page.getByTestId('config-editor-idle-empty')).toBeVisible();
    await expect(page.getByRole('alert')).toHaveCount(0);
    // An enabled poll fires on mount, before the config answer that renders
    // the prompt above, so by now it would have been sent.
    expect(catalogueReads).toEqual([]);
  });
});
