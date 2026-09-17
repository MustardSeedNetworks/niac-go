import { devices, expect, test } from '@playwright/test';
import { checkLongHistoryLayout } from './support/history-layout';
import { openMobileSidebar, sidebar } from './support/sidebar';

/**
 * The small-screen smoke subset.
 *
 * It covers the three things a viewport can break that nothing else here would
 * notice: the shell renders, the primary navigation is reachable and operable,
 * and the main journey completes.
 *
 * The device preset lives here rather than in a project of its own. There used
 * to be tablet-safari / mobile-chrome / mobile-safari projects, but
 * E2E_CONVENTIONS allows chromium and webkit only across all four products
 * (#2246). What that policy bans is browser projects, not device emulation —
 * and the reason the presets existed is still right: a narrow window is not a
 * phone, because the user agent, touch support and input modality are what
 * decide whether a control is reachable at all. So this file keeps a real
 * preset, Pixel 7, which is a Chromium device and therefore runs under the
 * chromium project. The webkit project ignores this file.
 *
 * Deliberately narrow. The full suite stays on desktop because most of what it
 * asserts is viewport-independent, and running all 26 spec files on more
 * devices would multiply E2E wall-clock for very little signal (#1320).
 */
test.use({ ...devices['Pixel 7'] });

test.describe('app shell on small screens', () => {
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
