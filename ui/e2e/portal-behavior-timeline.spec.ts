import { expect, type Page, test } from '@playwright/test';
import { parse } from 'yaml';
import type { ScenarioDraft } from '../src/api/library-client';

const config = `devices:
  - name: guest-gateway
    type: router
    mac: "02:00:00:00:cc:02"
    ips: ["192.0.2.20"]
    http:
      enabled: true
      server_name: GuestGateway/1.0
`;

async function importDraft(page: Page, content: string): Promise<ScenarioDraft> {
  await page.goto('/new-simulation');
  await page.getByTestId('wizard-interface-select').selectOption({ index: 1 });
  await page.getByLabel('Upload local file').setInputFiles({
    name: 'guest-portal.yaml',
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

test('authors a portal and reimports its persisted configuration', async ({ page, request }) => {
  const original = await importDraft(page, config);
  await page.getByTestId('add-timeline').click();
  await page.getByTestId('add-device-fault').click();
  await page.getByLabel('Latency (milliseconds)', { exact: true }).fill('500');
  await page.getByLabel('Fault', { exact: true }).selectOption('captive_portal');
  await expect(page.getByLabel('Latency (milliseconds)', { exact: true })).toHaveCount(0);
  await expect(
    page.getByTestId('behavior-composer').getByLabel('Interface', { exact: true }),
  ).toHaveCount(0);
  const savedResponse = page.waitForResponse(
    (response) => response.url().endsWith('/behaviors') && response.request().method() === 'PUT',
  );
  await page.getByTestId('save-behaviors').click();
  const response = await savedResponse;
  expect(response.ok(), await response.text()).toBe(true);
  const saved: ScenarioDraft = await response.json();
  const before: Record<string, unknown> = parse(original.content);
  const after: Record<string, unknown> = parse(saved.content);
  expect(after).toMatchObject({
    ...before,
    behavior_timelines: [
      { phases: [{ faults: [{ device: 'guest-gateway', type: 'captive_portal', value: 1 }] }] },
    ],
  });
  const { behavior_timelines: timelines, ...inventory } = after;
  expect(inventory).toEqual(before);
  expect(timelines).toHaveLength(1);
  expect(saved.revision).not.toBe(original.revision);
  const stored = await request.get(`/api/v1/library/drafts/${encodeURIComponent(saved.name)}`);
  expect(stored.ok(), await stored.text()).toBe(true);
  const persisted: ScenarioDraft = await stored.json();
  expect(persisted.content).toBe(saved.content);
  const reopened = await importDraft(page, persisted.content);
  expect(parse(reopened.content)).toEqual(after);
  await expect(page.getByLabel('Fault', { exact: true })).toHaveValue('captive_portal');
  await expect(page.getByLabel('Latency (milliseconds)', { exact: true })).toHaveCount(0);
});
