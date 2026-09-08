import { expect, test } from '@playwright/test';

// GET /api/v1/auth/scope decides whether the shell renders the page read-only,
// and it answers after the first paint. Issue #1941: the shell used to swap its
// tree shape when the answer arrived, remounting the routed page and discarding
// whatever the operator had already started. It reads as a flaky E2E run --
// under load the answer lands after the test's first interaction rather than
// before it -- so this spec makes the losing order the only order.
const scopeDelayMs = 1500;

test('work started before the token scope arrives survives it', async ({ page }) => {
  await page.route('**/api/v1/auth/scope', async (route) => {
    await new Promise((resolve) => setTimeout(resolve, scopeDelayMs));
    await route.fulfill({ json: { scope: 'admin' } });
  });
  await page.route('**/api/v1/library/walks', (route) => route.fulfill({ json: [] }));

  await page.goto('/walk-analyzer');
  const walkFile = page.getByTestId('walk-profile-file');
  await expect(walkFile).toBeVisible();
  await walkFile.setInputFiles('e2e/fixtures/office.snmpwalk');

  // The chosen file is what enables Import, so an enabled Import button is the
  // page's own report that it still holds the operator's work.
  await expect(page.getByTestId('walk-profile-import')).toBeEnabled({
    timeout: scopeDelayMs + 5000,
  });
});
