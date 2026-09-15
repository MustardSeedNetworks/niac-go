import { expect, test } from '@playwright/test';
import { sidebar } from './support/sidebar';

/**
 * Every page in `pageRegistry.ts` is `lazy()`, and the route table lives in a
 * descendant `<Routes>` under one `<Suspense>` in `App.tsx`. React Router wraps
 * its state update in a transition, and a transition that suspends keeps the
 * ENTIRE previous tree on screen — with no fallback — until the chunk arrives.
 *
 * Meanwhile the router has already pushed the new URL synchronously
 * (`completeNavigation` calls `history.push` before React commits). So there is
 * a window, as long as the chunk takes, in which the address bar says one page
 * and the rail, the page header and the page body are all still the previous
 * one. To an operator that reads as "the tap did nothing"; to a test it reads
 * as a layout that is about to move under the next click, which is how
 * `app-shell.mobile.spec.ts` came to click a nav item and go nowhere (#2151).
 *
 * The window is invisible locally because chunks come off localhost in single
 * milliseconds. Holding the response is what makes it deterministic.
 */
test.describe('route transitions', () => {
  test('the shell follows the URL while the destination chunk is still loading', async ({
    page,
  }) => {
    await page.goto('/');
    const rail = sidebar(page, 'desktop');
    await expect(rail.getByTestId('nav-item-root')).toHaveAttribute('aria-current', 'page');

    // Hold every script fetched from here on. The destination page is the only
    // thing this session still has to load, so this is its module.
    let release = (): void => {};
    const held = new Promise<void>((resolve) => {
      release = resolve;
    });
    await page.route('**/*.js', async (route) => {
      await held;
      await route.continue();
    });

    try {
      await rail.getByTestId('nav-item-devices').click();

      // The URL advances synchronously...
      await expect(page).toHaveURL(/\/devices$/);

      // ...so the shell has to agree with it, either by rendering the
      // destination or by showing the loading state. What it must not do is
      // keep presenting the previous page as the current one.
      await expect(rail.getByTestId('nav-item-devices')).toHaveAttribute('aria-current', 'page');
      await expect(rail.getByTestId('nav-item-root')).not.toHaveAttribute('aria-current', 'page');
    } finally {
      release();
    }
  });
});
