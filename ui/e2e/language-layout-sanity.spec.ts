import { expect, type Page, test } from '@playwright/test';

/**
 * Language layout sanity (ES overflow detection)
 *
 * Spanish UI strings run on average ~30% longer than English. The
 * unit tests, key-resolution tests, and language-switch.spec only
 * verify that ES text RENDERS — they don't catch when "Inyección de
 * errores" overflows a button sized for "Error Injection", or when
 * "Verificaciones de salud" clips the column header.
 *
 * This spec exercises a small fixed set of high-traffic views in ES
 * and asserts that key text containers don't overflow their parents
 * (causing clipping, mid-word wrap, or horizontal scrollbars). It's
 * an alternative to full visual-regression snapshots — much lighter
 * to maintain (no per-route PNG baselines that bit-rot on CSS
 * tweaks) and catches the specific class of bug that ES introduces.
 *
 * For full pixel-diff regression we'd need:
 * - `expect(page).toHaveScreenshot()` calls + per-route .png
 *   baselines committed to the repo
 * - `playwright test --update-snapshots` workflow in CI
 * - Snapshot storage for the inevitable churn as components evolve
 *
 * That's queued for a separate workstream when the UI shape settles
 * past the current rapid-iteration phase.
 */

const LOCAL_STORAGE_KEY = 'niac-language';

const setLanguage = async (page: Page, lang: 'en' | 'es'): Promise<void> => {
  await page.addInitScript(
    ({ key, value }) => {
      localStorage.setItem(key, value);
    },
    { key: LOCAL_STORAGE_KEY, value: lang },
  );
};

const noHorizontalScroll = async (page: Page): Promise<void> => {
  const { scrollWidth, clientWidth } = await page.evaluate(() => ({
    scrollWidth: document.documentElement.scrollWidth,
    clientWidth: document.documentElement.clientWidth,
  }));
  // Allow 1px tolerance for subpixel rendering. Anything more is a
  // real overflow that produces a horizontal scrollbar.
  expect(scrollWidth - clientWidth).toBeLessThan(2);
};

const noClippedText = async (page: Page, selector: string): Promise<void> => {
  // For each matching element, compare scroll width to client width.
  // If scrollWidth > clientWidth, the text is wider than its container
  // and is being clipped (typically by `text-overflow: ellipsis` or
  // `overflow: hidden`).
  const clipped = await page.locator(selector).evaluateAll((els: Element[]) =>
    els
      .filter((el) => {
        const e = el as HTMLElement;
        // Only count elements that actually have visible text.
        return e.scrollWidth > e.clientWidth && e.innerText.trim().length > 0;
      })
      .map((el) => (el as HTMLElement).innerText.trim().slice(0, 80)),
  );
  expect(clipped, `${selector} elements with clipped text:\n  ${clipped.join('\n  ')}`).toEqual([]);
};

test.describe('Language layout sanity', () => {
  for (const lang of ['en', 'es'] as const) {
    test.describe(`${lang.toUpperCase()} dashboard`, () => {
      test.beforeEach(async ({ page }) => {
        await setLanguage(page, lang);
        await page.goto('/');
        await page.waitForLoadState('domcontentloaded');
      });

      test('no horizontal scrollbar on dashboard', async ({ page }) => {
        await noHorizontalScroll(page);
      });

      test("sidebar nav labels don't get clipped", async ({ page }) => {
        // Nav labels are the highest-risk surface for ES overflow:
        // "Compare & Merge" → "Comparar y fusionar" (~30% wider),
        // "Walks" → "Recorridos" (~70% wider).
        await noClippedText(page, 'nav a, aside a, [role="navigation"] a');
      });

      test("page header h1/h2 don't get clipped", async ({ page }) => {
        await noClippedText(page, 'h1, h2');
      });
    });
  }
});
