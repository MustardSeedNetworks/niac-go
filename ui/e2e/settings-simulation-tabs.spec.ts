import { expect, test } from '@playwright/test';
import { openMobileSidebar, sidebar } from './support/sidebar';

for (const width of [1280, 390]) {
  test.describe(`Simulation settings tabs at ${width}px`, () => {
    test.use({ viewport: { width, height: 844 } });

    test('@smoke names tabs and navigates the configuration sources with the keyboard', async ({
      page,
    }) => {
      await page.goto('/');
      const nav = width < 1024 ? await openMobileSidebar(page) : sidebar(page, 'desktop');
      await nav.getByTestId('sidebar-settings-button').click();
      const drawer = page.getByTestId('settings-drawer');
      await expect
        .poll(() =>
          drawer.evaluate((element) => {
            const { left, right } = element.getBoundingClientRect();
            return left >= 0 && right <= window.innerWidth;
          }),
        )
        .toBe(true);
      await expect
        .poll(() => drawer.evaluate((element) => element.scrollWidth <= element.clientWidth))
        .toBe(true);
      const navigation = drawer.getByRole('navigation');
      await expect
        .poll(() =>
          navigation.evaluate((element) => {
            const scroller = element.parentElement;
            if (!scroller) throw new Error('Missing settings navigation scroller');
            return scroller.scrollHeight - scroller.clientHeight;
          }),
        )
        .toBe(0);
      await drawer.getByRole('tab', { name: 'About', exact: true }).click();
      await expect(drawer.getByRole('tab', { name: 'About', exact: true })).toHaveAttribute(
        'aria-selected',
        'true',
      );
      await drawer.getByRole('tab', { name: 'Simulation', exact: true }).click();
      const tabs = drawer.getByRole('tablist', { name: 'Configuration' });
      const templates = tabs.getByRole('tab', { name: 'Templates', exact: true });
      const configs = tabs.getByRole('tab', { name: 'My Configs', exact: true });
      const upload = tabs.getByRole('tab', { name: 'Upload', exact: true });
      await expect(templates).toHaveAttribute('aria-selected', 'true');
      await expect(configs).toHaveAccessibleName('My Configs');
      await expect(upload).toHaveAccessibleName('Upload');
      await page.screenshot({ path: test.info().outputPath('simulation-tabs.png') });
      await templates.focus();
      await templates.press('ArrowRight');
      await expect(configs).toBeFocused();
      await expect(configs).toHaveAttribute('aria-selected', 'true');
      await expect(drawer.getByRole('tabpanel', { name: 'My Configs' })).toBeVisible();
      await configs.press('Tab');
      await expect(drawer.getByRole('tabpanel', { name: 'My Configs' })).toBeFocused();
      await configs.focus();
      await configs.press('End');
      await expect(upload).toBeFocused();
      await upload.press('ArrowRight');
      await expect(templates).toBeFocused();
      await templates.press('ArrowLeft');
      await expect(upload).toBeFocused();
      await upload.press('Home');
      await expect(templates).toBeFocused();
      await templates.press('Tab');
      await expect(drawer.getByRole('tabpanel', { name: 'Templates' })).toBeFocused();
    });
  });
}
