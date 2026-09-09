import { expect, type Request, test } from '@playwright/test';

for (const route of ['/walk-analyzer', '/walk-validator']) {
  test(`${route} loads the library once across file selections`, async ({ page }) => {
    const reads: string[] = [];
    const onRequest = (request: Request) => {
      if (new URL(request.url()).pathname === '/api/v1/library/walks') reads.push(request.url());
    };
    page.on('request', onRequest);
    try {
      await page.goto(route);
      const picker = page.getByTestId(`${route.slice(1)}-picker`);
      await expect(picker).toBeEnabled();
      const options = picker.getByRole('option');
      expect(await options.count()).toBeGreaterThan(1);
      const first = await options.nth(0).getAttribute('value');
      const second = await options.nth(1).getAttribute('value');
      expect(first).toBeTruthy();
      expect(second).toBeTruthy();
      if (!first || !second) throw new Error('Starter library must contain two named walks');
      await picker.selectOption(second);
      await expect(picker).toHaveValue(second);
      await picker.selectOption(first);
      await expect(picker).toHaveValue(first);
      expect(reads).toHaveLength(1);
      await expect(page.getByRole('heading', { level: 1 })).toHaveCount(1);
    } finally {
      page.off('request', onRequest);
    }
  });
}
