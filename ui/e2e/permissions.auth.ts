import { expect, type Page, test } from '@playwright/test';

async function connect(page: Page, scope: string) {
  await page.goto('/');
  await page.getByTestId('api-token-input').fill(`niac-scope-test-${scope}`);
  await page.getByTestId('auth-gate-connect').click();
  await expect(page.getByTestId('page-header-title')).toHaveText('Dashboard');
  const response = await page.request.get('/api/v1/auth/scope', {
    headers: { Authorization: `Bearer niac-scope-test-${scope}` },
  });
  expect(await response.json()).toEqual({ scope });
}

test('viewer can filter and navigate but cannot prepare or start a simulation', async ({
  page,
}) => {
  await connect(page, 'read-only');
  await page.getByTestId('sidebar-desktop').getByTestId('nav-item-runtime').click();
  await expect(page.getByTestId('page-header-title')).toHaveText('Simulation');
  const search = page.getByTestId('config-picker-search');
  await search.fill('office');
  await expect(search).toHaveValue('office');
  await expect(page.getByTestId('runtime-prepare')).toBeDisabled();
  await expect(page.getByTestId('runtime-prepare')).toHaveAttribute(
    'title',
    /token does not allow/,
  );
});

test('operator can start while a viewer can inspect and export but not stop', async ({
  page,
  browser,
  baseURL,
}) => {
  await connect(page, 'read-write');
  await page.getByTestId('sidebar-desktop').getByTestId('nav-item-runtime').click();
  await page.getByTestId('config-picker-search').fill('');
  await page.getByRole('button', { name: 'Select', exact: true }).first().click();
  await page.getByTestId('runtime-interface').selectOption({ index: 1 });
  const prepare = page.getByTestId('runtime-prepare');
  await expect(prepare).toBeEnabled();
  await prepare.click();
  await expect(page.getByTestId('wizard-preflight-check')).toBeEnabled();
  await page.getByTestId('wizard-attachment-mode').selectOption('direct');
  await page.getByTestId('wizard-preflight-check').click();
  await expect(page.getByTestId('wizard-preflight-start')).toBeEnabled();
  const started = page.waitForResponse(
    (response) =>
      response.url().endsWith('/api/v1/simulation') && response.request().method() === 'POST',
  );
  await page.getByTestId('wizard-preflight-start').click();
  expect((await started).ok()).toBe(true);
  const viewer = await browser.newPage({ baseURL, ignoreHTTPSErrors: true });
  try {
    await connect(viewer, 'read-only');
    await viewer.getByTestId('sidebar-desktop').getByTestId('nav-item-runtime').click();
    await expect(viewer.getByTestId('runtime-stop')).toBeDisabled();
    const download = viewer.waitForEvent('download');
    await viewer.getByTestId('runtime-download-yaml').click();
    expect((await download).suggestedFilename()).toMatch(/\.ya?ml$/);
  } finally {
    await viewer.close();
    await page.getByTestId('runtime-stop').click();
    await page.getByRole('dialog').getByRole('button', { name: /stop/i }).click();
  }
});
