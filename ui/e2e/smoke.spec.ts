import { expect, test } from '@playwright/test';
import { AUTH_STORAGE_STATE } from './helpers/auth';
import { sidebar } from './support/sidebar';

const VERSION_KEYS = ['version', 'commit', 'buildTime', 'uiBuildHash'] as const;

test.describe('smoke @ unauthenticated', { tag: '@smoke' }, () => {
  test.use({ storageState: { cookies: [], origins: [] } });

  test('GET /__version returns canonical build metadata', async ({ request }) => {
    const res = await request.get('/__version');
    expect(res.status()).toBe(200);
    const body = await res.json();
    for (const k of VERSION_KEYS) {
      expect(body[k], `missing ${k} in /__version`).toBeTruthy();
      expect(typeof body[k]).toBe('string');
    }
  });
});

test.describe('smoke @ authenticated', { tag: '@smoke' }, () => {
  test.use({ storageState: AUTH_STORAGE_STATE });

  test('NIAC shell renders with page-header-title', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });
  });

  test('settings drawer opens and closes', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });
    await sidebar(page).getByTestId('sidebar-settings-button').click();
    await expect(page.getByTestId('settings-drawer')).toBeVisible();
    await page.getByTestId('settings-drawer-close').click();
    await expect(page.getByTestId('settings-drawer')).toBeHidden();
  });

  test('help drawer opens with NIAC version badge', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });
    await sidebar(page).getByTestId('sidebar-help-button').click();
    await expect(page.getByTestId('help-drawer')).toBeVisible();
    await expect(page.getByTestId('help-drawer-version')).toContainText(/NIAC v.+/);
    await page.getByTestId('help-drawer-close').click();
    await expect(page.getByTestId('help-drawer')).toBeHidden();
  });
});
