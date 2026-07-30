import { expect, test } from '@playwright/test';

const content = `devices:
  - name: access-1
    type: switch
    mac: 02:00:00:00:00:01
    interfaces:
      - name: Gi1/0/1
        type: ethernet
        speed: 1000000000
`;

test('authors and saves a deterministic behavior timeline', async ({ page }) => {
  let saved = false;
  await page.route('**/api/v1/simulation', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({ json: { running: false, interface: '', deviceCount: 0 } });
      return;
    }
    await route.fallback();
  });
  await page.route('**/api/v1/interfaces?filter=usable', (route) =>
    route.fulfill({
      json: {
        interfaces: [{ name: 'lo0', addresses: ['127.0.0.1'], isUp: true, isLoopback: true }],
      },
    }),
  );
  await page.route('**/api/v1/templates', (route) => route.fulfill({ json: [] }));
  await page.route('**/api/v1/library/networks', (route) => route.fulfill({ json: [] }));
  await page.route('**/api/v1/library/drafts', async (route) => {
    const request = route.request();
    expect(request.method()).toBe('POST');
    const body = request.postDataJSON() as { name: string };
    await route.fulfill({
      status: 201,
      json: {
        name: body.name,
        content,
        format: 'yaml',
        revision: 'revision-1',
        modifiedAt: '2026-07-29T12:00:00Z',
        sizeBytes: content.length,
      },
    });
  });
  await page.route('**/api/v1/library/drafts/*/behaviors', async (route) => {
    const request = route.request();
    expect(request.method()).toBe('PUT');
    expect(request.headers()['if-match']).toBe('"revision-1"');
    expect(request.postDataJSON()).toEqual({
      timelines: [
        {
          name: 'Business day',
          startOffsetMs: 0,
          repeatCount: 1,
          phases: [
            {
              name: 'Busy period',
              startOffsetMs: 0,
              durationMs: 30000,
              reset: true,
              traffic: [{ device: 'access-1', interface: 'Gi1/0/1', utilization: 75 }],
              faults: [],
            },
          ],
        },
      ],
    });
    saved = true;
    await route.fulfill({
      json: {
        name: 'browser-behavior-draft',
        content,
        format: 'yaml',
        revision: 'revision-2',
        modifiedAt: '2026-07-29T12:01:00Z',
        sizeBytes: content.length,
      },
    });
  });

  await page.goto('/new-simulation');
  await page.getByTestId('wizard-interface-select').selectOption('lo0');
  await page.getByTestId('wizard-start-empty').click();
  await page.getByTestId('wizard-next-button').click();
  await page.getByRole('tab', { name: 'Behaviors' }).click();
  await page.getByRole('button', { name: 'Add timeline' }).click();
  await page.getByTestId('save-behaviors').click();

  await expect.poll(() => saved).toBe(true);
});
