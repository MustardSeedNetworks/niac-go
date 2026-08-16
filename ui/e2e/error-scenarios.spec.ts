import { expect, test } from '@playwright/test';

/**
 * Error Scenario Tests
 *
 * Every test here used to assert only `expect(page.locator('#root')).toBeAttached()`.
 * `#root` is the mount point in index.html — it is attached before React runs and
 * stays attached whether the app renders, renders an error, or renders nothing at
 * all. So all seven passed regardless of how the failure was handled, and none
 * could fail if error handling regressed.
 *
 * The scenarios themselves were worth keeping; only the assertions were empty.
 * Each now asserts the specific outcome that scenario actually produces, and in
 * no case is the ErrorBoundary fallback shown — a bad daemon degrades, it does
 * not white-screen the operator.
 *
 * Note the split, which the old assertion made invisible: an unreachable daemon
 * and 4xx/5xx responses land on the AuthGate connect screen with an error line,
 * because the token probe fails; a slow or malformed-JSON response still boots
 * the full shell. Both are correct, and they are different — asserting "either"
 * is what let these tests pass for years without checking anything.
 */

/** The app shell rendered, and no render-crash fallback replaced it. */
async function expectShellSurvives(page: import('@playwright/test').Page): Promise<void> {
  await expect(page.getByTestId('sidebar-settings-button')).toBeVisible({ timeout: 15000 });
  await expect(page.getByTestId('error-boundary-fallback')).toHaveCount(0);
}

/**
 * The AuthGate connect screen with its error line — what the app correctly
 * shows when it cannot reach or authenticate against the daemon. Asserted
 * per-scenario rather than as a shell-or-gate alternative, because an
 * either/or assertion is how these tests became unfalsifiable the first time.
 */
async function expectConnectGateWithError(page: import('@playwright/test').Page): Promise<void> {
  await expect(page.getByTestId('api-token-input')).toBeVisible({ timeout: 15000 });
  await expect(page.getByTestId('auth-gate-error')).toBeVisible();
  await expect(page.getByTestId('error-boundary-fallback')).toHaveCount(0);
}

test.describe('Error Scenarios', () => {
  test('shows the connect gate with an error when the daemon refuses every connection', async ({
    page,
  }) => {
    await page.route('**/api/v1/**', (route) => route.abort('failed'));

    await page.goto('/');

    await expectConnectGateWithError(page);
  });

  test('survives a daemon that responds slowly', async ({ page }) => {
    await page.route('**/api/v1/**', async (route) => {
      await new Promise((resolve) => setTimeout(resolve, 2000));
      await route.continue();
    });

    await page.goto('/');

    await expectShellSurvives(page);
  });

  test('redirects an unknown route back to the dashboard', async ({ page }) => {
    await page.goto('/nonexistent-page-12345');

    // App.tsx: <Route path="*" element={<Navigate to="/" replace />} />.
    // The old test only checked #root, so a broken catch-all would have passed.
    await expect(page).toHaveURL(/\/$/);
    await expectShellSurvives(page);
  });

  test('shows the connect gate with an error on HTTP 500', async ({ page }) => {
    await page.route('**/api/v1/**', (route) =>
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Internal Server Error' }),
      }),
    );

    await page.goto('/');

    await expectConnectGateWithError(page);
  });

  test('survives malformed JSON from every endpoint', async ({ page }) => {
    await page.route('**/api/v1/**', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: 'invalid json{{{',
      }),
    );

    await page.goto('/');

    // A JSON parse error thrown during render is exactly what would trip the
    // ErrorBoundary, so this is the scenario the fallback assertion is for.
    await expectShellSurvives(page);
  });

  test('renders the devices empty state when the daemon returns no devices', async ({ page }) => {
    await page.route('**/api/v1/devices', (route) =>
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([]),
      }),
    );

    await page.goto('/devices');

    await expectShellSurvives(page);
    // BaseCard swaps the table for its empty branch when the list is empty.
    await expect(page.getByTestId('devices-card-empty')).toBeVisible({ timeout: 15000 });
  });

  test('shows the connect gate with an error on HTTP 400', async ({ page }) => {
    await page.route('**/api/v1/**', (route) =>
      route.fulfill({
        status: 400,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Bad Request' }),
      }),
    );

    await page.goto('/');

    await expectConnectGateWithError(page);
  });
});
