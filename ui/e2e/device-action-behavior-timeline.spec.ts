import { expect, type Page, test } from '@playwright/test';
import { parse } from 'yaml';
import type { ScenarioDraft } from '../src/api/library-client';

const config = `devices:
  - name: campus-switch
    type: switch
    mac: "02:00:00:00:aa:01"
    ips: ["192.0.2.30"]
    snmp_agent:
      enabled: true
      community: action_demo
      sysname: campus-switch
    stp:
      enabled: true
      bridge_priority: 32768
      forward_delay: 15
      hello_time: 2
      max_age: 20
      version: stp
`;

async function importActionsDraft(page: Page, content: string): Promise<ScenarioDraft> {
  await page.goto('/new-simulation');
  await page.getByTestId('wizard-interface-select').selectOption({ index: 1 });
  await page.getByLabel('Upload local file').setInputFiles({
    name: 'device-actions.yaml',
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

test('preserves one-shot operations through authoring, save and reimport', async ({
  page,
  request,
}) => {
  const original = await importActionsDraft(page, config);
  await page.getByTestId('add-timeline').click();
  await page.getByTestId('add-device-action').click();
  await page.getByTestId('add-device-action').click();
  const operations = page.getByLabel('Operation', { exact: true });
  await expect(operations).toHaveCount(2);
  await operations.nth(0).selectOption('reboot');
  await operations.nth(1).selectOption('reboot');
  await expect(page.getByTestId('save-behaviors')).toBeDisabled();
  await operations.nth(1).selectOption('stp_topology_change');
  await expect(page.getByTestId('save-behaviors')).toBeEnabled();
  await expect(page.getByLabel('Utilization (%)', { exact: true })).toHaveCount(0);
  await expect(
    page.getByTestId('behavior-composer').getByLabel('Interface', { exact: true }),
  ).toHaveCount(0);
  const saving = page.waitForResponse(
    (response) => response.url().endsWith('/behaviors') && response.request().method() === 'PUT',
  );
  await page.getByTestId('save-behaviors').click();
  const response = await saving;
  expect(response.ok(), await response.text()).toBe(true);
  const saved: ScenarioDraft = await response.json();
  const before: Record<string, unknown> = parse(original.content);
  const after: Record<string, unknown> = parse(saved.content);
  const { behavior_timelines: timelines, ...inventory } = after;
  expect(inventory).toEqual(before);
  expect(timelines).toEqual([
    {
      name: 'Business day',
      repeat_count: 1,
      phases: [
        {
          name: 'Busy period',
          duration_ms: 30000,
          reset: true,
          actions: [
            { device: 'campus-switch', type: 'reboot' },
            { device: 'campus-switch', type: 'stp_topology_change' },
          ],
        },
      ],
    },
  ]);
  const stored = await request.get(`/api/v1/library/drafts/${encodeURIComponent(saved.name)}`);
  expect(stored.ok(), await stored.text()).toBe(true);
  const persisted: ScenarioDraft = await stored.json();
  expect(persisted.content).toBe(saved.content);
  expect(persisted.revision).toBe(saved.revision);
  const reopened = await importActionsDraft(page, persisted.content);
  expect(parse(reopened.content)).toEqual(after);
  await expect(page.getByLabel('Operation', { exact: true }).nth(0)).toHaveValue('reboot');
  await expect(page.getByLabel('Operation', { exact: true }).nth(1)).toHaveValue(
    'stp_topology_change',
  );
});
