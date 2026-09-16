import { expect, test } from '@playwright/test';
import { disableAnimations } from './helpers/auth';

/**
 * Topology DeviceNode tooltip contract.
 *
 * The card truncates the device name and shows `+N` overflow for IPs and
 * protocols, so the tooltip (hover/focus) and accessible name are the only
 * places the full values are reachable. That is the contract worth pinning.
 *
 * This spec used to call `test.skip` when the graph rendered zero nodes,
 * which it did on both browsers on every run — the tests never executed. The
 * skip did more than remove coverage: it hid a test that *contradicted*
 * production. It asserted the tooltip matched `(<type>, <status>)`, but #1354
 * deliberately removed the device status, because the value was a literal
 * `'online'` the daemon never measured. The assertion would have failed the
 * moment it ran, and the spec was left claiming a contract that no longer
 * exists.
 *
 * So the tests now bring their own topology instead of hoping one is loaded,
 * and assert the real strings by value rather than checking a section is
 * non-empty when it happens to be present.
 */

/** The page reads its session from /api/v1/simulation, not from the session
 *  list: AppContext takes `sessionId` off the simulation status and every
 *  runtime fetch is disabled until it is non-null. Without a running session
 *  the topology resolves empty no matter what the topology route returns. */
const SESSION_ID = 'e2e-topology';

function simulationStatus(deviceCount: number) {
  return {
    sessionId: SESSION_ID,
    selected: true,
    running: true,
    interface: 'lo0',
    configName: 'e2e-topology',
    deviceCount,
    uptimeSeconds: 42,
    sessions: [
      {
        sessionId: SESSION_ID,
        running: true,
        deviceCount,
        uptimeSeconds: 42,
      },
    ],
  };
}

/** Two devices chosen so the assertions can tell them apart: different
 *  type, different IP count, and one with a single protocol. */
const DEVICES = [
  {
    name: 'core-sw-01',
    type: 'switch',
    ips: ['10.44.0.2', '10.44.10.2', '10.44.20.2'],
    protocols: ['lldp', 'snmp', 'stp'],
  },
  {
    name: 'edge-rtr-01',
    type: 'router',
    ips: ['10.44.0.1'],
    protocols: ['bgp'],
  },
];

const TOPOLOGY = {
  nodes: DEVICES.map((d) => ({ name: d.name, type: d.type })),
  links: [{ source: 'core-sw-01', target: 'edge-rtr-01', label: 'Gi0/1', discovered: false }],
};

/** The tooltip DeviceNode must produce, built from the same locale strings
 *  the component uses: `{{label}} ({{type}})`, `IPs: …`, `Protocols: …`. */
function expectedTooltip(device: (typeof DEVICES)[number]): string {
  return [
    `${device.name} (${device.type})`,
    `IPs: ${device.ips.join(', ')}`,
    `Protocols: ${device.protocols.join(', ')}`,
  ].join('\n');
}

async function serveTopology(
  page: import('@playwright/test').Page,
  opts: { devices: typeof DEVICES; topology: typeof TOPOLOGY },
) {
  const json = (body: unknown) => ({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify(body),
  });

  await page.route('**/api/v1/simulation', (route) =>
    route.fulfill(json(simulationStatus(opts.devices.length))),
  );
  await page.route('**/api/v1/sessions/*/topology', (route) => route.fulfill(json(opts.topology)));
  await page.route('**/api/v1/sessions/*/devices', (route) => route.fulfill(json(opts.devices)));
  await page.route('**/api/v1/sessions/*/neighbors', (route) => route.fulfill(json([])));
  await page.route('**/api/v1/sessions/*/stats', (route) => route.fulfill(json({})));
}

test.describe('Topology — DeviceNode tooltip contract', () => {
  test.beforeEach(async ({ page }) => {
    await disableAnimations(page);
  });

  test('every node exposes its details on keyboard focus', async ({ page }) => {
    await serveTopology(page, { devices: DEVICES, topology: TOPOLOGY });
    await page.goto('/topology');
    await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });

    const nodes = page.getByTestId('topology-device-node');
    // The prerequisite is required, not hoped for: a zero-node page must fail
    // here rather than skip past the assertions below.
    await expect(nodes).toHaveCount(DEVICES.length);

    for (let i = 0; i < DEVICES.length; i++) {
      const node = nodes.nth(i);
      const aria = await node.getAttribute('aria-label');
      expect(aria, `node ${i} has no accessible name`).toBeTruthy();
      // Tab-order membership is asserted on the element, not by a Shift+Tab /
      // Tab round trip. React Flow wraps every node in a div[tabindex="0"], and
      // whether such a non-control joins sequential focus navigation is
      // engine-specific: Linux WebKit skips it, so the round trip measured the
      // browser's focus model rather than this node's reachability and failed
      // there on the first node alone, while macOS WebKit and Chromium passed.
      // A non-disabled <button> with tabIndex >= 0 is the contract the round
      // trip was standing in for — and it is the one .focus() cannot fake, since
      // a tabindex="-1" node answers .focus() and is still unreachable by Tab.
      expect(
        await node.evaluate((el) => (el as HTMLButtonElement).tabIndex),
        `node ${i} is not in the tab order`,
      ).toBeGreaterThanOrEqual(0);
      expect(
        await node.evaluate((el) => (el as HTMLButtonElement).disabled),
        `node ${i} is disabled`,
      ).toBe(false);
      await node.focus();
      await expect(node).toBeFocused();
      const descriptionId = await node.getAttribute('aria-describedby');
      expect(descriptionId).toBeTruthy();
      const tooltip = page.locator(`[id="${descriptionId}"]`);
      await expect(tooltip).toHaveRole('tooltip');
      await expect(tooltip).toBeVisible();
      await expect(tooltip).toHaveText(aria ?? '');
      await page.keyboard.press('Escape');
      await expect(tooltip).toBeHidden();
      await expect(node).toBeFocused();
    }
  });

  test('the tooltip carries the full label, type, IPs and protocols', async ({ page }) => {
    await serveTopology(page, { devices: DEVICES, topology: TOPOLOGY });
    await page.goto('/topology');
    await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });
    await expect(page.getByTestId('topology-device-node')).toHaveCount(DEVICES.length);

    // React Flow frames the graph with JavaScript after two animation frames;
    // disabling CSS animations does not stop that viewport movement.
    await page.locator('.react-flow__viewport').evaluate(
      (viewport) =>
        new Promise<void>((resolve) => {
          let previous = '';
          let stableFrames = 0;
          const observe = () => {
            const transform = getComputedStyle(viewport).transform;
            stableFrames = transform === previous ? stableFrames + 1 : 0;
            previous = transform;
            if (stableFrames >= 3) resolve();
            else requestAnimationFrame(observe);
          };
          requestAnimationFrame(observe);
        }),
    );

    // Asserted by value. The previous version only checked that an "IPs:"
    // section was non-empty *if it was present*, which passes for a node that
    // dropped every address.
    for (const device of DEVICES) {
      const node = page.getByTestId('topology-device-node').filter({ hasText: device.name });
      await node.hover();
      const descriptionId = await node.getAttribute('aria-describedby');
      expect(descriptionId).toBeTruthy();
      const tooltip = page.locator(`[id="${descriptionId}"]`);
      await expect(tooltip).toBeVisible();
      await expect(tooltip).toHaveText(expectedTooltip(device));
      await expect(node).toHaveAttribute('aria-label', expectedTooltip(device));
    }

    // Every IP must survive the card's `+N` overflow into the tooltip.
    const core = page.getByTestId('topology-device-node').filter({ hasText: 'core-sw-01' });
    const coreDescriptionId = await core.getAttribute('aria-describedby');
    expect(coreDescriptionId).toBeTruthy();
    const coreText = await page.locator(`[id="${coreDescriptionId}"]`).textContent();
    for (const ip of DEVICES[0].ips) {
      expect(coreText, `tooltip dropped ${ip}`).toContain(ip);
    }
  });

  test('does not report a device status the daemon never measures', async ({ page }) => {
    await serveTopology(page, { devices: DEVICES, topology: TOPOLOGY });
    await page.goto('/topology');
    await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });
    await expect(page.getByTestId('topology-device-node')).toHaveCount(DEVICES.length);

    // #1354 removed the fabricated `status: 'online'` literal. This spec used
    // to require it, and could not notice the contradiction because it always
    // skipped. Pin the removal so it cannot come back.
    for (let i = 0; i < DEVICES.length; i++) {
      const descriptionId = await page
        .getByTestId('topology-device-node')
        .nth(i)
        .getAttribute('aria-describedby');
      expect(descriptionId).toBeTruthy();
      const description = await page.locator(`[id="${descriptionId}"]`).textContent();
      expect(description).toBeTruthy();
      expect(description, 'tooltip must not claim a device status').not.toMatch(
        /\b(online|offline|warning)\b/i,
      );
    }
  });

  test('renders no nodes, and does not fail, for a session with an empty topology', async ({
    page,
  }) => {
    await serveTopology(page, { devices: [], topology: { nodes: [], links: [] } });
    await page.goto('/topology');
    await expect(page.getByTestId('page-header-title')).toBeVisible({ timeout: 10000 });

    // The empty state is a real state with its own assertion, rather than the
    // reason the other tests do not run.
    await expect(page.getByTestId('topology-device-node')).toHaveCount(0);
  });
});
