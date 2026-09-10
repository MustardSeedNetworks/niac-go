import { expect, type Page, test } from '@playwright/test';
import { parse } from 'yaml';
import type { ScenarioDraft } from '../src/api/library-client';
import { withIsolatedDaemon } from './isolated-daemon';

const config = `devices:
  - name: dhcp-server
    type: server
    mac: "02:00:00:00:dd:01"
    ips: [192.0.2.1]
    dhcp:
      pool_start: 192.0.2.100
      pool_end: 192.0.2.110
  - name: peer
    type: workstation
    mac: "02:00:00:00:dd:02"
    ips: [192.0.2.20]
`;

async function openDraft(page: Page, content: string): Promise<ScenarioDraft> {
  await page.getByTestId('wizard-interface-select').selectOption({ index: 1 });
  await page.getByLabel('Upload local file').setInputFiles({
    name: 'address-conflict.yaml',
    mimeType: 'application/yaml',
    buffer: Buffer.from(content),
  });
  const created = page.waitForResponse(
    (response) =>
      response.url().endsWith('/library/drafts') && response.request().method() === 'POST',
  );
  await page.getByTestId('wizard-next-button').click();
  const response = await created;
  expect(response.ok(), await response.text()).toBe(true);
  const draft: ScenarioDraft = await response.json();
  await page.getByTestId('wizard-view-behaviors').click();
  return draft;
}

test('saves an imported interface conflict without changing its address or ownership', async ({
  page,
}) => {
  const content = `${config.replace('ips: [192.0.2.1]', 'ips: [192.0.2.1]\n    interfaces: [{name: Management}]')}
behavior_timelines:
  - name: Conflict
    repeat_count: 1
    phases:
      - name: ARP
        duration_ms: 1000
        reset: true
        faults:
          - device: dhcp-server
            interface: Management
            type: duplicate_ip
            address: 192.0.2.20
`;
  await page.goto('/new-simulation');
  await openDraft(page, content);
  await expect(page.getByLabel('Conflict IPv4 address')).toHaveValue('192.0.2.20');
  await expect(page.getByLabel('Fault', { exact: true })).toHaveValue('duplicate_ip');
  const savedResponse = page.waitForResponse(
    (response) => response.url().endsWith('/behaviors') && response.request().method() === 'PUT',
  );
  await page.getByTestId('save-behaviors').click();
  const response = await savedResponse;
  expect(response.ok(), await response.text()).toBe(true);
  const saved: ScenarioDraft = await response.json();
  expect(parse(saved.content)).toEqual(parse(content));
});

test('saves an explicit zero mask without changing canonical addressing', async ({ page }) => {
  const content = `${config.replace('ips: [192.0.2.1]', 'interfaces: [{name: eth0, address: 192.0.2.1/24}]')}
behavior_timelines:
  - name: Mask
    repeat_count: 1
    phases:
      - name: Fault
        duration_ms: 1000
        reset: true
        faults:
          - device: dhcp-server
            interface: eth0
            type: bad_mask
            prefix_bits: 0
`;
  await page.goto('/new-simulation');
  await openDraft(page, content);
  const mask = page.getByLabel('IPv4 prefix length (0–32)');
  await expect(mask).toHaveValue('0');
  await mask.fill('');
  await expect(page.getByTestId('save-behaviors')).toBeDisabled();
  await mask.fill('0');
  const savedResponse = page.waitForResponse(
    (response) => response.url().endsWith('/behaviors') && response.request().method() === 'PUT',
  );
  await page.getByTestId('save-behaviors').click();
  const response = await savedResponse;
  expect(response.ok(), await response.text()).toBe(true);
  const saved: ScenarioDraft = await response.json();
  expect(parse(saved.content)).toEqual(parse(content));
});

test('saves and reopens an address fault without changing device inventory', async ({
  page,
  request,
}) => {
  await page.goto('/new-simulation');
  const original = await openDraft(page, config);
  await page.getByTestId('add-timeline').click();
  await page.getByTestId('add-device-fault').click();
  await page.getByLabel('Fault', { exact: true }).selectOption('duplicate_dhcp_offer');
  await expect(page.getByTestId('save-behaviors')).toBeDisabled();
  await page.getByLabel('Conflict IPv4 address').fill('192.0.2.20');
  const savedResponse = page.waitForResponse(
    (response) => response.url().endsWith('/behaviors') && response.request().method() === 'PUT',
  );
  await page.getByTestId('save-behaviors').click();
  const response = await savedResponse;
  expect(response.ok(), await response.text()).toBe(true);
  const saved: ScenarioDraft = await response.json();
  const before: Record<string, unknown> = parse(original.content);
  expect(parse(saved.content)).toEqual({
    ...before,
    behavior_timelines: [
      {
        name: 'Business day',
        repeat_count: 1,
        phases: [
          {
            name: 'Busy period',
            duration_ms: 30000,
            reset: true,
            faults: [
              { device: 'dhcp-server', type: 'duplicate_dhcp_offer', address: '192.0.2.20' },
            ],
          },
        ],
      },
    ],
  });
  const persistedResponse = await request.get(
    `/api/v1/library/drafts/${encodeURIComponent(saved.name)}`,
  );
  expect(persistedResponse.ok()).toBe(true);
  const persisted: ScenarioDraft = await persistedResponse.json();
  expect(persisted.content).toBe(saved.content);
  expect(persisted.revision).toBe(saved.revision);
  expect(saved.revision).not.toBe(original.revision);
  await page.reload();
  const reopened = await openDraft(page, persisted.content);
  expect(parse(reopened.content)).toEqual(parse(saved.content));
  await expect(page.getByLabel('Conflict IPv4 address')).toHaveValue('192.0.2.20');
  await expect(page.getByLabel('Fault', { exact: true })).toHaveValue('duplicate_dhcp_offer');
});

test('applies an address fault and clears only that fault through the real daemon', async ({
  page,
  request,
}) => {
  await withIsolatedDaemon(request, async (baseURL) => {
    const csrf = await request.get(`${baseURL}/api/v1/csrf-token`);
    expect(csrf.ok()).toBe(true);
    const { token } = (await csrf.json()) as { token: string };
    const start = await request.post(`${baseURL}/api/v1/simulation`, {
      headers: { 'X-Csrf-Token': token },
      data: { interface: 'e2e-dry-run0', configData: config },
    });
    expect(start.ok(), await start.text()).toBe(true);
    await page.goto(`${baseURL}/traffic`);
    const panel = page.getByTestId('device-fault-panel');
    await panel.getByLabel('Device fault target').selectOption('dhcp-server');
    await panel.getByLabel('Device fault', { exact: true }).selectOption('Latency');
    await panel.getByRole('spinbutton').fill('50');
    await panel.getByTestId('apply-device-fault').click();
    await expect(panel.getByText('50 ms', { exact: true })).toBeVisible();
    await panel.getByLabel('Device fault', { exact: true }).selectOption('Duplicate DHCP Offer');
    await expect(panel.getByRole('spinbutton')).toHaveCount(0);
    await expect(panel.getByTestId('apply-device-fault')).toBeDisabled();
    await panel.getByLabel('Conflict IPv4 address').fill('192.0.2.20');
    await panel.getByTestId('apply-device-fault').click();
    await expect(panel.getByText('192.0.2.20', { exact: true })).toBeVisible();
    const active = await request.get(`${baseURL}/api/v1/errors`);
    expect(active.ok()).toBe(true);
    expect((await active.json()).active_device_errors).toEqual({
      'dhcp-server': { Latency: { value: 50 }, 'Duplicate DHCP Offer': { address: '192.0.2.20' } },
    });
    await panel.getByRole('button', { name: 'Clear Duplicate DHCP Offer on dhcp-server' }).click();
    await expect(panel.getByText('192.0.2.20', { exact: true })).toHaveCount(0);
    const cleared = await request.get(`${baseURL}/api/v1/errors`);
    expect(cleared.ok()).toBe(true);
    expect((await cleared.json()).active_device_errors).toEqual({
      'dhcp-server': { Latency: { value: 50 } },
    });
  });
});
