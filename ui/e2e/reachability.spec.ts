import { expect, test } from '@playwright/test';

/**
 * Mouse-reachability guards for Wave 6 (D1, D14).
 *
 * Both defects were controls that existed, were visible and were enabled — and
 * could not be clicked, because something else painted on top or the container
 * clipped them. Presence assertions pass against all of that; only geometry and
 * hit-testing catch it.
 */

/** True when a click at the element's centre would actually reach it. */
async function reachable(
  page: import('@playwright/test').Page,
  handle: import('@playwright/test').Locator,
): Promise<boolean> {
  return handle.evaluate((el) => {
    const r = el.getBoundingClientRect();
    if (r.width === 0 || r.height === 0) return false;
    // Off-screen entirely?
    if (r.right <= 0 || r.bottom <= 0 || r.left >= innerWidth || r.top >= innerHeight) return false;
    const hit = document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2);
    return hit === el || el.contains(hit) || Boolean(hit && hit.contains(el));
  });
}

test.describe('Help drawer tabs (D1)', () => {
  test('all tabs are on-screen and clickable', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');
    await page.locator('button[aria-label="Open help"]').last().click();

    const tabs = page.getByRole('tab');
    await expect(tabs.first()).toBeVisible({ timeout: 10000 });

    const count = await tabs.count();
    expect(count, 'help drawer should expose all of its tabs').toBeGreaterThanOrEqual(8);

    for (let i = 0; i < count; i++) {
      const tab = tabs.nth(i);
      const label = (await tab.innerText()).trim() || `tab ${i}`;
      expect(await reachable(page, tab), `tab "${label}" is not reachable by mouse`).toBe(true);
    }
  });

  test('a clipped tab actually activates when clicked', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');
    await page.locator('button[aria-label="Open help"]').last().click();

    // FAQ is the last tab — the one that used to sit furthest outside the drawer.
    const faq = page.getByRole('tab').last();
    await faq.click();
    await expect(faq).toHaveAttribute('aria-selected', 'true');
  });
});

test.describe('Topology export menu (D14)', () => {
  /**
   * The export dropdown lives in the header Card; the graph lives in a sibling
   * Card below it. Card's default variant sets `backdrop-blur-xl`, which makes
   * each Card its own stacking context — so the dropdown's own `z-50` could not
   * be compared against the graph Card at all, and the graph Card (later in the
   * DOM) painted its whole layer over the menu's last item.
   *
   * This asserts the stacking relationship rather than clicking a menu item,
   * because the E2E daemon runs with no simulation and the topology page has no
   * graph data to open the menu against. The relationship is the invariant the
   * fix establishes, and it holds with or without data.
   */
  test('the header card out-stacks the graph card', async ({ page }) => {
    await page.goto('/topology');
    await page.waitForLoadState('domcontentloaded');
    await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });

    const order = await page.evaluate(() => {
      const cards = [...document.querySelectorAll('main [class*="backdrop-blur"]')];
      if (cards.length < 2) return null;
      const z = (el: Element) => {
        const raw = getComputedStyle(el).zIndex;
        return raw === 'auto' ? 0 : Number(raw);
      };
      // Header card is the first, the graph card follows it in the DOM.
      return { header: z(cards[0]), graph: z(cards[cards.length - 1]) };
    });

    expect(order, 'expected a header card and a graph card on the topology page').not.toBeNull();
    expect(
      order?.header,
      'header card must paint above the graph card, or the export menu is swallowed',
    ).toBeGreaterThan(order?.graph ?? 0);
  });
});
