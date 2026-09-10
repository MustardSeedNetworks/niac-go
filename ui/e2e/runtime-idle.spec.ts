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
