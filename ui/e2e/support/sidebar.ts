import type { Locator, Page } from '@playwright/test';
import { expect } from '@playwright/test';

/**
 * The sidebar is mounted twice — a mobile drawer and a desktop rail — and both
 * stay in the DOM at every viewport, because the responsive classes toggle
 * display rather than mount. Every sidebar testid therefore exists twice, and
 * an unscoped getByTestId('sidebar-help-button') trips strict mode.
 *
 * So a spec has to say which surface it means. That is the point: a test that
 * does not know whether it is driving the phone drawer or the desktop rail is
 * not testing either of them.
 */
export type SidebarSurface = 'mobile' | 'desktop';

/** The sidebar container for one surface; chain getByTestId off it. */
export function sidebar(page: Page, surface: SidebarSurface = 'desktop'): Locator {
  return page.getByTestId(`sidebar-${surface}`);
}

/** The drawer's left edge; negative while it is parked off-screen. */
async function drawerEdge(drawer: Locator): Promise<number> {
  return (await drawer.boundingBox())?.x ?? Number.NEGATIVE_INFINITY;
}

/**
 * Opens the mobile drawer via the top bar's toggle, and does not return until
 * it has finished sliding in.
 *
 * Both waits here are load-bearing, and each replaced something that looked
 * equivalent and was not:
 *
 * 1. `waitFor({ state: 'visible' })` is true from the first frame. The drawer is
 *    translated off-screen, not unmounted, so Playwright calls it visible while
 *    it sits at -translate-x-full for the 300ms the transition runs. Clicking a
 *    control inside it then fails with "element is outside of the viewport".
 *
 * 2. Clicking the toggle unconditionally races the route-change close. The
 *    toggle FLIPS state, and navigating closes the drawer via an effect on
 *    location.pathname — so a click that lands before React has processed that
 *    close flips open -> closed, the effect then keeps it closed, and the
 *    drawer never appears. That is a 2-7% failure under parallel load, which on
 *    a flake budget of 0 is a red job.
 *
 * So: wait until it is definitively closed, then open it, then wait until it
 * has arrived. Waiting on the real state at each step rather than on a proxy
 * that is true too early.
 */
export async function openMobileSidebar(page: Page): Promise<Locator> {
  const drawer = sidebar(page, 'mobile');

  await expect
    .poll(() => drawerEdge(drawer), {
      message: 'mobile drawer never settled closed, so the toggle would re-close it',
    })
    .toBeLessThan(0);

  await page.getByTestId('mobile-menu-toggle').click();

  await expect
    .poll(() => drawerEdge(drawer), {
      message: 'mobile drawer never finished sliding into view',
    })
    .toBeGreaterThanOrEqual(0);

  return drawer;
}
