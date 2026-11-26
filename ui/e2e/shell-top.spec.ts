import { expect, type Locator, type Page, test } from '@playwright/test';
import { chooseSession, openMobileSidebar } from './support/sidebar';

async function tabTo(page: Page, target: Locator): Promise<void> {
  await expect(target).toBeVisible();
  for (let step = 0; step < 60; step += 1) {
    await page.keyboard.press('Tab');
    if (await target.evaluate((node) => node === document.activeElement)) break;
  }
  await expect(target).toBeFocused();
  await expect(target).toBeInViewport({ ratio: 1 });
}

/** `minHeight` is 40 (the shared footer's `h-10`); see the collapsed-rail note at the call site. */
async function expectControlFits(control: Locator, minHeight = 40): Promise<void> {
  const bounds = await control.evaluate((node) => {
    const box = node.getBoundingClientRect();
    const rail = node.closest('aside')?.getBoundingClientRect();
    return {
      width: box.width,
      height: box.height,
      contained: !!rail && box.left >= rail.left && box.right <= rail.right,
    };
  });
  expect(bounds.width).toBeGreaterThanOrEqual(40);
  expect(bounds.height).toBeGreaterThanOrEqual(minHeight);
  expect(bounds.contained).toBe(true);
}

test.beforeEach(async ({ page }) => {
  await page.route('**/api/v1/simulation', (route) =>
    route.fulfill({
      json: {
        running: true,
        sessionId: 'hospital',
        deviceCount: 0,
        uptimeSeconds: 1,
        sessions: ['hospital', 'warehouse'].map((sessionId) => ({
          running: true,
          sessionId,
          deviceCount: 0,
          uptimeSeconds: 1,
        })),
      },
    }),
  );
  await page.route('**/api/v1/sessions/*/**', (route) => route.fulfill({ json: [] }));
});

test('appearance settings and the rail toggle share the visible theme', async ({ page }) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.goto('/devices');
  const rail = page.getByTestId('sidebar-desktop');
  await rail.getByTestId('rail-settings').click();
  const drawer = page.getByTestId('settings-drawer');
  await drawer.getByRole('tab', { name: 'Appearance', exact: true }).click();
  await drawer.getByRole('radio', { name: 'Light', exact: true }).check({ force: true });
  await page.getByTestId('settings-drawer-close').click();
  await expect(page.locator('html')).not.toHaveClass(/dark/);
  await expect(page.locator('html')).toHaveCSS('color-scheme', 'light');
  await expect(rail.getByTestId('rail-theme')).toHaveAccessibleName('Switch to dark mode');
  await rail.getByTestId('rail-theme').click();
  await expect(page.locator('html')).toHaveClass(/dark/);
  // Native controls (select, scrollbars) follow color-scheme, not the tokens.
  await expect(page.locator('html')).toHaveCSS('color-scheme', 'dark');
});

for (const collapsed of [false, true]) {
  test(`desktop rail controls are keyboard reachable, collapsed=${collapsed}`, async ({ page }) => {
    await page.setViewportSize({ width: 1440, height: 900 });
    await page.addInitScript(
      (value) => localStorage.setItem('niac-sidebar-collapsed', String(value)),
      collapsed,
    );
    await page.goto('/devices');
    await expect(page.getByTestId('page-header-title')).toBeVisible();
    const rail = page.getByTestId('sidebar-desktop');
    const status = rail.getByTestId('rail-status');
    // The status is a verdict to read, not a control: a pill in the rail, a dot
    // when collapsed.
    await expect(status).toBeVisible();
    if (!collapsed) await expect(status).toContainText('Online');
    const switcher = rail.getByTestId('rail-product-slot');
    await tabTo(page, switcher);
    await expectControlFits(switcher);
    const theme = rail.getByTestId('rail-theme');
    await tabTo(page, theme);
    // The shared footer's collapsed icon buttons measure 20px high (its `flex-1` in a
    // column collapses their `h-10`), under the 24px WCAG 2.5.8 minimum; asserted at
    // the height they render until the shared footer gives them back 40.
    await expectControlFits(theme, collapsed ? 20 : 40);
    await page.keyboard.press('Enter');
    await expect(theme).toHaveAccessibleName('Switch to dark mode');
    await expect(page.locator('html')).not.toHaveClass(/dark/);
    if (collapsed) {
      await tabTo(page, rail.getByTestId('rail-collapse'));
      await page.keyboard.press('Enter');
    }
    await chooseSession(rail, 'warehouse');
    await expect(switcher).toContainText('warehouse');
    await expect(page.getByRole('main').getByTestId('rail-status')).toHaveCount(0);
    await expect(rail).toHaveJSProperty(
      'scrollWidth',
      await rail.evaluate((node) => node.clientWidth),
    );
  });
}

test('phone drawer retains controls after desktop collapse and shares the theme', async ({
  page,
}) => {
  await page.setViewportSize({ width: 1440, height: 900 });
  await page.addInitScript(() => localStorage.setItem('niac-sidebar-collapsed', 'true'));
  await page.goto('/devices');
  const desktopTheme = page.getByTestId('sidebar-desktop').getByTestId('rail-theme');
  await desktopTheme.click();
  await page.setViewportSize({ width: 390, height: 844 });
  const rail = await openMobileSidebar(page);
  await expect(rail.getByTestId('rail-theme')).toHaveAccessibleName('Switch to dark mode');
  const switcher = rail.getByTestId('rail-product-slot');
  await tabTo(page, switcher);
  await tabTo(page, rail.getByTestId('rail-theme'));
  await page.keyboard.press('Enter');
  await expect(page.locator('html')).toHaveClass(/dark/);
  await chooseSession(rail, 'warehouse');
  await expect(switcher).toContainText('warehouse');
  await page.getByTestId('mobile-menu-toggle').click();
  await expect(rail.locator('button')).toHaveCount(0);
  await expect(page.getByRole('main').getByTestId('rail-status')).toHaveCount(0);
});

for (const scheme of ['light', 'dark'] as const) {
  test.describe(`OS set to ${scheme}`, () => {
    test.use({ colorScheme: scheme, viewport: { width: 1440, height: 900 } });

    test('the theme follows the OS until the operator picks a side', async ({ page }) => {
      await page.goto('/devices');
      const html = page.locator('html');
      const opposite = scheme === 'dark' ? /^(?!.*dark)/ : /dark/;
      const own = scheme === 'dark' ? /dark/ : /^(?!.*dark)/;
      await expect(html).toHaveClass(own);

      // An explicit choice wins over the OS, and survives a reload.
      await page.getByTestId('sidebar-desktop').getByTestId('rail-theme').click();
      await expect(html).toHaveClass(opposite);
      await page.reload();
      await expect(html).toHaveClass(opposite);
    });
  });
}
