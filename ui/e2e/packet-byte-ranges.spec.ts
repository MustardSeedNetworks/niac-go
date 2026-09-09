import { expect, test } from '@playwright/test';

function taggedCapture(): Buffer {
  const frame = Buffer.from(
    '00112233445566778899aabb810000c80800450000280000000040060000c0000201c6336402005001bb00000001000000005002200000000000',
    'hex',
  );
  const header = Buffer.from('d4c3b2a1020004000000000000000000ffff000001000000', 'hex');
  const record = Buffer.alloc(16);
  record.writeUInt32LE(1700000000, 0);
  record.writeUInt32LE(frame.length, 8);
  record.writeUInt32LE(frame.length, 12);
  return Buffer.concat([header, record, frame]);
}

test('real capture upload highlights the decoded tagged TCP bytes', async ({ page }) => {
  await page.goto('/packets?view=files');
  const chooser = page.waitForEvent('filechooser');
  await page.getByRole('button', { name: 'Drop PCAP file here or click to select' }).click();
  await (await chooser).setFiles({
    name: 'tagged-tcp.pcap',
    mimeType: 'application/vnd.tcpdump.pcap',
    buffer: taggedCapture(),
  });
  const uploaded = page.waitForResponse('**/api/v1/pcap/upload');
  const analyzed = page.waitForResponse(
    (response) =>
      /\/api\/v1\/pcap\/[a-f0-9]+$/.test(response.url()) && response.request().method() === 'GET',
  );
  await page.getByRole('button', { name: 'Analyze PCAP' }).click();
  const response = await uploaded;
  expect(response.status()).toBe(200);
  const analysis = await analyzed;
  expect(analysis.status()).toBe(200);
  const body = (await analysis.json()) as {
    packets: { byteRanges: { layer: string; field: string; start: number; end: number }[] }[];
  };
  expect(body.packets[0]?.byteRanges).toContainEqual({
    layer: 'tcp',
    field: 'Source Port',
    start: 38,
    end: 40,
  });
  await page.getByTestId('pcap-packet-1').click();
  await page.getByRole('button', { name: 'Source Port: 80', exact: true }).click();
  await expect(page.getByTestId('hex-byte-38')).toHaveAttribute('data-highlighted', 'true');
  await expect(page.getByTestId('hex-byte-39')).toHaveAttribute('data-highlighted', 'true');
  await expect(page.getByTestId('hex-byte-34')).toHaveAttribute('data-highlighted', 'false');
});

test('pause retains incoming SSE packets and reports eviction', async ({ page }) => {
  await page.route('**/api/v1/simulation', (route) =>
    route.fulfill({ json: { running: false, sessions: [] } }),
  );
  await page.route('**/api/v1/capture', (route) =>
    route.fulfill({ json: { running: true, interface: 'eth0' } }),
  );
  const ready = Promise.withResolvers<void>();
  const release = Promise.withResolvers<void>();
  let delivered = false;
  await page.route('**/api/v1/stream/packets', async (route) => {
    ready.resolve();
    await release.promise;
    const body = delivered
      ? ''
      : Array.from(
          { length: 150 },
          (_, index) =>
            `data: ${JSON.stringify({ type: 'packet', timestamp: '2026-09-09T12:00:00Z', data: { protocol: 'UDP', size: 64, summary: `buffer-${index}` } })}\n\n`,
        ).join('');
    delivered = true;
    await route.fulfill({ contentType: 'text/event-stream', body });
  });
  await page.goto('/packets');
  await ready.promise;
  await page.getByRole('button', { name: 'Pause', exact: true }).click();
  release.resolve();
  await expect(page.getByTestId('packet-buffer-pending')).toHaveText('150 new');
  await expect(page.getByTestId('packet-buffer-evictions')).toHaveText(
    '50 evicted · 100 packet limit',
  );
  await expect(page.getByText('0 / 0 packets')).toBeVisible();
  await page.getByRole('button', { name: 'Resume', exact: true }).click();
  await expect(page.getByText('100 / 100 packets')).toBeVisible();
  await expect(page.getByText('buffer-149', { exact: true })).toBeVisible();
  await expect(page.getByText('buffer-0', { exact: true })).toHaveCount(0);
});
