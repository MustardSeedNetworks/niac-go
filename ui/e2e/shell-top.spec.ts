import { expect, type Locator, type Page, test } from '@playwright/test';
import { openMobileSidebar } from './support/sidebar';

async function tabTo(page: Page, target: Locator): Promise<void> {
  await expect(target).toBeVisible();
  for (let step = 0; step < 60; step += 1) {
    await page.keyboard.press('Tab');
    if (await target.evaluate((node) => node === document.activeElement)) break;
  }
  await expect(target).toBeFocused();
  await expect(target).toBeInViewport({ ratio: 1 });
}

async function expectControlFits(control: Locator): Promise<void> {
  const bounds = await control.evaluate((node) => {
    const box = node.getBoundingClientRect();
    const rail = node.closest('aside')?.getBoundingClientRect();
    return {
      width: box.width,
      height: box.height,
      contained: !!rail && box.left >= rail.left && box.right <= rail.right,
    };
  });
  expect(bounds.width).toBeGreaterThanOrEqual(44);
  expect(bounds.height).toBeGreaterThanOrEqual(44);
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
  await rail.getByTestId('sidebar-settings-button').click();
  const drawer = page.getByTestId('settings-drawer');
  await drawer.getByRole('tab', { name: 'Appearance', exact: true }).click();
  await drawer.getByRole('button', { name: 'Light', exact: true }).click();
  await page.getByTestId('settings-drawer-close').click();
  await expect(page.locator('html')).not.toHaveClass(/dark/);
  await expect(rail.getByTestId('theme-toggle')).toHaveAccessibleName('Switch to dark mode');
  await rail.getByTestId('theme-toggle').click();
  await expect(page.locator('html')).toHaveClass(/dark/);
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
    const status = rail.getByTestId('connection-status');
    await tabTo(page, status);
    await expectControlFits(status);
    await expect(status).toHaveAccessibleName('Connected to backend');
    const theme = rail.getByTestId('theme-toggle');
    await tabTo(page, theme);
    await expectControlFits(theme);
    await page.keyboard.press('Enter');
    await expect(theme).toHaveAccessibleName('Switch to dark mode');
    await expect(page.locator('html')).not.toHaveClass(/dark/);
    if (collapsed) {
      await tabTo(page, rail.getByTestId('sidebar-expand'));
      await page.keyboard.press('Enter');
    }
    const switcher = rail.getByTestId('session-switcher-select');
    await tabTo(page, switcher);
    await expectControlFits(switcher);
    await switcher.selectOption('warehouse');
    await expect(switcher).toHaveValue('warehouse');
    await expect(page.getByRole('main').getByTestId('connection-status')).toHaveCount(0);
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
  const desktopTheme = page.getByTestId('sidebar-desktop').getByTestId('theme-toggle');
  await desktopTheme.click();
  await page.setViewportSize({ width: 390, height: 844 });
  const rail = await openMobileSidebar(page);
  await expect(rail.getByTestId('theme-toggle')).toHaveAccessibleName('Switch to dark mode');
  await tabTo(page, rail.getByTestId('connection-status'));
  await tabTo(page, rail.getByTestId('theme-toggle'));
  await page.keyboard.press('Enter');
  await expect(page.locator('html')).toHaveClass(/dark/);
  const switcher = rail.getByTestId('session-switcher-select');
  await tabTo(page, switcher);
  await switcher.selectOption('warehouse');
  await expect(switcher).toHaveValue('warehouse');
  await page.getByTestId('mobile-menu-toggle').click();
  await expect(rail).toHaveAttribute('inert', '');
  await expect(page.getByRole('main').getByTestId('connection-status')).toHaveCount(0);
});
