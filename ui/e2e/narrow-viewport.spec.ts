import { expect, test } from '@playwright/test';

/**
 * Narrow-viewport overflow guard (#1483).
 *
 * A grid item defaults to `min-width: auto`, so it refuses to shrink below its
 * content width; the track is sized to the content and the whole document
 * scrolls sideways, carrying controls off-screen. Measured on CT304 at 390px:
 * /devices overflowed by 250px, /segments by 115px, /runtime by 83px.
 *
 * /runtime is the clearest case: its scenarios table is already wrapped in a
 * `div.overflow-x-auto`, which is the right structure — but that wrapper is
 * itself a grid item that cannot shrink, so instead of the table scrolling
 * inside its container, the container stretched and the page scrolled.
 *
 * Page-level horizontal scroll is the assertion because it is the thing the
 * user feels; wide tables are expected to scroll within their own container.
 */

const ROUTES = ['/runtime', '/devices', '/segments'];
const WIDTHS = [
  { name: 'phone', width: 390, height: 844 },
  { name: 'tablet', width: 820, height: 1180 },
];

for (const viewport of WIDTHS) {
  test.describe(`${viewport.name} (${viewport.width}px)`, () => {
    test.use({ viewport: { width: viewport.width, height: viewport.height } });

    for (const route of ROUTES) {
      test(`${route} does not scroll horizontally`, async ({ page }) => {
        await page.goto(route);
        await page.waitForLoadState('domcontentloaded');
        await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 15000 });

        const overflow = await page.evaluate(() => {
          const de = document.documentElement;
          return de.scrollWidth - de.clientWidth;
        });
        expect(overflow, `${route} overflows the viewport by ${overflow}px`).toBeLessThanOrEqual(0);
      });

      test(`${route} keeps its controls on screen`, async ({ page }) => {
        await page.goto(route);
        await page.waitForLoadState('domcontentloaded');
        await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 15000 });

        const offscreen = await page.evaluate(() => {
          // A control inside a horizontally scrollable container is reachable by
          // scrolling that container, which is the intended design for a wide
          // table. Only controls the page itself pushes out of reach count.
          const inScrollContainer = (el: Element): boolean => {
            let n = el.parentElement;
            while (n && n !== document.body) {
              const overflowX = getComputedStyle(n).overflowX;
              if (overflowX === 'auto' || overflowX === 'scroll') return true;
              n = n.parentElement;
            }
            return false;
          };
          return [...document.querySelectorAll('main button, main a')]
            .filter((el) => {
              const r = el.getBoundingClientRect();
              if (r.width === 0) return false;
              if (inScrollContainer(el)) return false;
              return r.left < -2 || r.right > window.innerWidth + 2;
            })
            .map((el) => (el.textContent || '').trim().slice(0, 30));
        });
        expect(offscreen, `controls outside the viewport on ${route}`).toEqual([]);
      });
    }
  });
}
