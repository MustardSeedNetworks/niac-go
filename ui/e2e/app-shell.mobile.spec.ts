import { expect, test } from '@playwright/test';
import { checkLongHistoryLayout } from './support/history-layout';
import { openMobileSidebar, sidebar } from './support/sidebar';

/**
 * The narrow-viewport smoke subset.
 *
 * Runs on the two policy engines at a phone-width viewport (owner 2026-09-15:
 * everything works at 390×844). It covers the three things a viewport can
 * break that nothing else here would notice: the shell renders, the primary
 * navigation is reachable and operable, and the main journey completes.
 *
 * A viewport, not a device project: E2E_CONVENTIONS.md covers the narrow
 * layout obligation on the engines already tested and bans phone/tablet
 * presets. Before #2247 this file ran on three device projects that the
 * desktop projects ignored, and it was the reason those projects existed.
 */
test.describe('app shell on small screens', () => {
  test.use({ viewport: { width: 390, height: 844 } });

  test('long run names preserve layout and mobile navigation', async ({ page }) => {
    await checkLongHistoryLayout(page);
  });

  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');
  });

  test('renders the shell without overflowing horizontally', async ({ page }) => {
    await expect(page.getByTestId('page-header-title')).toBeVisible();

    // A layout that overflows sideways is the classic mobile-only regression:
    // it looks fine on desktop, and on a phone it strands content off-screen
    // with no way to reach it.
    const overflow = await page.evaluate(() => {
      const doc = document.documentElement;
      return { scrollWidth: doc.scrollWidth, clientWidth: doc.clientWidth };
    });
    expect(
      overflow.scrollWidth,
      `page scrolls horizontally: ${overflow.scrollWidth}px of content in a ${overflow.clientWidth}px viewport`,
    ).toBeLessThanOrEqual(overflow.clientWidth + 1);
  });

  test('the closed drawer is hidden, not parked off-screen', async ({ page }) => {
    const drawer = sidebar(page, 'mobile');
    const toggle = page.getByTestId('mobile-menu-toggle');

    // Off-canvas alone left the drawer painted at x -288..0, which the fleet's
    // 390px gate fails as content leaving the viewport.
    await expect(drawer).toBeHidden();

    await toggle.click();
    await expect(drawer).toBeVisible();
    await toggle.click();
    await expect(drawer).toBeHidden();
  });

  test('the closed drawer holds no control and the screen one product mark', async ({ page }) => {
    const drawer = sidebar(page, 'mobile');
    await expect(page.getByTestId('page-header-title')).toBeVisible();

    // #2284: parked in the tree, the closed drawer kept the whole rail —
    // nav links, Help, Settings and a second product mark — in a menu nobody
    // could see. It mounts on open now, so there is nothing to hide.
    // A CSS locator, not getByRole: role queries skip `visibility: hidden`, so
    // they cannot see controls inside an invisible drawer at all.
    await expect(drawer.locator('button')).toHaveCount(0);
    await expect(drawer.getByTestId('product-mark')).toHaveCount(0);
    await expect(page.getByTestId('product-mark').filter({ visible: true })).toHaveCount(1);

    // Walk the whole tab order once: focus wraps to the body after the last
    // control, and nothing on the way may sit inside the drawer.
    const focused: string[] = [];
    for (let step = 0; step < 80; step += 1) {
      await page.keyboard.press('Tab');
      const where = await page.evaluate(() => {
        const node = document.activeElement;
        if (!node || node === document.body) return null;
        return node.closest('[data-testid="sidebar-mobile"]') ? 'drawer' : 'page';
      });
      if (where === null && focused.length > 0) break;
      if (where) focused.push(where);
    }
    expect(focused.length).toBeGreaterThan(0);
    expect(focused).not.toContain('drawer');
  });

  test('primary navigation is reachable and operable', async ({ page }) => {
    const isPhone = (page.viewportSize()?.width ?? 0) < 1024;

    // Below the lg breakpoint the rail is replaced by a drawer behind a toggle.
    // The drawer had no test hooks at all before #1320, so nothing could open
    // it and no test could reach any mobile layout.
    const nav = isPhone ? await openMobileSidebar(page) : sidebar(page, 'desktop');

    const devices = nav.getByTestId('nav-item-devices');
    await expect(devices).toBeVisible();

    // A control covered by the drawer's overlay or pushed off a 390px layout
    // fails here and nowhere else.
    await devices.click();
    await expect(page).toHaveURL(/\/devices$/);
    await expect(page.getByTestId('page-header-title')).toBeVisible();
  });

  test('completes the main journey: dashboard to a device list', async ({ page }) => {
    const isPhone = (page.viewportSize()?.width ?? 0) < 1024;

    const nav = isPhone ? await openMobileSidebar(page) : sidebar(page, 'desktop');
    await nav.getByTestId('nav-item-runtime').click();
    await expect(page).toHaveURL(/\/runtime$/);
    await expect(page.getByTestId('page-header-title')).toBeVisible();

    const backToDevices = isPhone ? await openMobileSidebar(page) : sidebar(page, 'desktop');
    await backToDevices.getByTestId('nav-item-devices').click();
    await expect(page).toHaveURL(/\/devices$/);
    await expect(page.getByTestId('page-header-title')).toBeVisible();
  });
});
