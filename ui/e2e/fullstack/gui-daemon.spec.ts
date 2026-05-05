import { expect, test } from '@playwright/test';

const simulationInterface = process.env.E2E_SIM_INTERFACE;
const simulationConfig = `
devices:
  - name: sim-router-01
    type: router
    mac: "02:00:00:00:00:01"
    ip: "192.0.2.10"
    arp:
      enabled: true
    icmp:
      enabled: true
  - name: sim-switch-01
    type: switch
    mac: "02:00:00:00:00:02"
    ip: "192.0.2.20"
    lldp:
      enabled: true
`;

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

  test('shows seeded devices from a real simulation when an interface is provided', async ({
    page,
    request,
  }) => {
    test.skip(
      !simulationInterface,
      'Set E2E_SIM_INTERFACE to run the privileged real-simulation GUI smoke.',
    );

    const csrf = await request.get('/api/v1/csrf-token');
    expect(csrf.ok()).toBe(true);
    const { token } = (await csrf.json()) as { token: string };
    const headers = { 'X-Csrf-Token': token };

    const start = await request.post('/api/v1/simulation', {
      headers,
      data: {
        interface: simulationInterface,
        config_data: simulationConfig,
      },
    });
    expect(start.ok()).toBe(true);

    try {
      await expect
        .poll(async () => {
          const status = await request.get('/api/v1/simulation');
          const body = (await status.json()) as { running?: boolean; device_count?: number };
          return body.running === true && body.device_count === 2;
        })
        .toBe(true);

      await page.goto('/devices');
      await expect(page.getByText('sim-router-01')).toBeVisible();
      await expect(page.getByText('sim-switch-01')).toBeVisible();
    } finally {
      await request.delete('/api/v1/simulation', { headers });
    }
  });
});
