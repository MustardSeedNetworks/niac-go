import type { Page } from '@playwright/test';

/**
 * Shared E2E helpers — niac flavor.
 *
 * niac is loopback-only by default and has NO auth surface (no token,
 * no login form), so the helpers here are a smaller subset of the seed
 * / stem siblings. The file shape (TEST_CREDENTIALS, AUTH_STORAGE_STATE,
 * disableAnimations) is kept parallel so spec patterns port across the
 * three repos verbatim per the cross-repo `feedback_harmonization_no_master`
 * convention.
 *
 * Each repo owns its own copy — do not import from seed/stem.
 */

/**
 * Placeholder for shape parity with seed/stem. niac doesn't drive a
 * login form, but specs that want to mirror the sibling pattern can
 * still reference this constant rather than hard-coding a value.
 */
export const TEST_CREDENTIALS = {
  username: 'admin',
  password: 'niac-loopback-no-auth-needed', // gitleaks:allow — placeholder fixture; niac has no login form, no real credential here
} as const;

/**
 * Path (relative to `ui/`) where global-setup persists the
 * (empty-but-disableAnimations-armed) storage state. Mirrors the
 * seed/stem path so per-spec storageState overrides use the same
 * literal.
 */
export const AUTH_STORAGE_STATE = 'playwright/.auth/user.json';

/**
 * Disable CSS transitions/animations on the page. The dropdown,
 * drawer, and topology-graph transitions otherwise race Playwright's
 * scroll-into-view + actionability checks under parallel CI load,
 * causing intermittent click failures on deep elements. Must be
 * called before `page.goto()` so the style installs on every
 * navigation.
 */
export async function disableAnimations(page: Page): Promise<void> {
  await page.addInitScript(() => {
    const style = document.createElement('style');
    style.textContent =
      '*,*::before,*::after{transition:none!important;animation:none!important;scroll-behavior:auto!important}';
    document.documentElement.appendChild(style);
  });
}
