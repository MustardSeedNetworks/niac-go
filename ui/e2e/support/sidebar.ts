import type { Locator, Page } from '@playwright/test';

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

/**
 * Opens the mobile drawer via the top bar's toggle and waits for it to finish
 * sliding in. The drawer is translated off-screen rather than unmounted, so
 * `toBeVisible()` is true before the transition completes and a click can land
 * on a control that is still moving.
 */
export async function openMobileSidebar(page: Page): Promise<Locator> {
  const drawer = sidebar(page, 'mobile');
  await page.getByTestId('mobile-menu-toggle').click();
  await drawer.waitFor({ state: 'visible' });
  return drawer;
}
