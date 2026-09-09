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
  try {
    await page.goto('/');
    const sidebar = page.getByTestId('sidebar-desktop');
    await sidebar.getByTestId('nav-item-devices').click();
    await page.getByTestId('device-select-e2e-rtr-01').click();
    const editor = page.getByRole('textbox', { name: 'YAML editor', exact: true });
    await expect(editor).toContainText('name: e2e-rtr-01');
    const edited = `${await editor.innerText()}\n# preserved router edit`;
    await editor.fill(edited);
    await page.getByTestId('device-select-e2e-sw-01').click();
    await expect(page.getByRole('dialog', { name: 'Unsaved changes' })).toBeVisible();
    await page.getByTestId('unsaved-cancel').click();
    await expect(editor).toContainText('preserved router edit');
    const invalidEdit = edited.replace('02:00:00:00:77:01', 'not-a-mac');
    expect(invalidEdit).not.toBe(edited);
    await editor.fill(invalidEdit);
    await page.getByTestId('device-select-e2e-sw-01').click();
    const refused = page.waitForResponse(
      (response) =>
        response.url().endsWith('/api/v1/config') && response.request().method() === 'PUT',
    );
    await page.getByTestId('unsaved-save').click();
    expect((await refused).status()).toBe(400);
    await expect(page.getByRole('dialog').getByRole('alert')).toBeVisible();
    await expect(editor).toContainText('not-a-mac');
    await page.getByTestId('unsaved-cancel').click();
    await editor.fill(edited);
    await page.getByTestId('device-select-e2e-sw-01').click();
    await page.getByTestId('unsaved-save').click();
    await expect(editor).toContainText('name: e2e-sw-01');
    const savedConfig = await request.get('/api/v1/config');
    expect((await savedConfig.json()).content).toContain('preserved router edit');

    await editor.fill(`${await editor.innerText()}\n# discard this edit`);
    await page.getByTestId('device-select-e2e-rtr-01').click();
    await page.getByTestId('unsaved-discard').click();
    await expect(editor).toContainText('name: e2e-rtr-01');
    expect((await (await request.get('/api/v1/config')).json()).content).not.toContain(
      'discard this edit',
    );
    await editor.fill(`${await editor.innerText()}\n# history edit`);
    await page.goBack();
    await expect(page.getByRole('dialog', { name: 'Unsaved changes' })).toBeVisible();
    await page.getByTestId('unsaved-cancel').click();
    await expect(page).toHaveURL(/\/devices$/);
    await expect(editor).toContainText('history edit');
    await page.goBack();
    await page.getByTestId('unsaved-discard').click();
    await expect(page).toHaveURL(/\/$/);
    await page.goForward();
    await expect(page).toHaveURL(/\/devices$/);
    await sidebar.getByTestId('nav-item-root').click();
    await expect(page).toHaveURL(/\/$/);
    await page.goBack();
    await page.getByTestId('device-select-e2e-rtr-01').click();
    await expect(editor).toContainText('name: e2e-rtr-01');
    await expect(editor).not.toContainText('networks:');
    await editor.fill(`${await editor.innerText()}\n# forward edit`);
    await page.goForward();
    await expect(page.getByRole('dialog', { name: 'Unsaved changes' })).toBeVisible();
    await page.getByTestId('unsaved-cancel').click();
    await expect(editor).toContainText('forward edit');
    await page.goForward();
    await page.getByTestId('unsaved-discard').click();
    await expect(page).toHaveURL(/\/$/);

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
    await page.getByRole('heading', { level: 1 }).click();
    await page.keyboard.press('g');
    await page.keyboard.press('h');
    await expect(page.getByRole('dialog', { name: 'Unsaved changes' })).toBeVisible();
    await page.getByTestId('unsaved-cancel').click();
    await expect(draftEditor).toContainText('preserved wizard edit');
    await page.getByTestId('sidebar-desktop').getByTestId('nav-item-root').click();
    await expect(page.getByRole('dialog', { name: 'Unsaved changes' })).toBeVisible();
    await page.getByTestId('unsaved-cancel').click();
    await expect(draftEditor).toContainText('preserved wizard edit');
    await page.getByTestId('sidebar-desktop').getByTestId('nav-item-root').click();
    await page.getByTestId('unsaved-save').click();
    await expect(page).toHaveURL(/\/$/);
    const savedDraft = await request.get(
      `/api/v1/library/drafts/${encodeURIComponent(draft.name)}`,
    );
    expect((await savedDraft.json()).content).toContain('preserved wizard edit');
  } finally {
    const stopped = await request.delete('/api/v1/simulation?sessionId=unsaved-acceptance', {
      headers,
    });
    expect(stopped.ok(), await stopped.text()).toBe(true);
  }
});
