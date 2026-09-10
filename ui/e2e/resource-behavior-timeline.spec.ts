import { type APIRequestContext, expect, type Page, test } from '@playwright/test';
import { parse } from 'yaml';
import type { ScenarioDraft } from '../src/api/library-client';

const types = ['cpu_percent', 'memory_percent', 'disk_percent'];
const values = ['1', '100', '73'];

async function prepareDraft(page: Page): Promise<ScenarioDraft> {
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

async function selectInterface(page: Page) {
  await page.getByTestId('wizard-interface-select').selectOption({ index: 1 });
}

async function editResourceValues(page: Page) {
  const faults = page.getByLabel('Fault', { exact: true });
  const percentages = page.getByLabel('Utilization (%)', { exact: true });
  await expect(faults).toHaveCount(3);
  await expect(percentages).toHaveCount(3);
  await expect(
    page.getByTestId('behavior-composer').getByLabel('Interface', { exact: true }),
  ).toHaveCount(0);
  for (const [index, type] of types.entries()) {
    await expect(faults.nth(index)).toHaveValue(type);
    await percentages.nth(index).fill('101');
    await expect(page.getByTestId('save-behaviors')).toBeDisabled();
    await percentages.nth(index).fill('0');
    await expect(page.getByTestId('save-behaviors')).toBeDisabled();
    await percentages.nth(index).fill(values[index]);
  }
}

async function saveBehaviors(page: Page): Promise<ScenarioDraft> {
  const saved = page.waitForResponse(
    (response) => response.url().endsWith('/behaviors') && response.request().method() === 'PUT',
  );
  await page.getByTestId('save-behaviors').click();
  const response = await saved;
  expect(response.ok(), await response.text()).toBe(true);
  return response.json();
}

const expectedTimelines = [
  {
    name: 'application-resource-pressure',
    repeat_count: 1,
    phases: [
      {
        name: 'pressure',
        start_offset_ms: 5000,
        duration_ms: 15000,
        reset: true,
        faults: types.map((type, index) => ({
          device: 'DEMO-APP01',
          type,
          value: Number(values[index]),
        })),
      },
    ],
  },
];

function assertSavedDocument(original: ScenarioDraft, saved: ScenarioDraft) {
  const before: Record<string, unknown> = parse(original.content);
  const after: Record<string, unknown> = parse(saved.content);
  expect(after).toEqual({ ...before, behavior_timelines: expectedTimelines });
  expect(saved.revision).not.toBe(original.revision);
}

async function readSavedDraft(request: APIRequestContext, saved: ScenarioDraft) {
  const response = await request.get(`/api/v1/library/drafts/${encodeURIComponent(saved.name)}`);
  expect(response.ok(), await response.text()).toBe(true);
  const persisted: ScenarioDraft = await response.json();
  expect(persisted.content).toBe(saved.content);
  expect(persisted.revision).toBe(saved.revision);
  return persisted;
}

async function reopenSavedYaml(page: Page, content: string) {
  await selectInterface(page);
  await page.getByLabel('Upload local file').setInputFiles({
    name: 'resource-pressure-saved.yaml',
    mimeType: 'application/yaml',
    buffer: Buffer.from(content),
  });
  const reopened = await prepareDraft(page);
  expect(parse(reopened.content)).toEqual(parse(content));
  for (const [index, type] of types.entries()) {
    await expect(page.getByLabel('Fault', { exact: true }).nth(index)).toHaveValue(type);
    await expect(page.getByLabel('Utilization (%)', { exact: true }).nth(index)).toHaveValue(
      values[index],
    );
  }
}

test('persists resource faults and reopens saved YAML through the real daemon', async ({
  page,
  request,
}) => {
  await page.goto('/new-simulation');
  await selectInterface(page);
  await page.getByTestId('config-picker-search').fill('resource-pressure');
  await page.getByRole('button', { name: 'Select', exact: true }).click();
  const original = await prepareDraft(page);
  await editResourceValues(page);
  const saved = await saveBehaviors(page);
  assertSavedDocument(original, saved);
  await page.reload();
  const persisted = await readSavedDraft(request, saved);
  await reopenSavedYaml(page, persisted.content);
});
