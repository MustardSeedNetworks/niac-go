import { type APIRequestContext, expect, test } from '@playwright/test';
import { ATTACHMENT, NETWORK_YAML, SIM_INTERFACE } from './fixtures/three-way-network';

async function recordRun(request: APIRequestContext) {
  const csrf = await request.get('/api/v1/csrf-token');
  expect(csrf.ok()).toBe(true);
  const { token } = (await csrf.json()) as { token: string };
  const headers = { 'X-Csrf-Token': token };
  const started = await request.post('/api/v1/simulation', {
    headers,
    data: {
      sessionId: 'history-acceptance',
      interface: SIM_INTERFACE,
      configData: NETWORK_YAML,
      attachment: ATTACHMENT.name,
      attachmentMode: ATTACHMENT.mode,
      accessVlan: ATTACHMENT.accessVlan,
    },
  });
  expect(started.ok(), await started.text()).toBe(true);
  const stopped = await request.delete('/api/v1/simulation?sessionId=history-acceptance', {
    headers,
  });
  expect(stopped.ok(), await stopped.text()).toBe(true);
}

test('all forty persisted runs are reachable through the shipped UI and API', async ({
  page,
  request,
}) => {
  const first = await request.get('/api/v1/history');
  expect(first.ok()).toBe(true);
  expect((await first.json()).map((record: { id: number }) => record.id)).toEqual(
    Array.from({ length: 20 }, (_, i) => 40 - i),
  );
  await page.goto('/runtime#recent-runs');
  const rows = page.getByTestId('history-run');
  await expect(rows).toHaveCount(20);
  await expect(rows.first()).toHaveAttribute('data-run-id', '40');
  await expect(rows.last()).toHaveAttribute('data-run-id', '21');
  await recordRun(request);
  await page.getByTestId('history-older').click();
  await expect(rows).toHaveCount(20);
  await expect(rows.first()).toHaveAttribute('data-run-id', '20');
  await expect(rows.last()).toHaveAttribute('data-run-id', '1');
  await expect(page.getByTestId('history-older')).toBeDisabled();
  await page.getByTestId('history-newer').click();
  await expect(rows.first()).toHaveAttribute('data-run-id', '41');
  await expect(page.getByTestId('history-newer')).toBeDisabled();
});
