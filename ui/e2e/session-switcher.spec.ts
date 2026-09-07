import { expect, test } from '@playwright/test';

/**
 * Scenario switcher (U3) E2E.
 *
 * A NIAC daemon runs several scenarios at once and every runtime read is
 * scoped to one of them. The switcher lives in the header so the operator can
 * see which scenario they are reading, and change it, from any page.
 *
 * The assertion is the request URL rather than what is on screen: two
 * scenarios can show the same device names, so "the page changed" is not
 * evidence that the reads moved. Switching must repoint
 * /api/v1/sessions/<id>/... for Devices, Topology and Packets, and it must do
 * so without navigating.
 */

const session = (sessionId: string, deviceCount: number) => ({
  running: true,
  sessionId,
  interface: `veth-${sessionId}`,
  deviceCount,
  uptimeSeconds: 42,
});

const scoped = /\/api\/v1\/sessions\/([a-z0-9-]+)\//;

/**
 * Assert that reads from here on name only `sessionId`. Clearing the log and
 * asserting immediately would race a poll that was already in flight across
 * the switch; waiting for one more read first means the window under test is
 * steady state.
 */
async function expectStableOn(
  page: import('@playwright/test').Page,
  reads: string[],
  sessionId: string,
): Promise<void> {
  reads.length = 0;
  await expect.poll(() => reads.length).toBeGreaterThan(0);
  expect(reads.filter((path) => scoped.exec(path)?.[1] !== sessionId)).toEqual([]);
}

test('switches every runtime read to the scenario picked in the header', async ({ page }) => {
  const reads: string[] = [];

  await page.route('**/api/v1/simulation', async (route) => {
    if (route.request().method() !== 'GET') {
      await route.fallback();
      return;
    }
    await route.fulfill({
      json: {
        ...session('hospital', 2),
        sessions: [session('hospital', 2), session('warehouse', 1)],
      },
    });
  });

  const streamSessions: string[] = [];
  await page.route('**/api/v1/stream/packets*', async (route) => {
    streamSessions.push(new URL(route.request().url()).searchParams.get('sessionId') ?? '');
    // An SSE response that never sends an event: the test only needs the URL
    // the page subscribed to, and a body would keep the request open.
    await route.fulfill({ status: 200, contentType: 'text/event-stream', body: '' });
  });
  // The pages this test walks through poll the content library, which nothing
  // here asserts on. Left unstubbed, three browser projects at once drive the
  // daemon's rate limiter into 429s and the app drops to its "NIAC could not
  // be reached" screen — a failure of the test's own load, not of the switcher.
  await page.route('**/api/v1/library/**', (route) => route.fulfill({ json: [] }));
  await page.route('**/api/v1/capture/status', (route) =>
    route.fulfill({ json: { running: false } }),
  );

  await page.route('**/api/v1/sessions/*/**', async (route) => {
    const url = new URL(route.request().url());
    reads.push(url.pathname);
    const resource = url.pathname.split('/').pop();
    await route.fulfill({
      json: resource === 'topology' ? { nodes: [], edges: [] } : [],
    });
  });

  await page.goto('/devices');
  await page.waitForLoadState('domcontentloaded');
  // Both sidebars are in the DOM; drive the desktop one at this viewport.
  const nav = page.getByTestId('sidebar-desktop');

  const switcher = page.getByTestId('session-switcher-select');
  await expect(switcher).toBeVisible({ timeout: 10000 });
  await expect(switcher).toHaveValue('hospital');
  await expect
    .poll(() => reads.some((path) => path.startsWith('/api/v1/sessions/hospital/')))
    .toBe(true);

  const urlBeforeSwitch = page.url();
  await switcher.selectOption('warehouse');

  // Devices is the page we are on; it must repoint where it stands. Wait for
  // the first warehouse read before clearing: a hospital poll already in
  // flight when the switch happens is not a failure, so the window that has
  // to be free of hospital is the one after the switch has taken effect.
  await expect
    .poll(() => reads.some((path) => path === '/api/v1/sessions/warehouse/devices'))
    .toBe(true);
  expect(page.url()).toBe(urlBeforeSwitch);
  await expectStableOn(page, reads, 'warehouse');

  // Topology reads the same selection, reached by in-app navigation: the
  // selection is this browser's and lives in the running app, so the switch
  // has to survive moving between pages without being made again.
  reads.length = 0;
  await nav.getByTestId('nav-item-topology').click();
  await expect(page).toHaveURL(/\/topology$/);
  await expect(page.getByTestId('session-switcher-select')).toHaveValue('warehouse');
  await expect
    .poll(() => reads.some((path) => path === '/api/v1/sessions/warehouse/topology'))
    .toBe(true);
  await expectStableOn(page, reads, 'warehouse');

  // Packets is scoped through the stream URL rather than a /sessions/ read:
  // before U3 it subscribed to whichever session the daemon reported, so the
  // operator watched one scenario's packets under another scenario's name.
  streamSessions.length = 0;
  await nav.getByTestId('nav-item-packets').click();
  await expect(page).toHaveURL(/\/packets/);
  await expect.poll(() => streamSessions).toContain('warehouse');
  expect(streamSessions).not.toContain('hospital');
});
