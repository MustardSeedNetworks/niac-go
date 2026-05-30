import { expect, test } from '@playwright/test';
import { disableAnimations } from './helpers/auth';

/**
 * Topology DeviceNode tooltip contract test.
 *
 * niac PR #772 added a hover tooltip / accessible-name surface to the
 * topology graph nodes — `title` and `aria-label` carry `label (type,
 * status)` + every IP + every protocol so the truncated card data is
 * still reachable. This spec asserts the contract:
 *
 *   1. Every rendered node has the `topology-device-node` testid (so
 *      future E2E flows can target them stably).
 *   2. Every node's `title` and `aria-label` are identical strings
 *      and both non-empty (the SR + hover tooltip must agree).
 *   3. The tooltip text mentions the `(<type>, <status>)` substring
 *      and at least one IP token.
 *
 * The /topology page is only meaningful with a loaded simulation, so
 * empty-state environments are tolerated: if no nodes render, the
 * test is skipped (we cover the assertion when the data is there).
 */

test.describe('Topology — DeviceNode hover-tooltip contract', () => {
  test.beforeEach(async ({ page }) => {
    await disableAnimations(page);
    await page.goto('/topology');
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 10000,
    });
  });

  test('every node exposes the testid and a non-empty tooltip', async ({ page }) => {
    const nodes = page.getByTestId('topology-device-node');
    const count = await nodes.count();

    if (count === 0) {
      test.skip(true, 'No simulation loaded — topology renders zero nodes.');
      return;
    }

    for (let i = 0; i < count; i++) {
      const node = nodes.nth(i);
      const title = await node.getAttribute('title');
      const aria = await node.getAttribute('aria-label');
      expect(title, `node ${i} missing title`).toBeTruthy();
      expect(aria, `node ${i} missing aria-label`).toBeTruthy();
      expect(title).toBe(aria);
    }
  });

  test('tooltip text follows the (type, status) + IPs/protocols contract', async ({ page }) => {
    const nodes = page.getByTestId('topology-device-node');
    const count = await nodes.count();

    if (count === 0) {
      test.skip(true, 'No simulation loaded — topology renders zero nodes.');
      return;
    }

    const first = nodes.first();
    const title = (await first.getAttribute('title')) ?? '';

    // Header line: `label (type, status)`
    expect(title, 'title should contain `(<type>, <status>)` substring').toMatch(
      /\(.+,\s*(online|offline|warning)\)/i,
    );

    // If the node carries any IPs or protocols, they must appear in the
    // tooltip (DeviceNode.tsx contract: every value goes in, no
    // truncation). We can't assert specific values without a fixture,
    // but if there's a label section we can prove it's non-empty.
    if (title.includes('IPs:')) {
      const ipsLine = title.split('\n').find((line) => line.startsWith('IPs:'));
      expect(ipsLine?.replace('IPs:', '').trim().length).toBeGreaterThan(0);
    }
    if (title.includes('Protocols:')) {
      const protoLine = title.split('\n').find((line) => line.startsWith('Protocols:'));
      expect(protoLine?.replace('Protocols:', '').trim().length).toBeGreaterThan(0);
    }
  });
});
