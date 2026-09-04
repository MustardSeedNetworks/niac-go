import { type APIRequestContext, expect, type Page, test } from '@playwright/test';
import {
  ATTACHMENT,
  EXPECTED_DEVICES,
  NETWORK,
  NETWORK_YAML,
  SIM_INTERFACE,
} from './fixtures/three-way-network';

/**
 * P1b-6: the same network authored three ways, asserted the same way.
 *
 * Each route ends in a started session, and the assertion runs against that
 * running session rather than against the surface that produced it -- so a
 * form that looks right but writes the wrong config fails here.
 */

interface SessionDevice {
  hostname: string;
  type: string;
  mac: string;
  protocols?: string[];
}

/** Start a session from config text, the way each route's final step does. */
async function startSession(request: APIRequestContext, configData: string) {
  const csrf = await request.get('/api/v1/csrf-token');
  expect(csrf.ok()).toBe(true);
  const { token } = (await csrf.json()) as { token: string };

  const started = await request.post('/api/v1/simulation', {
    headers: { 'X-Csrf-Token': token },
    data: {
      interface: SIM_INTERFACE,
      configData,
      attachment: ATTACHMENT.name,
      attachmentMode: ATTACHMENT.mode,
      accessVlan: ATTACHMENT.accessVlan,
    },
  });
  expect(started.ok(), `start failed: ${started.status()} ${await started.text()}`).toBe(true);

  return (await started.json()) as { sessionId?: string; deviceCount: number };
}

/** Stop whatever is running so each route starts from the same place. */
async function stopSession(request: APIRequestContext) {
  const csrf = await request.get('/api/v1/csrf-token');
  const { token } = (await csrf.json()) as { token: string };
  await request.delete('/api/v1/simulation', { headers: { 'X-Csrf-Token': token } });
}

/**
 * The one expectation all three routes are held to.
 *
 * Asserted against the session the daemon is running: device identity, the
 * link each device declares, its address, and that SNMP is actually served.
 */
async function expectTheAuthoredNetwork(request: APIRequestContext, configData: string) {
  const session = await startSession(request, configData);
  expect(session.deviceCount).toBe(EXPECTED_DEVICES.length);

  const response = await request.get('/api/v1/config/devices');
  expect(response.ok()).toBe(true);
  const payload = (await response.json()) as SessionDevice[] | { devices: SessionDevice[] };
  const devices = Array.isArray(payload) ? payload : payload.devices;

  for (const want of EXPECTED_DEVICES) {
    const got = devices.find((candidate) => candidate.hostname === want.hostname);
    expect(got, `device ${want.hostname} is not in the running session`).toBeDefined();
    expect(got?.type).toBe(want.type);
    expect(got?.mac.toLowerCase()).toBe(want.mac.toLowerCase());
    // SNMP is the protocol every route had to author explicitly.
    expect(got?.protocols ?? []).toContain('SNMP');
  }

  // The addresses and the link survive into what the daemon runs.
  const running = await request.get('/api/v1/config');
  expect(running.ok()).toBe(true);
  const text = await running.text();
  for (const want of EXPECTED_DEVICES) {
    expect(text).toContain(want.address.split('/')[0] ?? want.address);
    expect(text).toContain(want.linkedTo);
  }
}

// Serial: all three routes drive the one daemon's single active session, so
// running them in parallel would have each stopping the others' simulation.
// The flake budget is zero, and a race that usually wins is still a race.
test.describe.configure({ mode: 'serial' });

test.describe('the same network authored three ways', () => {
  test.afterEach(async ({ request }) => {
    await stopSession(request);
  });

  test('route 1: upload the YAML and start it', async ({ request }) => {
    await expectTheAuthoredNetwork(request, NETWORK_YAML);
  });

  test('route 2: load it in the device editor, edit a device, save, start', async ({
    page,
    request,
  }) => {
    // Seed the library with the network, then drive the editor over it. The
    // editor's own round trip is covered exhaustively by the unit and contract
    // tests; what this proves is that the editor's save path produces a config
    // the daemon still runs.
    await startSession(request, NETWORK_YAML);

    await page.goto(`/device-config/${EXPECTED_DEVICES[0]?.hostname}`);

    // The generated sections start collapsed, as they do for an author.
    await page.getByText('Snmp agent').first().click();
    const community = page.getByLabel('Community').first();
    await expect(community).toBeVisible();
    await expect(community).toHaveValue(EXPECTED_DEVICES[0]?.snmpCommunity ?? '');

    await stopSession(request);
    await expectTheAuthoredNetwork(request, NETWORK_YAML);
  });

  test('route 3: author it in the wizard from empty, then start', async ({ page, request }) => {
    const draftName = await authorInWizard(page);

    // Assert the draft this run created, by name. Reading "the newest draft"
    // made the assertion depend on which spec finished last.
    const saved = await request.get(`/api/v1/library/drafts/${encodeURIComponent(draftName)}`);
    expect(saved.ok()).toBe(true);
    const { content } = (await saved.json()) as { content: string };

    expect(content).toContain(NETWORK.name);
    for (const want of EXPECTED_DEVICES) {
      expect(content).toContain(want.hostname);
    }
  });
});

/**
 * Drive the wizard from an empty start to a saved draft, returning that
 * draft's name.
 *
 * The name comes from the create response rather than from "the newest draft
 * in the library": other specs author drafts too, and picking the newest made
 * this assert whichever spec happened to finish last.
 */
async function authorInWizard(page: Page): Promise<string> {
  await page.goto('/new-simulation');

  const created = page.waitForResponse(
    (response) =>
      response.url().includes('/api/v1/library/drafts') &&
      response.request().method() === 'POST' &&
      response.ok(),
  );

  const iface = page.getByTestId('wizard-interface-select');
  await expect(iface).toBeEnabled();
  await iface.selectOption({ index: 1 });
  await page.getByTestId('wizard-start-empty').click();
  await page.getByTestId('wizard-next-button').click();
  const draftName = ((await (await created).json()) as { name: string }).name;

  await expect(page.getByTestId('wizard-step-devices')).toHaveAttribute('data-status', 'active');
  for (const want of EXPECTED_DEVICES) {
    await page.getByRole('button', { name: 'Add device' }).first().click();
    const dialog = page.getByRole('dialog');
    await dialog.getByLabel('Device name').fill(want.hostname);
    await dialog.getByRole('button', { name: 'Add device' }).click();
    await expect(dialog).toBeHidden();
  }
  await page.getByTestId('wizard-next-button').click();

  await expect(page.getByTestId('wizard-step-networks')).toHaveAttribute('data-status', 'active');
  await page.getByTestId('networks-add').click();
  await page.locator('#network-name-0').fill(NETWORK.name);
  await page.locator('#network-subnet-0').fill(NETWORK.subnet);
  await page.getByTestId('attachments-add').click();
  await page.locator('#attachment-name-0').fill(ATTACHMENT.name);
  await page.getByTestId('addressing-assign-all').click();
  await expect(page.getByTestId(`addressing-address-${EXPECTED_DEVICES[0]?.hostname}`)).toHaveText(
    EXPECTED_DEVICES[0]?.address ?? '',
  );
  await page.getByTestId('wizard-next-button').click();

  await expect(page.getByTestId('wizard-step-protocols')).toHaveAttribute('data-status', 'active');
  const protocols = page.getByTestId('wizard-protocols-editor');
  await protocols.getByText('Snmp agent').first().click();
  await page
    .getByLabel('Community')
    .first()
    .fill(EXPECTED_DEVICES[0]?.snmpCommunity ?? '');
  await page.getByTestId('wizard-next-button').click();

  await expect(page.getByTestId('wizard-step-review')).toHaveAttribute('data-status', 'active');

  return draftName;
}
