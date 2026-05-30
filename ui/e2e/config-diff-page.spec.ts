import { expect, test } from '@playwright/test';

/**
 * Config Diff Page (/config-diff) E2E
 *
 * Covers the compare-and-merge surface:
 * - Page renders the "Compare & Merge" heading
 * - Original File + Modified File panels are present
 */

test.describe('Config Diff Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/config-diff');
    await page.waitForLoadState('domcontentloaded');
  });

  test('should render the Compare & Merge heading', async ({ page }) => {
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 10000,
    });
  });

  test('should land on the /config-diff route', async ({ page }) => {
    await expect(page).toHaveURL(/\/config-diff$/);
  });

  test('should render both Original File and Modified File panels', async ({ page }) => {
    // diff-panel-original-title and diff-panel-modified-title are on
    // ConfigDiffPage.tsx. Previously matched by /original file/i and
    // /modified file/i regex — both translated under es.
    await expect(page.getByTestId('diff-panel-original-title')).toBeVisible({
      timeout: 5000,
    });
    await expect(page.getByTestId('diff-panel-modified-title')).toBeVisible({
      timeout: 5000,
    });
  });
});
