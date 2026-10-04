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
      const interfacePicker = drawer.getByTestId('simulation-interface');
      await expect(interfacePicker).toHaveCSS('appearance', 'none');
      expect(
        await interfacePicker.evaluate((element) => element.getBoundingClientRect().height),
      ).toBeGreaterThanOrEqual(44);
      const tabs = drawer.getByRole('tablist', { name: 'Configuration' });
      // The library is the one list of starter networks; the Built-in tab
      // listed nothing on an installed host and is gone (#2131).
      await expect(tabs.getByRole('tab')).toHaveText(['My Configs', 'Upload']);
      const configs = tabs.getByRole('tab', { name: 'My Configs', exact: true });
      const upload = tabs.getByRole('tab', { name: 'Upload', exact: true });
      await expect(configs).toHaveAttribute('aria-selected', 'true');
      await expect(upload).toHaveAccessibleName('Upload');
      // First run seeds the library with the shipped starters, so the default
      // tab is never empty.
      const configsPanel = drawer.getByRole('tabpanel', { name: 'My Configs' });
      await expect(configsPanel.getByRole('button', { name: /small-office/ })).toBeVisible();
      await page.screenshot({ path: test.info().outputPath('simulation-tabs.png') });
      await configs.focus();
      await configs.press('ArrowRight');
      await expect(upload).toBeFocused();
      await expect(upload).toHaveAttribute('aria-selected', 'true');
      await expect(drawer.getByRole('tabpanel', { name: 'Upload' })).toBeVisible();
      await upload.press('Tab');
      await expect(drawer.getByRole('tabpanel', { name: 'Upload' })).toBeFocused();
      await upload.focus();
      await upload.press('ArrowRight');
      await expect(configs).toBeFocused();
      await configs.press('ArrowLeft');
      await expect(upload).toBeFocused();
      await upload.press('Home');
      await expect(configs).toBeFocused();
      await configs.press('End');
      await expect(upload).toBeFocused();
      await upload.press('Home');
      await configs.press('Tab');
      await expect(configsPanel).toBeFocused();
    });
  });
}
