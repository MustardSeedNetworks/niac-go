import { expect, test } from '@playwright/test';
import { openMobileSidebar, sidebar } from './support/sidebar';

/**
 * The small-screen smoke subset.
 *
 * Runs on tablet-safari, mobile-chrome and mobile-safari only — the desktop
 * projects ignore this file. It covers the three things a viewport can break
 * that nothing else here would notice: the shell renders, the primary
 * navigation is reachable and operable, and the main journey completes.
 *
 * Deliberately narrow. The full suite stays on desktop because most of what it
 * asserts is viewport-independent, and running all 26 spec files on three more
 * devices would multiply E2E wall-clock for very little signal (#1320).
 */
test.describe('app shell on small screens', () => {
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

  test('primary navigation is reachable and operable', async ({ page }) => {
    const isPhone = (page.viewportSize()?.width ?? 0) < 1024;

    // Below the lg breakpoint the rail is replaced by a drawer behind a toggle.
    // The drawer had no test hooks at all before #1320, so nothing could open
    // it and no test could reach any mobile layout.
    const nav = isPhone ? await openMobileSidebar(page) : sidebar(page, 'desktop');

    const devices = nav.getByTestId('nav-item-devices');
    await expect(devices).toBeVisible();

    // Tapping, not clicking: these projects emulate touch, and a control that
    // is covered by an overlay or below a 44px target fails here and nowhere
    // else.
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
