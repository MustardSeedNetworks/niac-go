import { expect, type Locator, type Page, test } from '@playwright/test';
import { sidebar } from './support/sidebar';

async function checkTooltip(page: Page, trigger: Locator, hoverTarget = trigger) {
  const id = await trigger.getAttribute('aria-describedby');
  expect(id).toBeTruthy();
  const tooltip = page.locator(`[id="${id}"]`);
  await hoverTarget.hover();
  await expect(tooltip).toBeVisible();
  await tooltip.hover();
  await expect(tooltip).toBeVisible();
  const box = await tooltip.boundingBox();
  expect(box).not.toBeNull();
  expect(box?.x).toBeGreaterThanOrEqual(0);
  expect((box?.x ?? 0) + (box?.width ?? 0)).toBeLessThanOrEqual(page.viewportSize()?.width ?? 0);
  await page.mouse.move(0, 0);
  await trigger.focus();
  await page.keyboard.press('Shift+Tab');
  await page.keyboard.press('Tab');
  await expect(trigger).toBeFocused();
  await expect(tooltip).toBeVisible();
  await expect(trigger).toHaveAccessibleDescription(/.+/);
  await page.keyboard.press('Escape');
  await expect(tooltip).toBeHidden();
  await expect(trigger).toBeFocused();
}

test.beforeEach(async ({ page }) => {
  await page.route('**/api/v1/auth/scope', (route) => route.fulfill({ json: { scope: 'admin' } }));
  await page.route('**/api/v1/simulation', (route) =>
    route.fulfill({ json: { running: false, interface: '', deviceCount: 0 } }),
  );
  await page.route('**/api/v1/library/walks', (route) =>
    route.fulfill({
      json: [
        {
          name: 'router.walk',
          sizeBytes: 200,
          modifiedAt: '2026-09-15T00:00:00Z',
          source: 'starter',
          edited: false,
        },
      ],
    }),
  );
  await page.route('**/api/v1/debug/level', (route) =>
    route.fulfill({ json: { level: 'info', defaultLevel: 'basic' } }),
  );
});

test('walk controls keep their labels and expose instructions by keyboard', async ({ page }) => {
  await page.goto('/walk-analyzer');
  const path = page.getByTestId('walk-analyzer-path-input');
  await expect(path).toBeVisible();
  const originalName = await path.evaluate((element) => element.closest('label')?.innerText.trim());
  expect(originalName).toBeTruthy();
  await checkTooltip(page, path);
  await expect(path).toHaveAccessibleName(originalName);
  await checkTooltip(page, page.getByTestId('walk-analyzer-analyze-button'));
});

test('a denied action reveals its reason without executing by mouse, Enter, or Space', async ({
  page,
}) => {
  await page.route('**/api/v1/auth/scope', (route) =>
    route.fulfill({ json: { scope: 'read-only' } }),
  );
  const writes: string[] = [];
  page.on('request', (request) => {
    if (['POST', 'PUT', 'PATCH', 'DELETE'].includes(request.method())) writes.push(request.url());
  });
  await page.goto('/walk-analyzer');
  const button = page.getByTestId('walk-analyzer-analyze-button');
  await expect(button).toHaveAttribute('aria-disabled', 'true');
  await checkTooltip(page, button);
  await expect(button).toHaveAccessibleDescription('Your token does not allow this action.');
  await page.keyboard.press('Enter');
  await page.keyboard.press('Space');
  await button.click({ force: true });
  await page.goto('/debug');
  await page.getByTestId('debug-level-toggle').click();
  const radio = page.getByTestId('debug-level-info');
  await expect(radio).toBeChecked();
  await expect(radio).toHaveAttribute('aria-disabled', 'true');
  await checkTooltip(page, radio, radio.locator('..'));
  await page.keyboard.press('ArrowRight');
  await page.getByTestId('debug-level-trace').click({ force: true });
  await expect(radio).toBeChecked();
  expect(writes).toEqual([]);
});

test('file upload, static status, and help version have keyboard descriptions at 390px', async ({
  page,
}) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto('/runtime');
  await checkTooltip(page, page.getByTestId('config-upload'));
  expect(await page.evaluate(() => document.documentElement.scrollWidth)).toBeLessThanOrEqual(390);
  await checkTooltip(page, page.getByTestId('connection-status').filter({ visible: true }));
  await page.getByTestId('mobile-menu-toggle').click();
  await sidebar(page, 'mobile').getByTestId('sidebar-help-button').click();
  const drawer = page.getByTestId('help-drawer');
  await expect(drawer).toBeInViewport({ ratio: 1 });
  const drawerBounds = await drawer.boundingBox();
  expect(drawerBounds).not.toBeNull();
  expect(drawerBounds?.x).toBeGreaterThanOrEqual(0);
  expect((drawerBounds?.x ?? 0) + (drawerBounds?.width ?? 0)).toBeLessThanOrEqual(390);
  await expect(page.getByTestId('help-drawer-close')).toBeInViewport();
  await checkTooltip(page, page.getByTestId('help-drawer-version'));
  await page.keyboard.press('Shift+Tab');
  await page.keyboard.press('Tab');
  await page.screenshot({ path: test.info().outputPath('tooltip-390.png') });
  await page.getByTestId('help-drawer-close').click();
  await expect(drawer).toBeHidden();
});

test('debug radio descriptions belong to the focused radio', async ({ page }) => {
  await page.goto('/debug');
  await page.getByTestId('debug-level-toggle').click();
  const radio = page.getByTestId('debug-level-info');
  await expect(radio).toBeChecked();
  await checkTooltip(page, radio, radio.locator('..'));
  await expect(radio).toHaveAccessibleName('Info');
});

test('collapsed navigation keeps every name and reveals its label on focus', async ({ page }) => {
  await page.goto('/');
  const nav = sidebar(page);
  const buttons = nav.locator('[data-testid^="nav-item-"]');
  const names = await buttons.evaluateAll((elements) =>
    elements.map((element) => ({
      id: element.getAttribute('data-testid') ?? '',
      name: element.getAttribute('aria-label') ?? '',
    })),
  );
  expect(names.length).toBeGreaterThan(0);
  await nav.getByRole('button', { name: 'Collapse sidebar' }).click();
  for (const { id, name } of names) {
    expect(name).not.toBe('');
    await expect(nav.getByTestId(id)).toHaveAccessibleName(name);
  }
  const simulation = nav.getByTestId('nav-item-runtime');
  await checkTooltip(page, simulation);
  await simulation.click();
  await expect(page).toHaveURL(/\/runtime$/);
});

test('Escape dismisses a hover-only tooltip before closing its modal', async ({ page }) => {
  await page.goto('/runtime');
  await sidebar(page, 'desktop').getByTestId('sidebar-help-button').click();
  const drawer = page.getByTestId('help-drawer');
  const version = page.getByTestId('help-drawer-version');
  const close = page.getByTestId('help-drawer-close');
  await expect(version).toBeFocused();
  await page.keyboard.press('Tab');
  await expect(close).toBeFocused();
  await version.hover();
  const descriptionId = await version.getAttribute('aria-describedby');
  expect(descriptionId).toBeTruthy();
  const tooltip = page.locator(`[id="${descriptionId}"]`);
  await expect(tooltip).toBeVisible();
  await expect(version).not.toBeFocused();
  await page.keyboard.press('Escape');
  await expect(tooltip).toBeHidden();
  await expect(close).toBeFocused();
  await expect(drawer).toBeVisible();
  await page.keyboard.press('Escape');
  await expect(drawer).toBeHidden();
});

test('activating Help clears its tooltip so Escape closes the drawer once focus leaves version', async ({
  page,
}) => {
  await page.goto('/runtime');
  const help = sidebar(page, 'desktop').getByTestId('sidebar-help-button');
  const descriptionId = await help.getAttribute('aria-describedby');
  expect(descriptionId).toBeTruthy();
  const tooltip = page.locator(`[id="${descriptionId}"]`);
  await help.hover();
  await expect(tooltip).toBeVisible();
  await help.click();
  const drawer = page.getByTestId('help-drawer');
  await expect(drawer).toBeVisible();
  await expect(tooltip).toBeHidden();
  await expect(page.getByTestId('help-drawer-version')).toBeFocused();
  await page.keyboard.press('Tab');
  await expect(page.getByTestId('help-drawer-close')).toBeFocused();
  await page.keyboard.press('Escape');
  await expect(drawer).toBeHidden();
});
