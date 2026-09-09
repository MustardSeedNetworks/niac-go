import { expect, test } from '@playwright/test';

test('shares device polling and clears previous-session data before the next response', async ({
  page,
  request,
}) => {
  const csrf = await request.get('/api/v1/csrf-token');
  expect(csrf.ok()).toBe(true);
  const { token } = (await csrf.json()) as { token: string };
  const headers = { 'X-Csrf-Token': token };
  const started: string[] = [];
  const release = Promise.withResolvers<void>();

  try {
    for (const [index, sessionId] of ['query-a', 'query-b'].entries()) {
      const start = await request.post('/api/v1/simulation', {
        headers,
        data: {
          sessionId,
          interface: `e2e-${sessionId}`,
          attachmentMode: 'access',
          accessVlan: 200 + index,
          configData: `devices:\n  - name: ${sessionId}-router\n    type: router\n    mac: "02:00:00:00:00:0${index + 1}"\n    ips: ["192.0.2.${index + 10}"]\n    icmp:\n      enabled: true\n`,
        },
      });
      expect(start.ok(), await start.text()).toBe(true);
      started.push(sessionId);
    }
    const selected = await request.put('/api/v1/simulation', {
      headers,
      data: { sessionId: 'query-a' },
    });
    expect(selected.ok()).toBe(true);

    const reads: string[] = [];
    page.on('request', (incoming) => {
      const path = new URL(incoming.url()).pathname;
      if (/\/sessions\/query-[ab]\/devices$/.test(path)) reads.push(path);
    });
    await page.goto('/devices');
    await expect(page.getByTestId('device-select-query-a-router')).toBeVisible();
    const firstPath = '/api/v1/sessions/query-a/devices';
    expect(reads).toEqual([firstPath]);

    // Observe the actual polling interval without freezing query notifications.
    const response = await page.waitForResponse(`**${firstPath}`);
    expect(response.ok()).toBe(true);
    expect(reads).toEqual([firstPath, firstPath]);

    const nav = page.getByTestId('sidebar-desktop');
    await nav.getByTestId('nav-item-topology').click();
    await expect(page).toHaveURL(/\/topology$/);
    await nav.getByTestId('nav-item-devices').click();
    await expect(page.getByTestId('device-select-query-a-router')).toBeVisible();
    expect(reads).toEqual([firstPath, firstPath]);

    const pending = Promise.withResolvers<void>();
    await page.route('**/api/v1/sessions/query-b/devices', async (route) => {
      pending.resolve();
      await release.promise;
      // Delay transport only: the daemon still supplies the entire response.
      await route.continue();
    });
    await page.getByTestId('session-switcher-select').selectOption('query-b');
    await pending.promise;
    await expect(page.getByTestId('device-select-query-a-router')).toHaveCount(0);
    await expect(page.getByTestId('device-select-query-b-router')).toHaveCount(0);
    release.resolve();
    await expect(page.getByTestId('device-select-query-b-router')).toBeVisible();
    await expect(page.getByTestId('device-select-query-a-router')).toHaveCount(0);
    expect(reads).toEqual([firstPath, firstPath, '/api/v1/sessions/query-b/devices']);
  } finally {
    release.resolve();
    for (const sessionId of started) {
      const stop = await request.delete(`/api/v1/simulation?sessionId=${sessionId}`, { headers });
      expect(stop.ok(), await stop.text()).toBe(true);
    }
  }
});
