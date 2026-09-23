import enPages from '@locales/en/pages.json' with { type: 'json' };
import { expect, test } from '@playwright/test';
import { expectPageFirst } from './support/shell-layout';

/**
 * Every route names itself the same way twice (UI-NIAC-5, niac-go#2189).
 *
 * The breadcrumb trail used to read from a hand-kept map of 11 of the 16
 * routes, so /new-simulation, /segments, /walk-analyzer, /library/walks and
 * /library/pcaps fell through to the raw URL slug — lowercase, hyphenated and
 * English-only — while the rail and the page header showed a translated name
 * for the same page. document.title was never set per route at all, so the
 * browser history, a bookmark and a second window were indistinguishable.
 *
 * Both now come from `pageRegistry`, and this spec is the ratchet: it asserts
 * the rendered trail and the tab against the locale catalog rather than
 * against a second list, so adding a route with no label fails here.
 */

/** Every registry route, with the pages.*.label the catalog gives it. */
const ROUTES = [
  { path: '/runtime', label: enPages.runtime.label },
  { path: '/new-simulation', label: enPages.newSimWizard.label },
  { path: '/devices', label: enPages.devices.label },
  { path: '/segments', label: enPages.segments.label },
  { path: '/device-config', label: enPages.deviceLibrary.label },
  { path: '/topology', label: enPages.topology.label },
  { path: '/alerts', label: enPages.alerts.label },
  { path: '/traffic', label: enPages.traffic.label },
  { path: '/debug', label: enPages.debug.label },
  { path: '/packets', label: enPages.packets.label },
  { path: '/config-diff', label: enPages.configDiff.label },
  { path: '/walk-validator', label: enPages.walkValidator.label.replace('{{protocol}}', 'SNMP') },
  { path: '/walk-analyzer', label: enPages.walkAnalyzer.label },
  { path: '/library/walks', label: enPages.libraryWalks.label },
  { path: '/library/pcaps', label: enPages.libraryPcaps.label.replace('{{format}}', 'PCAP') },
] as const;

/** A crumb an operator reads: no lowercase slug, no hyphen standing for a space. */
const TITLE_CASE = /^[A-Z0-9]/;

for (const viewport of [
  { width: 1440, height: 900 },
  { width: 390, height: 844 },
]) {
  for (const { path, label } of ROUTES) {
    test(`${path} names itself and starts with its page header at ${viewport.width}px`, async ({
      page,
    }) => {
      await page.setViewportSize(viewport);
      await page.goto(path);

      const crumbs = page.locator('nav[aria-label="Breadcrumb"] [data-crumb]');
      await expect(crumbs.last()).toHaveText(label);
      for (const text of await crumbs.allTextContents()) {
        expect(text, `${path} crumb "${text}"`).toMatch(TITLE_CASE);
        expect(text, `${path} crumb "${text}" is a URL slug`).not.toContain('-');
      }

      await expect(page).toHaveTitle(`${label} | ${enPages.app?.name ?? 'NIAC'}`);
      await expectPageFirst(page);
    });
  }
}

test('the dashboard is the product, with no trail above it', async ({ page }) => {
  await page.goto('/');
  await expect(page).toHaveTitle('NIAC');
  await expect(page.locator('nav[aria-label="Breadcrumb"]')).toHaveCount(0);
});
