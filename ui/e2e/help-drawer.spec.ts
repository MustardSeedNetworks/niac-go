import { expect, test } from '@playwright/test';
import { disableAnimations } from './helpers/auth';
import { sidebar } from './support/sidebar';

/**
 * Help Drawer — version-badge contract test.
 *
 * niac PR #774 added a `NIAC v<version>` badge to the HelpDrawer
 * header sourced from /__version. This spec asserts the user-visible
 * surface: opening the drawer from the sidebar reveals a version badge
 * matching `/NIAC v.+/`, and the close affordance dismisses it.
 *
 * The testids consumed here ship on `main` from PR-C #774
 * (HelpDrawer.tsx:91/106/115) and from niac E2E N3 (Sidebar.tsx
 * FooterIconButton's new `data-testid` prop).
 */

test.describe('Help Drawer', () => {
  test.beforeEach(async ({ page }) => {
    await disableAnimations(page);
    await page.goto('/');
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 10000,
    });
  });

  test('clicking the sidebar Help button opens the drawer', async ({ page }) => {
    await sidebar(page).getByTestId('sidebar-help-button').click();
    await expect(page.getByTestId('help-drawer')).toBeVisible();
  });

  test('the version badge renders and matches /NIAC v.+/', async ({ page }) => {
    await sidebar(page).getByTestId('sidebar-help-button').click();
    const badge = page.getByTestId('help-drawer-version');
    await expect(badge).toBeVisible();
    await expect(badge).toHaveText(/NIAC v.+/);
  });

  test('clicking the close affordance dismisses the drawer', async ({ page }) => {
    await sidebar(page).getByTestId('sidebar-help-button').click();
    const drawer = page.getByTestId('help-drawer');
    await expect(drawer).toBeVisible();

    await page.getByTestId('help-drawer-close').click();
    await expect(drawer).not.toBeVisible();
  });
});
