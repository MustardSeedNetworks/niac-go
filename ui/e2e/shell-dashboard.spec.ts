import { expect, test } from '@playwright/test';
import { expectPageFirst } from './support/shell-layout';

for (const viewport of [
  { width: 1440, height: 900 },
  { width: 390, height: 844 },
]) {
  for (const theme of ['dark', 'light']) {
    test(`dashboard shell ${theme} at ${viewport.width}px`, async ({ page }, testInfo) => {
      await page.setViewportSize(viewport);
      await page.addInitScript((value) => localStorage.setItem('niac-theme', value), theme);
      await page.goto('/');
      await expect(page).toHaveTitle('NIAC');
      await expectPageFirst(page);
      await expect(page.locator('html')).toHaveClass(theme === 'dark' ? /dark/ : /^(?!.*dark)/);
      await testInfo.attach(`dashboard-${theme}-${viewport.width}`, {
        body: await page.screenshot({ fullPage: true }),
        contentType: 'image/png',
      });
    });
  }
}
