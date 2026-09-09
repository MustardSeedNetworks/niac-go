import { expect, test } from '@playwright/test';

test('an empty daemon guides device authors into a saved draft', async ({ page, request }) => {
  const inventory = await request.get('/api/v1/config/devices');
  expect((await inventory.json()).configurationLoaded).toBe(false);
  for (const path of ['/device-config', '/device-config/new']) {
    await page.goto(path);
    await expect(page.getByTestId('create-device-draft')).toBeVisible();
    await expect(page.getByTestId('device-editor-save')).toHaveCount(0);
  }
  await page.getByTestId('create-device-draft').click();
  const iface = page.getByTestId('wizard-interface-select');
  await expect(iface).toBeEnabled();
  await iface.selectOption({ index: 1 });
  await page.getByTestId('wizard-start-empty').click();
  const created = page.waitForResponse(
    (response) =>
      response.url().includes('/api/v1/library/drafts') && response.request().method() === 'POST',
  );
  await page.getByTestId('wizard-next-button').click();
  const response = await created;
  expect(response.ok(), await response.text()).toBe(true);
  await expect(page.getByTestId('wizard-step-devices')).toHaveAttribute('data-status', 'active');
  expect((await (await request.get('/api/v1/config/devices')).json()).configurationLoaded).toBe(
    false,
  );
});
