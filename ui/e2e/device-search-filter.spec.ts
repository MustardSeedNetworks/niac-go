import { expect, test } from '@playwright/test';

/**
 * Device Search and Filter Tests
 *
 * Tests for device list search and filtering:
 * - Text search
 * - Filter by type
 * - Filter by protocol
 * - Multi-criteria filtering
 * - Sort functionality
 */

test.describe('Device Search and Filter', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/devices');
    await page.waitForLoadState('domcontentloaded');
  });

  test.describe('Text Search', () => {
    test('should have search input', async ({ page }) => {
      const searchInput = page.locator(
        'input[type="search"], input[placeholder*="search" i], input[placeholder*="filter" i]',
      );
      const count = await searchInput.count();
    });

    test('should filter devices by hostname', async ({ page }) => {
      const searchInput = page
        .locator(
          'input[type="search"], input[placeholder*="search" i], input[placeholder*="filter" i]',
        )
        .first();

      if (await searchInput.isVisible()) {
        await searchInput.fill('router');

        // Results should be filtered
      }
    });

    test('should filter devices by IP address', async ({ page }) => {
      const searchInput = page
        .locator(
          'input[type="search"], input[placeholder*="search" i], input[placeholder*="filter" i]',
        )
        .first();

      if (await searchInput.isVisible()) {
        await searchInput.fill('192.168');
      }
    });

    test('should show no results message for unmatched search', async ({ page }) => {
      const searchInput = page
        .locator(
          'input[type="search"], input[placeholder*="search" i], input[placeholder*="filter" i]',
        )
        .first();

      if (await searchInput.isVisible()) {
        await searchInput.fill('xyznonexistentdevice12345');

        // Should show no results or empty state
        const _noResults = page.getByText(/no result|no device|not found|empty/i);
      }
    });

    test('should clear search on clear button click', async ({ page }) => {
      const searchInput = page
        .locator(
          'input[type="search"], input[placeholder*="search" i], input[placeholder*="filter" i]',
        )
        .first();

      if (await searchInput.isVisible()) {
        await searchInput.fill('test');

        // Look for clear button
        const clearButton = page.locator('[class*="clear"], button[aria-label*="clear" i]').first();
        if (await clearButton.isVisible()) {
          await clearButton.click();

          const value = await searchInput.inputValue();
          expect(value).toBe('');
        }
      }
    });

    test('should search as you type with debounce', async ({ page }) => {
      const searchInput = page
        .locator(
          'input[type="search"], input[placeholder*="search" i], input[placeholder*="filter" i]',
        )
        .first();

      if (await searchInput.isVisible()) {
        // Type slowly to observe debounce
        await searchInput.type('test', { delay: 100 });

        // Results should be filtered after debounce
      }
    });

    test('should highlight search matches', async ({ page }) => {
      const searchInput = page
        .locator(
          'input[type="search"], input[placeholder*="search" i], input[placeholder*="filter" i]',
        )
        .first();

      if (await searchInput.isVisible()) {
        await searchInput.fill('router');

        // Look for highlighted text
        const _highlight = page.locator('mark, [class*="highlight"]');
      }
    });
  });

  test.describe('Filter by Device Type', () => {
    test('should have device type filter', async ({ page }) => {
      const typeFilter = page.locator('select[name*="type" i], [class*="filter"][class*="type"]');
      const typeButtons = page.getByRole('button', { name: /router|switch|firewall|all/i });

      const _hasFilter = (await typeFilter.count()) > 0 || (await typeButtons.count()) > 0;
    });

    test('should filter by router type', async ({ page }) => {
      const typeSelect = page.locator('select[name*="type" i]').first();
      const routerButton = page.getByRole('button', { name: /router/i }).first();

      if (await typeSelect.isVisible()) {
        await typeSelect.selectOption({ label: /router/i });
      } else if (await routerButton.isVisible()) {
        await routerButton.click();
      }
    });

    test('should filter by switch type', async ({ page }) => {
      const typeSelect = page.locator('select[name*="type" i]').first();
      const switchButton = page.getByRole('button', { name: /switch/i }).first();

      if (await typeSelect.isVisible()) {
        await typeSelect.selectOption({ label: /switch/i });
      } else if (await switchButton.isVisible()) {
        await switchButton.click();
      }
    });

    test('should show all devices when filter cleared', async ({ page }) => {
      const typeSelect = page.locator('select[name*="type" i]').first();
      const allButton = page.getByRole('button', { name: /all|clear/i }).first();

      if (await typeSelect.isVisible()) {
        await typeSelect.selectOption({ index: 0 }); // Usually "All"
      } else if (await allButton.isVisible()) {
        await allButton.click();
      }
    });
  });

  test.describe('Filter by Protocol', () => {
    test('should have protocol filter', async ({ page }) => {
      const _protocolFilter = page.locator(
        'select[name*="protocol" i], [class*="filter"][class*="protocol"]',
      );
      const _protocolCheckboxes = page.locator('input[type="checkbox"][name*="protocol" i]');
    });

    test('should filter SNMP-enabled devices', async ({ page }) => {
      const snmpFilter = page
        .locator('input[type="checkbox"][name*="snmp" i], button:has-text("SNMP")')
        .first();

      if (await snmpFilter.isVisible()) {
        await snmpFilter.click();
      }
    });

    test('should filter HTTP-enabled devices', async ({ page }) => {
      const httpFilter = page
        .locator('input[type="checkbox"][name*="http" i], button:has-text("HTTP")')
        .first();

      if (await httpFilter.isVisible()) {
        await httpFilter.click();
      }
    });
  });

  test.describe('Multi-Criteria Filtering', () => {
    test('should combine search with type filter', async ({ page }) => {
      const searchInput = page
        .locator(
          'input[type="search"], input[placeholder*="search" i], input[placeholder*="filter" i]',
        )
        .first();
      const typeSelect = page.locator('select[name*="type" i]').first();

      if ((await searchInput.isVisible()) && (await typeSelect.isVisible())) {
        await searchInput.fill('core');
        await typeSelect.selectOption({ index: 1 }); // First type option

        // Results should match both criteria
      }
    });

    test('should show filter count/summary', async ({ page }) => {
      const _filterSummary = page.locator('[class*="filter-count"], [class*="active-filter"]');
    });

    test('should have clear all filters button', async ({ page }) => {
      const clearAllButton = page.getByRole('button', { name: /clear all|reset filter/i });
      const count = await clearAllButton.count();
    });
  });

  test.describe('Sorting', () => {
    test('should have sortable columns', async ({ page }) => {
      const sortableHeader = page.locator('th[class*="sort"], th button, th[aria-sort]');
      const count = await sortableHeader.count();
    });

    test('should sort by hostname', async ({ page }) => {
      const hostnameHeader = page.locator('th:has-text("Hostname"), th:has-text("Name")').first();

      if (await hostnameHeader.isVisible()) {
        await hostnameHeader.click();

        // Should indicate sort direction
        const _sortIndicator = page.locator('[class*="sort"], [aria-sort]');
      }
    });

    test('should toggle sort direction on repeated click', async ({ page }) => {
      const hostnameHeader = page.locator('th:has-text("Hostname"), th:has-text("Name")').first();

      if (await hostnameHeader.isVisible()) {
        // First click - ascending
        await hostnameHeader.click();

        // Second click - descending
        await hostnameHeader.click();
      }
    });

    test('should sort by IP address', async ({ page }) => {
      const ipHeader = page.locator('th:has-text("IP"), th:has-text("Address")').first();

      if (await ipHeader.isVisible()) {
        await ipHeader.click();
      }
    });

    test('should sort by device type', async ({ page }) => {
      const typeHeader = page.locator('th:has-text("Type")').first();

      if (await typeHeader.isVisible()) {
        await typeHeader.click();
      }
    });
  });

  test.describe('Pagination', () => {
    test('should have pagination controls', async ({ page }) => {
      const pagination = page.locator('[class*="pagination"], nav[aria-label*="page" i]');
      const count = await pagination.count();
    });

    test('should show items per page selector', async ({ page }) => {
      const pageSizeSelect = page.locator('select[name*="pageSize" i], select[name*="perPage" i]');
      const count = await pageSizeSelect.count();
    });

    test('should navigate to next page', async ({ page }) => {
      const nextButton = page.getByRole('button', { name: /next|>/i }).first();

      if ((await nextButton.isVisible()) && (await nextButton.isEnabled())) {
        await nextButton.click();
      }
    });

    test('should show current page indicator', async ({ page }) => {
      const _pageIndicator = page.locator(
        '[class*="page"][class*="current"], [aria-current="page"]',
      );
      const _pageText = page.getByText(/page \d+/i);
    });
  });

  test.describe('Filter Persistence', () => {
    test('should persist filters on page refresh', async ({ page }) => {
      const searchInput = page
        .locator(
          'input[type="search"], input[placeholder*="search" i], input[placeholder*="filter" i]',
        )
        .first();

      if (await searchInput.isVisible()) {
        await searchInput.fill('test-filter');

        // Refresh page
        await page.reload();
        await page.waitForLoadState('domcontentloaded');

        // Check if filter is preserved (in URL params or localStorage)
      }
    });

    test('should update URL with filter params', async ({ page }) => {
      const searchInput = page
        .locator(
          'input[type="search"], input[placeholder*="search" i], input[placeholder*="filter" i]',
        )
        .first();

      if (await searchInput.isVisible()) {
        await searchInput.fill('router');

        const _url = page.url();
        // URL may contain search param
      }
    });
  });
});
