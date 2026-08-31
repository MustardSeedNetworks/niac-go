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
  let draftMethod: string | undefined;
  let behaviorRequest: { method: string; ifMatch?: string; body: unknown } | undefined;
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
    draftMethod = request.method();
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
    // Capture only. Asserting in here used to hang the test: a failed expect()
    // throws before route.fulfill(), so the request the app is awaiting never
    // gets a response, and the test dies on the 30s timeout with no diff to
    // read instead of failing fast on the mismatch. Assertions live after the
    // await below, where a failure prints what actually differed.
    behaviorRequest = {
      method: request.method(),
      ifMatch: request.headers()['if-match'],
      body: request.postDataJSON() as unknown,
    };
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
  // The Behaviors tab is disabled until the draft POST resolves -- the step
  // derives visualSupported from draft.content, which does not exist until
  // then. Clicking straight after Next therefore raced the response: Playwright
  // waited for the button to become enabled and on WebKit that wait ran past
  // the 30s timeout, while Chromium usually won. Arm the wait before the click
  // so a fast response cannot be missed, then require the tab to be enabled
  // before clicking it.
  const draftSaved = page.waitForResponse(
    (response) =>
      response.url().includes('/api/v1/library/drafts') && response.request().method() === 'POST',
  );
  await page.getByTestId('wizard-next-button').click();
  await draftSaved;

  const behaviorsTab = page.getByTestId('wizard-view-behaviors');
  await expect(behaviorsTab).toBeEnabled();
  await behaviorsTab.click();
  await expect(behaviorsTab).toHaveAttribute('aria-selected', 'true');
  await page.getByRole('button', { name: 'Add timeline' }).click();
  // Wait on the response itself rather than polling a closure flag: the wait is
  // armed before the click, so it cannot miss a fast response, and a failure
  // names the request that never arrived.
  const behaviorsSaved = page.waitForResponse(
    (response) => response.url().includes('/behaviors') && response.request().method() === 'PUT',
  );
  await page.getByTestId('save-behaviors').click();
  await behaviorsSaved;

  expect(draftMethod).toBe('POST');
  expect(behaviorRequest?.method).toBe('PUT');
  expect(behaviorRequest?.ifMatch).toBe('"revision-1"');
  expect(behaviorRequest?.body).toEqual({
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
});
