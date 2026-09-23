import { expect, type Page } from '@playwright/test';

export async function expectPageFirst(page: Page): Promise<void> {
  const frame = page.getByTestId('page-frame');
  await expect(page.getByTestId('page-header-title')).toBeVisible();
  await expect(frame).toBeVisible();
  const offset = await frame.evaluate((node) => {
    const main = node.closest('main');
    const content = node.parentElement;
    if (!main || !content) throw new Error('Missing page layout');
    const inset =
      Number.parseFloat(getComputedStyle(main).paddingTop) +
      Number.parseFloat(getComputedStyle(content).paddingTop);
    return node.getBoundingClientRect().top - main.getBoundingClientRect().top - inset;
  });
  expect(offset, 'Unexpected content above the page frame').toBeCloseTo(0, 1);
}
