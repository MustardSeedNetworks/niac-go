import { expect, test } from '@playwright/test';
import type { ScenarioDraft, ScenarioPack } from '../src/api/scenario-client';
import type { LibraryNetwork, Template } from '../src/api/template-types';
import { parseNetworkModel } from '../src/components/wizard/network-addressing';

test('Hospital pack preserves its AP uplinks through Networks and Review', async ({ page }) => {
  // The pack builds and addresses ~250 devices before Review, and rendering that
  // many addressing rows is the cost: chromium 21.4s, webkit 22.0s on the
  // chromium+webkit matrix #2248 settled on. The 30s default is too tight and
  // 90s was not enough either — on the old matrix firefox took ~66s under
  // contention, hit exactly 90s on a CI runner and passed only on retry, which
  // the job's zero flake budget fails. Held at 4-5x the measured cost, because
  // what broke the budget last time was runner contention, not the work itself.
  test.setTimeout(120000);
  await page.goto('/new-simulation');
  await page.getByTestId('wizard-interface-select').selectOption({ index: 1 });
  const packs: ScenarioPack[] = await (await page.request.get('/api/v1/scenario/packs')).json();
  const hospital = packs.find((pack) => pack.id === 'hospital');
  expect(hospital).toBeDefined();
  if (!hospital) throw new Error('Hospital pack missing');
  const expectedAPs =
    hospital.request.sites.length *
    hospital.request.counts.accessSwitches *
    hospital.request.counts.accessPointsPerAccess;
  await page.getByTestId('scenario-pack-hospital').click();
  const created = page.waitForResponse(
    (response) =>
      response.url().endsWith('/api/v1/library/drafts') && response.request().method() === 'POST',
  );
  await page.getByTestId('wizard-next-button').click();
  const response = await created;
  expect(response.ok()).toBe(true);
  const draft: ScenarioDraft = await response.json();
  const model = parseNetworkModel(draft.content);
  const accessPoints = model.devices.filter(
    (device) => device.interfaceName === 'mGigabitEthernet0',
  );
  expect(expectedAPs).toBeGreaterThan(0);
  expect(accessPoints).toHaveLength(expectedAPs);
  await expect(page.getByTestId('wizard-step-devices')).toHaveAttribute('data-status', 'active');
  await page.getByTestId('wizard-next-button').click();
  await expect(page.getByTestId('wizard-step-networks')).toHaveAttribute('data-status', 'active');
  const addresses = page.getByTestId(/^addressing-address-/);
  await expect(addresses).toHaveCount(model.devices.length);
  for (const device of model.devices) expect(device.address).toBeTruthy();
  expect(await addresses.allTextContents()).toEqual(model.devices.map((device) => device.address));
  await expect(page.getByTestId('addressing-assign-all')).toBeDisabled();
  // Read the uplinks in one pass, the same way the addresses above are read.
  // The per-AP loop this replaces made two awaited round trips per device; the
  // batch covers every device rather than only the APs. It was not what fixed
  // the flake that prompted it — the engine measured 59s with the loop and
  // 49-53s without, so the cost is rendering the rows, not the assertions.
  // Kept because asserting more in fewer round trips is the better test anyway.
  const networks = page.getByTestId(/^addressing-network-/);
  await expect(networks).toHaveCount(model.devices.length);
  expect(
    await networks.evaluateAll((elements) =>
      elements.map((element) => (element as HTMLInputElement | HTMLSelectElement).value),
    ),
  ).toEqual(model.devices.map((device) => device.network ?? ''));
  for (const device of accessPoints) expect(device.address).toBeTruthy();
  await page.getByTestId('wizard-next-button').click();
  await expect(page.getByTestId('wizard-step-protocols')).toHaveAttribute('data-status', 'active');
  await page.getByTestId('wizard-next-button').click();
  await expect(page.getByTestId('wizard-step-review')).toHaveAttribute('data-status', 'active');
  const saved = await page.request.get(`/api/v1/library/drafts/${encodeURIComponent(draft.name)}`);
  expect(saved.ok()).toBe(true);
  const reviewed: ScenarioDraft = await saved.json();
  expect(reviewed.content).toBe(draft.content);
});

for (const theme of ['light', 'dark']) {
  test(`Help search has visible keyboard focus and a single close stop in ${theme}`, async ({
    page,
  }) => {
    await page.addInitScript((value) => localStorage.setItem('niac-theme', value), theme);
    await page.goto('/');
    // Open it the way a keyboard user does. The close assertion at the end of
    // this test is that focus returns to the trigger, and the focus trap
    // restores whatever held focus when it armed — so the trigger has to hold
    // focus for that to mean anything. WebKit on macOS does not focus a
    // <button> on click (platform convention), which leaves the trap
    // restoring to <body> and says nothing about the drawer.
    const helpButton = page.getByTestId('sidebar-desktop').getByTestId('sidebar-help-button');
    await helpButton.focus();
    await page.keyboard.press('Enter');
    const drawer = page.getByTestId('help-drawer');
    const close = page.getByTestId('help-drawer-close');
    await expect(drawer).toBeVisible();
    await expect(drawer.getByRole('button').first()).toBeFocused();
    await close.focus();
    await page.keyboard.press('Tab');
    const search = drawer.getByRole('textbox');
    await expect(search).toBeFocused();
    await search.evaluate(async (element) => {
      await Promise.all(element.getAnimations().map((animation) => animation.finished));
    });
    const visibleFocus = await search.evaluate((element) => {
      const style = getComputedStyle(element);
      const canvas = document.createElement('canvas');
      const context = canvas.getContext('2d');
      if (!context) throw new Error('Canvas color inspection unavailable');
      const visible = (color: string) => {
        context.clearRect(0, 0, 1, 1);
        context.fillStyle = color;
        context.fillRect(0, 0, 1, 1);
        return context.getImageData(0, 0, 1, 1).data[3] > 0;
      };
      const outline =
        style.outlineStyle !== 'none' &&
        Number.parseFloat(style.outlineWidth) >= 2 &&
        visible(style.outlineColor);
      const shadowColors =
        style.boxShadow.match(/(?:rgba?|hsla?|oklch|oklab|color)\([^)]*\)/g) ?? [];
      const shadow = /[2-9]\d*px/.test(style.boxShadow) && shadowColors.some(visible);
      return { visible: outline || shadow, outline: style.outline, shadow: style.boxShadow };
    });
    expect(visibleFocus.visible, JSON.stringify(visibleFocus)).toBe(true);
    let closeStops = 0;
    let completedCycle = false;
    for (let index = 0; index < 70; index++) {
      await page.keyboard.press('Tab');
      expect(await drawer.evaluate((element) => element.contains(document.activeElement))).toBe(
        true,
      );
      if (await close.evaluate((element) => element === document.activeElement)) closeStops++;
      if (await search.evaluate((element) => element === document.activeElement)) {
        completedCycle = true;
        break;
      }
    }
    expect(completedCycle).toBe(true);
    expect(closeStops).toBe(1);
    await expect(drawer.getByRole('button', { name: 'Close help', exact: true })).toHaveCount(1);
    await close.focus();
    await page.keyboard.press('Enter');
    await expect(drawer).toBeHidden();
    await expect(
      page.getByTestId('sidebar-desktop').getByTestId('sidebar-help-button'),
    ).toBeFocused();
  });
}

test('Spanish page help and shared glossary render from the active locale', async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('niac-language', 'es'));
  for (const [route, text] of [
    ['/new-simulation', 'Revisar antes de iniciar'],
    ['/runtime', 'Inicie, detenga e inspeccione la simulación.'],
    ['/packets', 'Captura en vivo'],
  ]) {
    await page.goto(route);
    await page.getByRole('button', { name: /^Open help for / }).click();
    const drawer = page.getByTestId('help-drawer');
    await expect(drawer.getByText(text, { exact: false }).last()).toBeVisible();
    await page.getByTestId('help-drawer-close').click();
  }
  await page.getByTestId('sidebar-desktop').getByTestId('sidebar-help-button').click();
  const drawer = page.getByTestId('help-drawer');
  await drawer.getByRole('tab', { name: 'Glosario', exact: true }).click();
  await drawer.getByRole('textbox').fill('Borrador');
  await expect(drawer.getByText('Borrador', { exact: true })).toBeVisible();
  await expect(
    drawer.getByText('Configuración de simulación guardada y editable.', { exact: false }),
  ).toBeVisible();
  await expect(drawer.getByText('Draft', { exact: true })).toHaveCount(0);
});

test('the starting point stays compact and the library supports keyboard, search and family selection', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto('/new-simulation');
  const start = page.getByTestId('wizard-source-tab-start');
  const library = page.getByTestId('wizard-source-tab-library');
  const panel = page.getByTestId('wizard-source-panel');
  await expect(start).toHaveAttribute('aria-selected', 'true');
  await expect(page.getByTestId('scenario-pack-hospital')).toBeVisible();
  await expect(page.getByTestId('config-picker-search')).toHaveCount(0);
  await expect(page.getByTestId('fleet-domain')).toBeHidden();
  const next = page.getByTestId('wizard-next-button');
  const height = await next.evaluate(
    (element) => element.getBoundingClientRect().bottom + window.scrollY,
  );
  expect(height).toBeLessThanOrEqual(1800);
  await page.screenshot({ path: test.info().outputPath('wizard-start-1440.png'), fullPage: true });

  await start.focus();
  await page.keyboard.press('ArrowRight');
  await expect(library).toBeFocused();
  await expect(library).toHaveAttribute('aria-selected', 'true');
  const templateResponse = await page.request.get('/api/v1/templates');
  expect(templateResponse.ok()).toBe(true);
  const templates: Template[] = await templateResponse.json();
  const networksResponse = await page.request.get('/api/v1/library/networks');
  expect(networksResponse.ok()).toBe(true);
  const networks: LibraryNetwork[] = await networksResponse.json();
  expect(templates.length + networks.length).toBe(32);
  const allCards = page.getByTestId(/^config-item-/);
  await expect(allCards).toHaveCount(32);
  expect(
    (
      await allCards.evaluateAll((elements) =>
        elements.map((element) => element.getAttribute('data-testid') ?? ''),
      )
    ).sort((a, b) => a.localeCompare(b)),
  ).toEqual(
    [
      ...templates.map((template) => `config-item-builtin:${template.name}`),
      ...networks.map((network) => `config-item-saved:${network.name}`),
    ].sort((a, b) => a.localeCompare(b)),
  );
  const cards = page.getByTestId(/^config-item-builtin:/);
  await expect(cards).toHaveCount(templates.length);
  expect(
    (
      await cards.evaluateAll((elements) =>
        elements.map((element) => element.getAttribute('data-testid') ?? ''),
      )
    ).sort((a, b) => a.localeCompare(b)),
  ).toEqual(
    templates
      .map((template) => `config-item-builtin:${template.name}`)
      .sort((a, b) => a.localeCompare(b)),
  );
  await page.keyboard.press('Tab');
  await expect(page.getByTestId('config-upload')).toBeFocused();
  await page.keyboard.press('Tab');
  const search = page.getByTestId('config-picker-search');
  await expect(search).toBeFocused();
  await page.keyboard.press('Tab');
  const family = page.getByTestId('config-picker-family');
  await expect(family).toBeFocused();

  const selected = templates.find((template) => template.vendor);
  expect(selected?.vendor).toBeTruthy();
  if (!selected?.vendor) throw new Error('No vendor template in the library');
  await search.fill(selected.vendor);
  await expect(cards).not.toHaveCount(0);
  expect(await cards.count()).toBeLessThan(templates.length);
  await expect(page.getByTestId(`config-item-builtin:${selected.name}`)).toBeVisible();
  await search.fill('');
  await family.selectOption(selected.type);
  await expect(cards).toHaveCount(
    templates.filter((template) => template.type === selected.type).length,
  );
  await search.fill(selected.name);
  const card = page.getByTestId(`config-item-builtin:${selected.name}`);
  await card.getByRole('button', { name: 'Select', exact: true }).click();
  await expect(start).toHaveAttribute('aria-selected', 'true');
  await expect(start).toBeFocused();
  await expect(page.getByTestId('wizard-selected-library')).toBeVisible();
  await expect(page.getByTestId('wizard-selected-library')).toContainText(selected.name);
  await expect(panel.getByTestId('config-picker-search')).toHaveCount(0);
  await expect(page.locator('[data-wizard-source][aria-pressed="true"]')).toHaveCount(1);
});

test('wizard starting points and library fit a 390px viewport', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/new-simulation');
  await expect(page.getByTestId('scenario-pack-hospital')).toBeVisible();
  await page.screenshot({ path: test.info().outputPath('wizard-start-390.png'), fullPage: true });
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(390);
  await page.getByTestId('wizard-source-tab-library').click();
  await expect(page.getByTestId('config-item-builtin:router')).toBeVisible();
  await expect(page.getByTestId('config-picker-search')).toBeInViewport();
  await expect(page.getByTestId('config-picker-family')).toBeInViewport();
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(390);
});
