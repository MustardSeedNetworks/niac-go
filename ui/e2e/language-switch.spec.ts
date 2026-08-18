import { expect, test } from '@playwright/test';

/**
 * Language switching E2E
 *
 * Confirms the localStorage-driven locale handoff between i18next and
 * the browser actually round-trips:
 *
 * - The Dashboard page renders English by default.
 * - Setting `niac-language` in localStorage to `es` and reloading
 *   produces Spanish strings (per ES locale JSON shipped with the
 *   binary).
 * - The `<html lang>` attribute flips to match (WCAG 3.1.1/3.1.2).
 * - Pluralized strings honor the count parameter on both sides
 *   (Phase 5 `common.plurals.deviceCount_one/_other`).
 *
 * Markers used for the Spanish assertions are taken from the
 * Translation Memory (msn-docs-internal/05-Engineering/
 * I18N_TRANSLATION_MEMORY.md) — the canonical translations for the
 * smoke-tested labels. If a label here ever needs to change, update
 * the TM and this spec in lockstep.
 */

const LOCAL_STORAGE_KEY = 'niac-language';

test.describe('Language switching', () => {
  test.beforeEach(async ({ page }) => {
    // Start from a clean storage state so the default-detection path
    // is what each test exercises (rather than carried-over preferences
    // from another spec file in the same run).
    await page.goto('/');
    await page.evaluate((key) => localStorage.removeItem(key), LOCAL_STORAGE_KEY);
  });

  test('renders English by default and sets <html lang="en">', async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');

    const htmlLang = await page.locator('html').getAttribute('lang');
    expect(htmlLang).toBe('en');

    // Two stable EN markers: the Dashboard nav label (sidebar) and the
    // page-header h1. Both come from pages.dashboard.{label,title} —
    // changing them should require updating this test AND the TM.
    //
    // The canonical Sidebar renders its body in both a mobile aside
    // (`lg:hidden`) and a desktop aside (`hidden lg:flex`). At the default
    // 1280px viewport the mobile aside is display:none, so `.first()`
    // matches a hidden node. Use `.last()` for the visible desktop aside.
    await expect(page.getByText(/^Dashboard$/).last()).toBeVisible();
  });

  test('flips to Spanish when localStorage is set to es', async ({ page }) => {
    // Set the language preference before the i18next bundle bootstraps.
    // `addInitScript` runs in the new page context before any user code.
    await page.addInitScript(
      ({ key }) => {
        localStorage.setItem(key, 'es');
      },
      { key: LOCAL_STORAGE_KEY },
    );

    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');

    // <html lang> must reflect the active locale for accessibility tools.
    await expect.poll(async () => page.locator('html').getAttribute('lang')).toBe('es');

    // ES marker: Dashboard's translated label is "Panel" per
    // pages.dashboard.label in es/pages.json + the TM. The sidebar
    // shows this label, and so does the page header title.
    // See above for why `.last()` rather than `.first()`.
    await expect(page.getByText(/Panel/).last()).toBeVisible();
  });

  test('translates the dashboard rollup', async ({ page }) => {
    // The device count is a rollup figure now — a bare number with a separate
    // label — so this page no longer renders a pluralised count string, and
    // the plural forms themselves are covered by i18n.plurals.test.ts, which
    // does not depend on daemon state. What is worth asserting here is that
    // the rollup's own copy is translated.
    await page.addInitScript(
      ({ key }) => {
        localStorage.setItem(key, 'es');
      },
      { key: LOCAL_STORAGE_KEY },
    );
    await page.goto('/');
    await page.waitForLoadState('domcontentloaded');

    await expect
      .poll(async () => page.evaluate((key) => localStorage.getItem(key), LOCAL_STORAGE_KEY))
      .toBe('es');
    await expect(page.getByTestId('status-rollup')).toBeVisible();
    await expect(page.getByTestId('status-rollup')).not.toContainText(/dashboard\.rollup\./);
  });
});
