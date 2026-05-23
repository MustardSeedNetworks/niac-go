import { expect, test } from '@playwright/test';

/**
 * Library File Pages (/library/walks and /library/pcaps) E2E
 *
 * Covers the SNMP walk + PCAP library browsers. Same React component
 * (LibraryFilesView) wrapped twice in pageRegistry with different
 * `kind` prop.
 */

test.describe('Library — SNMP Walks', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/library/walks');
    await page.waitForLoadState('domcontentloaded');
  });

  test('should render the SNMP Walks heading', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /snmp walks/i })).toBeVisible({
      timeout: 10000,
    });
  });

  test('should land on the /library/walks route', async ({ page }) => {
    await expect(page).toHaveURL(/\/library\/walks$/);
  });

  test('should render a search input or empty-state hint', async ({ page }) => {
    // Either there are walks (search input rendered) or the page shows
    // the empty-state hint about dropping files into ~/.niac/library/walks/.
    const content = page
      .getByPlaceholder(/search/i)
      .or(page.getByText(/drop walk files|niac content install|library\/walks/i));
    await expect(content.first()).toBeVisible({ timeout: 5000 });
  });
});

test.describe('Library — PCAP Captures', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/library/pcaps');
    await page.waitForLoadState('domcontentloaded');
  });

  test('should render the PCAP Captures heading', async ({ page }) => {
    await expect(page.getByRole('heading', { name: /pcap captures/i })).toBeVisible({
      timeout: 10000,
    });
  });

  test('should land on the /library/pcaps route', async ({ page }) => {
    await expect(page).toHaveURL(/\/library\/pcaps$/);
  });

  test('should render a search input or empty-state hint', async ({ page }) => {
    const content = page
      .getByPlaceholder(/search/i)
      .or(page.getByText(/drop pcap files|niac content install|library\/pcaps/i));
    await expect(content.first()).toBeVisible({ timeout: 5000 });
  });
});
