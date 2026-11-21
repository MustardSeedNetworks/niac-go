import { expect, test } from '@playwright/test';
import { withIsolatedDaemon } from './isolated-daemon';

/**
 * Shipped scenarios are Examples: served from the binary, read-only, listed in
 * the library beside the operator's own networks, and duplicated into an
 * editable copy. Each test runs against its own daemon and library so a
 * duplicate here cannot change what the shared harness library lists.
 */

interface ListedNetwork {
  name: string;
  source: string;
}

// Valid on its own, so a refusal can only be about the name.
const VALID_NETWORK = `devices:
  - name: lab-router
    type: router
    mac: "02:00:00:00:ee:01"
    ips: [192.0.2.1]
    snmp_agent: {community: public}
`;

test('the library lists every shipped scenario as an example and duplicates one into a copy', async ({
  page,
  request,
}) => {
  await withIsolatedDaemon(request, async (baseURL) => {
    const listed = await request.get(`${baseURL}/api/v1/library/networks`);
    expect(listed.ok()).toBe(true);
    const networks = (await listed.json()) as ListedNetwork[];
    for (const name of ['basic-network', 'captured-switch', 'hospital', 'campus']) {
      expect(networks.find((network) => network.name === name)?.source).toBe('example');
    }

    await page.goto(`${baseURL}/new-simulation`);
    await page.getByTestId('wizard-source-tab-library').click();
    await expect(page.getByTestId('config-example-badge-basic-network')).toBeVisible();
    await expect(page.getByTestId('config-example-badge-hospital')).toBeVisible();

    await page.getByTestId('config-duplicate-basic-network').click();
    await expect(page.getByTestId('config-duplicate-saved')).toContainText('basic-network-copy');
    const copy = page.getByTestId('config-item-saved:basic-network-copy');
    await expect(copy).toBeVisible();
    await expect(page.getByTestId('config-example-badge-basic-network-copy')).toHaveCount(0);
    await expect(page.getByTestId('config-duplicate-basic-network-copy')).toHaveCount(0);
    // The example it came from is still listed.
    await expect(page.getByTestId('config-example-badge-basic-network')).toBeVisible();

    // A second copy under the same default name is refused, and says so.
    await page.getByTestId('config-duplicate-basic-network').click();
    await expect(page.getByTestId('config-duplicate-failed')).toBeVisible();
  });
});

test('a shipped example starts by name, and a library file of the same name takes over', async ({
  request,
}) => {
  await withIsolatedDaemon(request, async (baseURL) => {
    const csrf = await request.get(`${baseURL}/api/v1/csrf-token`);
    expect(csrf.ok()).toBe(true);
    const { token } = (await csrf.json()) as { token: string };
    const headers = { 'X-Csrf-Token': token };

    const started = await request.post(`${baseURL}/api/v1/simulation`, {
      headers,
      data: { interface: 'e2e-dry-run0', scenarioName: 'basic-network' },
    });
    expect(started.ok(), await started.text()).toBe(true);
    const status = await request.get(`${baseURL}/api/v1/simulation`);
    expect(((await status.json()) as { running: boolean }).running).toBe(true);

    // Examples are read-only: no delete, and no save over the name.
    const removed = await request.delete(`${baseURL}/api/v1/library/networks/basic-network`, {
      headers,
    });
    expect(removed.status()).toBe(409);
    const overwritten = await request.post(`${baseURL}/api/v1/library/networks`, {
      headers,
      data: { name: 'basic-network', content: VALID_NETWORK },
    });
    expect(overwritten.status()).toBe(409);
  });
});
