import { expect, test } from '@playwright/test';
import { withIsolatedDaemon } from './isolated-daemon';

test('applies and clears device faults through the real daemon', async ({ page, request }) => {
  await withIsolatedDaemon(request, async (baseURL) => {
    const csrf = await request.get(`${baseURL}/api/v1/csrf-token`);
    expect(csrf.ok()).toBe(true);
    const { token } = (await csrf.json()) as { token: string };
    const start = await request.post(`${baseURL}/api/v1/simulation`, {
      headers: { 'X-Csrf-Token': token },
      data: {
        interface: 'e2e-dry-run0',
        configData: `devices:
  - name: portal-host
    type: server
    mac: "02:00:00:00:cc:01"
    ips: ["192.0.2.10"]
    http:
      enabled: true
`,
      },
    });
    expect(start.ok(), await start.text()).toBe(true);
    await page.goto(`${baseURL}/traffic`);
    const panel = page.getByTestId('device-fault-panel');
    await panel.getByLabel('Device fault target').selectOption('portal-host');
    await panel.getByLabel('Device fault', { exact: true }).selectOption('Captive Portal');
    await expect(panel.getByRole('spinbutton')).toHaveCount(0);
    await panel.getByTestId('apply-device-fault').click();
    await expect(panel.getByText('Armed', { exact: true })).toBeVisible();
    const active = await request.get(`${baseURL}/api/v1/errors`);
    expect(await active.json()).toMatchObject({
      active_device_errors: { 'portal-host': { 'Captive Portal': { value: 1 } } },
    });
    await panel.getByRole('button', { name: 'Clear Captive Portal on portal-host' }).click();
    await expect(panel.getByText('Armed', { exact: true })).toHaveCount(0);
    await panel.getByLabel('Device fault', { exact: true }).selectOption('Latency');
    await panel.getByRole('spinbutton').fill('60000');
    await panel.getByTestId('apply-device-fault').click();
    await expect(panel.getByText('60000 ms', { exact: true })).toBeVisible();
    const delayed = await request.get(`${baseURL}/api/v1/errors`);
    expect(await delayed.json()).toMatchObject({
      active_device_errors: { 'portal-host': { Latency: { value: 60000 } } },
    });
  });
});
