import { expect, type Page } from '@playwright/test';
import { openMobileSidebar } from './sidebar';

export async function checkLongHistoryLayout(page: Page) {
  // The recovery generation name from the failing CI history response.
  const configName = '_running.e2e-three-way.dd51b35c1e85d1c7718579cc6c5ff6f5.inline.yaml';
  await page.route('**/api/v1/history*', (route) =>
    route.fulfill({
      json: [
        {
          id: 1,
          started_at: '2026-09-10T07:37:13Z',
          duration: 42861669,
          interface: 'e2e-dry-run0',
          config_name: configName,
          device_count: 2,
          packets_sent: 0,
          packets_received: 0,
          errors: 0,
        },
      ],
    }),
  );

  for (const path of ['/', '/runtime']) {
    await page.goto(path);
    await expect(page.getByText(configName, { exact: true })).toBeVisible();
    const layout = await page.evaluate(() => ({
      width: document.documentElement.clientWidth,
      content: document.documentElement.scrollWidth,
    }));
    expect.soft(layout.content, `${path} with a long run name`).toBeLessThanOrEqual(layout.width);
    const nav = await openMobileSidebar(page);
    await nav.getByTestId('nav-item-devices').click();
    await expect(page).toHaveURL(/\/devices$/);
  }
}
