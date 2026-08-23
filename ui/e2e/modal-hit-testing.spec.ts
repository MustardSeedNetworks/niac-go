import { expect, test } from '@playwright/test';

/**
 * Modal hit-testing E2E — regression guard for D8.
 *
 * `ui/Modal.tsx` renders a full-viewport scrim button and the dialog as
 * siblings. The scrim is `position: absolute`; the dialog was `static`. CSS
 * paints positioned descendants above non-positioned ones regardless of DOM
 * order, so the scrim covered the dialog's own buttons: a real mouse click on
 * "Stop" hit the scrim's close handler and dismissed the dialog **without
 * stopping the simulation**.
 *
 * The important part of this guard is that it hit-tests. A test that asserts
 * the button exists, is visible, or is enabled passes against the broken code —
 * every one of those was true while the dialog was unclickable. Only
 * `document.elementFromPoint()` at the button's centre catches it.
 *
 * jsdom cannot express this: it has no layout engine, so this has to be an E2E
 * test rather than a Vitest one.
 */

/** Resolve what the browser would actually deliver a click to, at an element's centre. */
async function hitTestCentre(
  page: import('@playwright/test').Page,
  selector: string,
): Promise<{ isSelf: boolean; actual: string }> {
  return page.evaluate((sel) => {
    const el = document.querySelector(sel);
    if (!el) return { isSelf: false, actual: 'element-not-found' };
    const r = el.getBoundingClientRect();
    const hit = document.elementFromPoint(r.x + r.width / 2, r.y + r.height / 2);
    const isSelf = hit === el || el.contains(hit);
    const actual = isSelf
      ? 'self'
      : `${hit?.tagName ?? 'none'}[${hit?.getAttribute('aria-label') ?? hit?.className ?? ''}]`;
    return { isSelf, actual };
  }, selector);
}

test.describe('Modal buttons are reachable by mouse', () => {
  test('every button in an open ConfirmModal hit-tests to itself', async ({ page }) => {
    await page.goto('/debug');
    await page.waitForLoadState('domcontentloaded');

    // "Clear all logs" opens a ConfirmModal, which is the shared component
    // behind every destructive confirmation in the app.
    await page.locator('main button[aria-label="Clear all logs"]').click();
    await expect(page.locator('[role="dialog"]')).toBeVisible({ timeout: 10000 });

    const buttons = page.locator('[role="dialog"] button');
    const count = await buttons.count();
    expect(count).toBeGreaterThan(0);

    for (let i = 0; i < count; i++) {
      const label = (await buttons.nth(i).innerText()).trim() || `button ${i}`;
      const result = await hitTestCentre(page, `[role="dialog"] button:nth-of-type(${i + 1})`);
      expect(result.isSelf, `dialog button "${label}" is covered by ${result.actual}`).toBe(true);
    }
  });

  test('the dialog paints above its own backdrop', async ({ page }) => {
    await page.goto('/debug');
    await page.waitForLoadState('domcontentloaded');
    await page.locator('main button[aria-label="Clear all logs"]').click();
    await expect(page.locator('[role="dialog"]')).toBeVisible({ timeout: 10000 });

    // The dialog must establish its own positioned context; a `static` dialog
    // loses to the absolutely-positioned scrim no matter the DOM order.
    const position = await page
      .locator('[role="dialog"]')
      .evaluate((el) => getComputedStyle(el).position);
    expect(position, 'dialog must be positioned to beat the absolute scrim').not.toBe('static');
  });
});
