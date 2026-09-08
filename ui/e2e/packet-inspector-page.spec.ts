import { expect, test } from '@playwright/test';

/**
 * Packet Inspector Page (/packets) E2E
 *
 * Covers the live-packet capture / inspection surface:
 * - Page renders the "Packets" heading
 * - Page lands on /packets route
 */

test.describe('Packet Inspector Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/packets');
    await page.waitForLoadState('domcontentloaded');
  });

  test('should render the Packets heading', async ({ page }) => {
    await expect(page.getByTestId('page-header-title')).toBeVisible({
      timeout: 10000,
    });
  });

  test('should land on the /packets route', async ({ page }) => {
    await expect(page).toHaveURL(/\/packets$/);
  });
});

/**
 * U7: stopping a standalone capture ends the only source of live frames on
 * the page, so it must ask first. Both cases stub the capture as running —
 * the E2E daemon has no interface it may sniff.
 */
test.describe('Packet Inspector Page — stop capture confirmation', () => {
  test.beforeEach(async ({ page }) => {
    await page.route('**/api/v1/capture', async (route) => {
      if (route.request().method() === 'GET') {
        await route.fulfill({ json: { running: true, interface: 'eth0', filter: '' } });
        return;
      }
      await route.fulfill({ json: { status: 'stopped' } });
    });
    await page.goto('/packets');
    await page.waitForLoadState('domcontentloaded');
  });

  test('asks before stopping, and backing out leaves the capture running', async ({ page }) => {
    const stops: string[] = [];
    page.on('request', (request) => {
      if (request.url().includes('/api/v1/capture') && request.method() === 'DELETE') {
        stops.push(request.url());
      }
    });

    await page.getByRole('button', { name: 'Stop capture' }).click();

    const dialog = page.getByRole('dialog', { name: 'Stop the capture?' });
    await expect(dialog).toBeVisible();
    expect(stops).toHaveLength(0);

    await dialog.getByRole('button', { name: 'Cancel' }).click();
    await expect(dialog).toBeHidden();
    expect(stops).toHaveLength(0);
    await expect(page.getByRole('button', { name: 'Stop capture' })).toBeVisible();
  });

  test('confirming sends the stop', async ({ page }) => {
    const stopped = page.waitForRequest(
      (request) => request.url().includes('/api/v1/capture') && request.method() === 'DELETE',
    );

    await page.getByRole('button', { name: 'Stop capture' }).click();
    const dialog = page.getByRole('dialog', { name: 'Stop the capture?' });
    await dialog.getByRole('button', { name: 'Stop capture' }).click();

    await stopped;
    await expect(dialog).toBeHidden();
  });
});

/**
 * U7: an upload of a 100 MB capture must be stoppable. This drives the real
 * XHR abort path in requestUpload.ts against the served binary — the unit
 * test can only prove the page hands a signal down.
 *
 * The daemon writes nothing to disk for an upload (handlePcapUpload decodes
 * the body in memory and caches the analysis), so "no partial file" is
 * observable as: no analysis, no error, and the file still selected to retry.
 */
test.describe('PCAP analyzer — cancelling an upload', () => {
  test('cancel abandons the upload and leaves the page ready to retry', async ({ page }) => {
    // Never settle: the upload stays on the wire until the client aborts it.
    // Deliberately not fulfilled or aborted afterwards — the client aborts
    // first, and Playwright rejects a second disposition of the same route.
    await page.route('**/api/v1/pcap/upload', () => new Promise(() => {}));

    await page.goto('/packets?view=files');
    // Pick the file the way a person does — click the drop zone and answer the
    // file chooser it opens. Addressing the hidden <input> by CSS selector
    // instead ejected this PR from the merge queue twice: the selector matches
    // any input node that happens to be in the DOM, the change event is then
    // lost if React replaces that node, and the analyzer silently stays empty
    // (the disabled Analyze button below is what the failure snapshots showed).
    // The chooser is bound to the element that was clicked, so there is no
    // stale node to lose.
    await expect(page.getByRole('button', { name: 'Analyze PCAP' })).toBeDisabled();

    const chooser = page.waitForEvent('filechooser');
    await page.getByRole('button', { name: 'Drop PCAP file here or click to select' }).click();
    await (await chooser).setFiles({
      name: 'sample.pcap',
      mimeType: 'application/vnd.tcpdump.pcap',
      buffer: Buffer.from('sample capture bytes'),
    });
    await expect(page.getByText('sample.pcap')).toBeVisible();

    const uploaded = page.waitForRequest((request) =>
      request.url().includes('/api/v1/pcap/upload'),
    );
    await page.getByRole('button', { name: 'Analyze PCAP' }).click();
    await uploaded;

    await expect(page.getByTestId('pcap-upload-progress')).toBeVisible();
    await page.getByRole('button', { name: 'Cancel upload' }).click();

    await expect(page.getByTestId('pcap-upload-progress')).toBeHidden();
    // Ready to retry: the same file is still selected and Analyze is live again.
    await expect(page.getByRole('button', { name: 'Analyze PCAP' })).toBeEnabled();
    await expect(page.getByText('sample.pcap')).toBeVisible();
    // A cancel is not a failure, and nothing was analysed.
    await expect(page.getByRole('alert')).toHaveCount(0);
    await expect(page.getByTestId('pcap-upload-progress')).toHaveCount(0);
  });
});
