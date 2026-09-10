import { expect, test } from '@playwright/test';
import { withIsolatedDaemon } from './isolated-daemon';

test('applies and independently clears an interface IPv4 conflict', async ({ page, request }) => {
  await withIsolatedDaemon(request, async (baseURL) => {
    const csrf = await request.get(`${baseURL}/api/v1/csrf-token`);
    expect(csrf.ok()).toBe(true);
    const { token } = (await csrf.json()) as { token: string };
    const start = await request.post(`${baseURL}/api/v1/simulation`, {
      headers: { 'X-Csrf-Token': token },
      data: {
        interface: 'e2e-dry-run0',
        configData: `devices:
  - name: conflict-host
    mac: "02:00:00:00:ed:01"
    ips: [192.0.2.1]
    interfaces: [{name: Management}]
    snmp_agent: {community: public}
  - name: peer
    mac: "02:00:00:00:ed:02"
    ips: [192.0.2.20]
`,
      },
    });
    expect(start.ok(), await start.text()).toBe(true);
    await page.goto(`${baseURL}/traffic`);
    const panel = page.getByTestId('interface-fault-panel');
    await panel.getByLabel('Device', { exact: true }).selectOption('conflict-host');
    await panel.getByLabel('Interface', { exact: true }).selectOption('Management');
    await panel.getByLabel('Error Type').selectOption('High Utilization');
    await panel.getByTestId('apply-interface-fault').click();
    await expect(panel.getByText('50%', { exact: true })).toBeVisible();
    await panel.getByLabel('Error Type').selectOption('Duplicate IP');
    await expect(panel.getByTestId('apply-interface-fault')).toBeDisabled();
    await panel.getByLabel('Conflict IPv4 address').fill('192.0.2.20');
    await panel.getByTestId('apply-interface-fault').click();
    const conflictRow = panel.getByRole('row').filter({ hasText: 'Duplicate IP' });
    await expect(conflictRow.getByText('192.0.2.20', { exact: true })).toBeVisible();
    const active = await request.get(`${baseURL}/api/v1/errors`);
    expect(active.ok()).toBe(true);
    expect((await active.json()).active_errors).toEqual({
      'conflict-host': {
        Management: {
          'High Utilization': { value: 50 },
          'Duplicate IP': { address: '192.0.2.20' },
        },
      },
    });
    await conflictRow
      .getByRole('button', { name: 'Clear error on conflict-host Management' })
      .click();
    await expect(conflictRow).toHaveCount(0);
    await expect(panel.getByText('50%', { exact: true })).toBeVisible();
    const cleared = await request.get(`${baseURL}/api/v1/errors`);
    expect((await cleared.json()).active_errors).toEqual({
      'conflict-host': { Management: { 'High Utilization': { value: 50 } } },
    });
  });
});
