import { expect, test } from '@playwright/test';

const authToken = 'niac-e2e-browser-auth-token'; // gitleaks:allow — local test fixture

test('authenticates production REST and SSE traffic without persisting the bearer', async ({
  page,
}) => {
  const protectedResponses: Array<{ url: string; status: number }> = [];
  page.on('response', (response) => {
    if (response.url().includes('/api/v1/')) {
      protectedResponses.push({ url: response.url(), status: response.status() });
    }
  });

  await page.goto('/');
  await expect(page.getByTestId('api-token-input')).toBeVisible();
  await page.getByTestId('api-token-input').fill(authToken);
  await page.getByRole('button', { name: 'Connect' }).click();

  await expect(page.getByRole('navigation')).toBeVisible();
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible();
  await expect
    .poll(() =>
      protectedResponses.some(
        (response) => response.url.endsWith('/api/v1/auth/scope') && response.status === 200,
      ),
    )
    .toBe(true);

  await page.getByRole('link', { name: /Debug/i }).click();
  await expect
    .poll(() =>
      protectedResponses.some(
        (response) => response.url.endsWith('/api/v1/stream/logs') && response.status === 200,
      ),
    )
    .toBe(true);

  expect(page.url()).not.toContain(authToken);
  const persistedValues = await page.evaluate(() => [
    ...Object.values({ ...localStorage }),
    ...Object.values({ ...sessionStorage }),
  ]);
  expect(persistedValues).not.toContain(authToken);
});
