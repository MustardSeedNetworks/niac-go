import { expect, test } from '@playwright/test';
import { ATTACHMENT, NETWORK_YAML, SIM_INTERFACE } from './fixtures/three-way-network';

test('device switching and wizard navigation preserve edits until explicitly resolved', async ({
  page,
  request,
}) => {
  const csrf = await request.get('/api/v1/csrf-token');
  const { token } = (await csrf.json()) as { token: string };
  const headers = { 'X-Csrf-Token': token };
  const started = await request.post('/api/v1/simulation', {
    headers,
    data: {
      sessionId: 'unsaved-acceptance',
      interface: SIM_INTERFACE,
      configData: NETWORK_YAML,
      attachment: ATTACHMENT.name,
      attachmentMode: ATTACHMENT.mode,
      accessVlan: ATTACHMENT.accessVlan,
    },
  });
  expect(started.ok(), await started.text()).toBe(true);
  await page.goto('/devices');
  await page.getByTestId('device-select-e2e-rtr-01').click();
  const editor = page.getByRole('textbox', { name: 'YAML editor', exact: true });
  await expect(editor).toContainText('name: e2e-rtr-01');
  const edited = `${await editor.innerText()}\n# preserved router edit`;
  await editor.fill(edited);
  await page.getByTestId('device-select-e2e-sw-01').click();
  await expect(page.getByRole('dialog', { name: 'Unsaved changes' })).toBeVisible();
  await page.getByTestId('unsaved-cancel').click();
  await expect(editor).toContainText('preserved router edit');
  await page.getByTestId('device-select-e2e-sw-01').click();
  await page.getByTestId('unsaved-save').click();
  await expect(editor).toContainText('name: e2e-sw-01');
  const savedConfig = await request.get('/api/v1/config');
  expect((await savedConfig.json()).content).toContain('preserved router edit');

  await page.goto('/new-simulation');
  await expect(page.getByTestId('wizard-interface-select')).toBeEnabled();
  await page.getByTestId('wizard-interface-select').selectOption({ index: 1 });
  await page.getByTestId('wizard-start-empty').click();
  const created = page.waitForResponse(
    (response) =>
      response.url().endsWith('/library/drafts') && response.request().method() === 'POST',
  );
  await page.getByTestId('wizard-next-button').click();
  const draft = (await (await created).json()) as { name: string };
  await page.getByRole('tab', { name: 'YAML', exact: true }).click();
  const draftEditor = page.getByTestId('wizard-step-panel').getByRole('textbox');
  await draftEditor.fill('devices: []\n# preserved wizard edit');
  await page.getByTestId('sidebar-desktop').getByTestId('nav-item-root').click();
  await expect(page.getByRole('dialog', { name: 'Unsaved changes' })).toBeVisible();
  await page.getByTestId('unsaved-cancel').click();
  await expect(draftEditor).toContainText('preserved wizard edit');
  await page.getByTestId('sidebar-desktop').getByTestId('nav-item-root').click();
  await page.getByTestId('unsaved-save').click();
  await expect(page).toHaveURL(/\/$/);
  const savedDraft = await request.get(`/api/v1/library/drafts/${encodeURIComponent(draft.name)}`);
  expect((await savedDraft.json()).content).toContain('preserved wizard edit');
  await request.delete('/api/v1/simulation?sessionId=unsaved-acceptance', { headers });
});
