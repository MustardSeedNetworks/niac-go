import { expect, type Page, test } from '@playwright/test';
import { buildScalePack, packResponses, SCALE_PACK_DEVICE_COUNT } from './fixtures/scale-pack';
import { disableAnimations } from './helpers/auth';

/**
 * Scenario-pack scale (AUD-10).
 *
 * `enterprise-scale` runs 543 devices. The device list used to mount a row per
 * device and the topology drew every node, so both pages blocked the main
 * thread for seconds. These specs load a 543-device pack through route
 * fixtures and pin two things: only a window of the list or graph is in the
 * DOM, and no stretch of main-thread work exceeds the budget.
 *
 * The block measurement is a timer heartbeat rather than the Long Tasks API:
 * WebKit does not implement `longtask`, and the fleet runs both engines. A
 * 20 ms interval that fires `gap` ms late was blocked for about `gap - 20` ms,
 * which is the same quantity a long task reports.
 */

const SESSION_ID = 'e2e-pack-scale';
const HEARTBEAT_MS = 20;

/** Longest main-thread block tolerated while scrolling and filtering the list. */
const LIST_BLOCK_BUDGET_MS = 200;
/** Longest block tolerated while the topology lays out and paints 543 devices. */
const TOPOLOGY_BLOCK_BUDGET_MS = 2500;
/** DOM ceiling: a window plus overscan, far below the 543-device pack. */
const MAX_MOUNTED_ROWS = 60;

declare global {
  interface Window {
    __maxBlockMs: number;
  }
}

async function installHeartbeat(page: Page) {
  await page.addInitScript((interval) => {
    window.__maxBlockMs = 0;
    let last = performance.now();
    setInterval(() => {
      const now = performance.now();
      window.__maxBlockMs = Math.max(window.__maxBlockMs, now - last - interval);
      last = now;
    }, interval);
  }, HEARTBEAT_MS);
}

const resetBlock = (page: Page) =>
  page.evaluate(() => {
    window.__maxBlockMs = 0;
  });
const readBlock = (page: Page) => page.evaluate(() => Math.round(window.__maxBlockMs));

async function servePack(page: Page) {
  const { runtimeDevices, topology, configDevices } = packResponses();
  const json = (body: unknown) => ({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify(body),
  });
  await page.route('**/api/v1/simulation', (route) =>
    route.fulfill(
      json({
        sessionId: SESSION_ID,
        selected: true,
        running: true,
        interface: 'lo0',
        configName: 'enterprise-scale',
        deviceCount: SCALE_PACK_DEVICE_COUNT,
        uptimeSeconds: 42,
        sessions: [
          {
            sessionId: SESSION_ID,
            running: true,
            deviceCount: SCALE_PACK_DEVICE_COUNT,
            uptimeSeconds: 42,
          },
        ],
      }),
    ),
  );
  await page.route('**/api/v1/config/devices', (route) => route.fulfill(json(configDevices)));
  await page.route('**/api/v1/sessions/*/topology', (route) => route.fulfill(json(topology)));
  await page.route('**/api/v1/sessions/*/devices', (route) => route.fulfill(json(runtimeDevices)));
  await page.route('**/api/v1/sessions/*/neighbors', (route) => route.fulfill(json([])));
  await page.route('**/api/v1/sessions/*/stats', (route) => route.fulfill(json({})));
}

/** Surfaces the measurements in the JSON report; the PR's timings come from it. */
function report(label: string, metrics: Record<string, number>) {
  test
    .info()
    .annotations.push({ type: `pack-scale ${label}`, description: JSON.stringify(metrics) });
}

const VIEWPORTS = [
  { name: 'desktop', width: 1280, height: 800 },
  { name: 'phone', width: 390, height: 844 },
];

for (const viewport of VIEWPORTS) {
  test.describe(`pack scale at ${viewport.name} (${viewport.width}px)`, () => {
    test.use({ viewport: { width: viewport.width, height: viewport.height } });

    test.beforeEach(async ({ page }) => {
      // Heartbeat first: in WebKit disableAnimations' init script throws before
      // documentElement exists, which silently drops every init script after it.
      await installHeartbeat(page);
      await disableAnimations(page);
      await servePack(page);
    });

    for (const mode of ['table', 'cards'] as const) {
      test(`device list (${mode}) mounts a window and scrolls the whole pack`, async ({ page }) => {
        const { devices } = buildScalePack();
        const started = Date.now();
        await page.goto('/device-config');
        await page.getByTestId(`device-view-${mode}`).click();
        const rows = page.getByTestId(mode === 'table' ? 'device-list-row' : 'device-list-card');
        await expect(rows.first()).toBeVisible({ timeout: 20_000 });
        const firstRowMs = Date.now() - started;

        expect(await rows.count()).toBeLessThanOrEqual(MAX_MOUNTED_ROWS);

        const scroller = page.getByTestId('device-list-rows');
        await resetBlock(page);
        await scroller.evaluate((el) => {
          el.scrollTop = el.scrollHeight;
        });
        const last = devices[devices.length - 1]?.name as string;
        await expect(page.getByText(last, { exact: true })).toBeVisible();
        const scrollBlockMs = await readBlock(page);
        expect(await rows.count()).toBeLessThanOrEqual(MAX_MOUNTED_ROWS);

        // Search narrows the whole pack, not just the mounted window.
        await resetBlock(page);
        await page.getByTestId('device-search-input').fill('EHV-CORE-02');
        await expect(page.getByText('EHV-CORE-02', { exact: true })).toBeVisible();
        await expect(rows).toHaveCount(1);
        const searchBlockMs = await readBlock(page);

        report(`list-${mode}-${viewport.name}`, { firstRowMs, scrollBlockMs, searchBlockMs });
        expect(scrollBlockMs).toBeLessThanOrEqual(LIST_BLOCK_BUDGET_MS);
        expect(searchBlockMs).toBeLessThanOrEqual(LIST_BLOCK_BUDGET_MS);
      });
    }

    test('topology renders a window of the pack without blocking', async ({ page }) => {
      const started = Date.now();
      await page.goto('/topology');
      const nodes = page.getByTestId('topology-device-node');
      await expect(nodes.first()).toBeVisible({ timeout: 30_000 });
      const firstNodeMs = Date.now() - started;
      const mounted = await nodes.count();
      // Let layout, measurement and the first fit settle before reading the block.
      await page.waitForTimeout(1000);
      const blockMs = await readBlock(page);

      // Selecting a node dims everything outside its neighbourhood. Recorded, not
      // budgeted: React Flow repainting a fitted pack varies too much by runner
      // to gate on. That the click does not re-run the layout is pinned by
      // useTopologyLayout.test.tsx.
      await resetBlock(page);
      // A script click: at 390 px the legend panel sits over the first node, which
      // is a layout matter this spec does not own.
      await nodes.first().evaluate((node) => node.click());
      await page.waitForTimeout(500);
      const focusBlockMs = await readBlock(page);

      report(`topology-${viewport.name}`, { firstNodeMs, mounted, blockMs, focusBlockMs });
      expect(mounted).toBeLessThan(SCALE_PACK_DEVICE_COUNT);
      expect(blockMs).toBeLessThanOrEqual(TOPOLOGY_BLOCK_BUDGET_MS);

      // One site is one readable graph; the pack's other sites leave the canvas.
      await page.getByTestId('topology-site-filter').selectOption('EVT');
      await expect(nodes.first()).toBeVisible();
      await expect(page.getByTestId('topology-site-filter')).toHaveValue('EVT');
      expect(await nodes.count()).toBeLessThanOrEqual(SCALE_PACK_DEVICE_COUNT / 3);
    });
  });
}
