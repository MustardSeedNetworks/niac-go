import { expect, test } from '@playwright/test';
import { withIsolatedDaemon } from './isolated-daemon';

test('applies /0 and /32 masks and clears them independently', async ({
  page,
  request,
}, testInfo) => {
  await withIsolatedDaemon(request, async (baseURL) => {
    const csrf = await request.get(`${baseURL}/api/v1/csrf-token`);
    expect(csrf.ok()).toBe(true);
    const { token } = (await csrf.json()) as { token: string };
    const start = await request.post(`${baseURL}/api/v1/simulation`, {
      headers: { 'X-Csrf-Token': token },
      data: {
        interface: 'e2e-dry-run0',
        configData: `devices:
  - name: mask-host
    type: server
    mac: "02:00:00:00:ed:01"
    ips: [192.0.2.1]
    interfaces: [{name: Management, address: 192.0.2.1/24}]
    snmp_agent: {community: public}
`,
      },
    });
    expect(start.ok(), await start.text()).toBe(true);
    await page.goto(`${baseURL}/traffic`);
    const panel = page.getByTestId('interface-fault-panel');
    await panel.getByLabel('Device', { exact: true }).selectOption('mask-host');
    await panel.getByLabel('Interface', { exact: true }).selectOption('Management');
    await panel.getByLabel('Error Type').selectOption('High Utilization');
    await panel.getByTestId('apply-interface-fault').click();
    await expect(panel.getByText('50%', { exact: true })).toBeVisible();
    await panel.getByLabel('Error Type').selectOption('Bad Subnet Mask');
    const prefix = panel.getByTestId('interface-fault-prefix');
    const apply = panel.getByTestId('apply-interface-fault');
    await expect(apply).toBeDisabled();
    await expect(prefix).toHaveAccessibleName('Subnet mask length');
    for (const invalid of ['-1', '33', '1.5', '']) {
      await prefix.fill(invalid);
      await expect(apply).toBeDisabled();
    }
    for (const prefixBits of [0, 32]) {
      await prefix.fill(String(prefixBits));
      await apply.click();
      const row = panel.getByRole('row').filter({ hasText: 'Bad Subnet Mask' });
      await expect(row.getByText(`/${prefixBits}`, { exact: true })).toBeVisible();
      const active = await request.get(`${baseURL}/api/v1/errors`);
      expect(active.ok()).toBe(true);
      expect((await active.json()).active_errors).toEqual({
        'mask-host': {
          Management: { 'High Utilization': { value: 50 }, 'Bad Subnet Mask': { prefixBits } },
        },
      });
      if (prefixBits === 0) {
        await page.screenshot({
          path: testInfo.outputPath('interface-mask-active.png'),
          fullPage: true,
        });
        const viewport = page.viewportSize();
        await page.setViewportSize({ width: 320, height: 800 });
        for (const theme of ['light', 'dark']) {
          await page.evaluate(
            (dark) => document.documentElement.classList.toggle('dark', dark),
            theme === 'dark',
          );
          await expect(prefix).toBeVisible();
          const dimensions = await page.evaluate(() => ({
            width: document.documentElement.clientWidth,
            content: document.documentElement.scrollWidth,
          }));
          expect(dimensions.content).toBeLessThanOrEqual(dimensions.width + 1);
          await page.screenshot({
            path: testInfo.outputPath(`interface-mask-mobile-${theme}.png`),
            fullPage: true,
          });
        }
        if (viewport) await page.setViewportSize(viewport);
      }
      await row.getByRole('button', { name: 'Clear error on mask-host Management' }).click();
      await expect(row).toHaveCount(0);
      await expect(panel.getByText('50%', { exact: true })).toBeVisible();
      const cleared = await request.get(`${baseURL}/api/v1/errors`);
      expect(cleared.ok()).toBe(true);
      expect((await cleared.json()).active_errors).toEqual({
        'mask-host': { Management: { 'High Utilization': { value: 50 } } },
      });
    }
  });
});
