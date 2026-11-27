import { mkdir, writeFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { expect, test } from '@playwright/test';
import { withIsolatedDaemon } from './isolated-daemon';

/**
 * The owner's path (#3154). A saved session from an earlier run cannot be
 * restored, so the daemon reports `recovery.state: failed`. The Simulation page
 * must still start a different scenario, the library network it picks must
 * start by name so the walks its `include_path: ../walks` names resolve, and
 * the stale session must be forgettable from the page instead of by editing a
 * root-owned file.
 */

const STALE_SESSION = 'modern';

// A library network that replays a starter capture the way a content pack does.
// The save API refuses `..`, so it goes onto disk the way a bundle install leaves it.
const STARTER_WALK = 'cisco-c2948-01.walk';

function packNetwork(walkName: string): string {
  return `include_path: ../walks
devices:
  - name: pack-switch
    type: switch
    mac: "02:00:00:00:ee:09"
    ips: [192.0.2.9]
    snmp_agent:
      community: public
      walk_file: ${walkName}
`;
}

async function seedStaleSession(root: string) {
  const networks = resolve(root, 'library', 'networks');
  await mkdir(networks, { recursive: true });
  await writeFile(resolve(networks, 'pack-net.yaml'), packNetwork(STARTER_WALK));
  const state = resolve(root, 'state');
  await mkdir(state, { recursive: true, mode: 0o700 });
  await writeFile(
    resolve(state, 'active-simulation.json'),
    JSON.stringify({
      schemaVersion: 3,
      savedAt: '2026-11-22T00:00:00Z',
      sessions: [
        {
          request: {
            sessionId: STALE_SESSION,
            interface: 'en16',
            configPath: resolve(root, 'configs', 'modern-hospital.yaml'),
          },
          generation: '00000000000000000000000000000001',
        },
      ],
    }),
    { mode: 0o600 },
  );
}

interface Status {
  running: boolean;
  recovery?: { state: string; failed?: { sessionId: string }[] };
}

test('starts a library network on loopback beside a failed saved session, then forgets the session', async ({
  page,
  request,
}) => {
  await withIsolatedDaemon(
    request,
    async (baseURL) => {
      // The fixture must really produce the failed state, or the rest proves nothing.
      const before = (await (await request.get(`${baseURL}/api/v1/simulation`)).json()) as Status;
      expect(before.recovery?.state).toBe('failed');
      expect(before.recovery?.failed?.map((s) => s.sessionId)).toEqual([STALE_SESSION]);

      await page.goto(`${baseURL}/runtime`);
      const card = page.getByTestId('saved-session-recovery-card');
      await expect(card).toBeVisible();
      await expect(page.getByTestId(`saved-session-${STALE_SESSION}`)).toContainText('en16');

      const iface = page.getByTestId('runtime-interface');
      const loopback = await iface
        .locator('option')
        .evaluateAll(
          (options) =>
            (options as HTMLOptionElement[]).find((option) => /^lo0?$/.test(option.value))?.value,
        );
      expect(loopback, 'the host has no loopback interface to start on').toBeDefined();
      await iface.selectOption(loopback ?? '');

      await page.getByTestId('config-picker-search').fill('pack-net');
      await page.getByRole('button', { name: 'Select', exact: true }).first().click();
      await page.getByTestId('runtime-prepare').click();
      await expect(page.getByTestId('wizard-attachments-error')).toHaveCount(0);
      await page.getByTestId('wizard-preflight-check').click();
      const started = page.waitForResponse(
        (response) =>
          response.url().endsWith('/api/v1/simulation') && response.request().method() === 'POST',
      );
      await page.getByTestId('wizard-preflight-start').click();
      expect((await started).status()).toBe(201);

      const running = (await (await request.get(`${baseURL}/api/v1/simulation`)).json()) as Status;
      expect(running.running).toBe(true);
      // Starting beside the failed record leaves it listed.
      expect(running.recovery?.failed?.map((s) => s.sessionId)).toEqual([STALE_SESSION]);

      await page.getByTestId(`forget-saved-session-${STALE_SESSION}`).click();
      await page.getByRole('dialog').getByRole('button', { name: 'Forget' }).click();
      await expect(card).toHaveCount(0);

      const after = (await (await request.get(`${baseURL}/api/v1/simulation`)).json()) as Status;
      expect(after.running).toBe(true);
      expect(after.recovery).toBeUndefined();
    },
    undefined,
    seedStaleSession,
  );
});

test('forgetting a session that did not fail recovery is refused', async ({ request }) => {
  await withIsolatedDaemon(
    request,
    async (baseURL) => {
      const { token } = (await (await request.get(`${baseURL}/api/v1/csrf-token`)).json()) as {
        token: string;
      };
      const headers = { 'X-Csrf-Token': token };
      const unknown = await request.delete(`${baseURL}/api/v1/simulation/recovery?sessionId=nope`, {
        headers,
      });
      expect(unknown.status()).toBe(404);
      const noCsrf = await request.delete(
        `${baseURL}/api/v1/simulation/recovery?sessionId=${STALE_SESSION}`,
      );
      expect(noCsrf.status()).toBe(403);
      const status = (await (await request.get(`${baseURL}/api/v1/simulation`)).json()) as Status;
      expect(status.recovery?.failed?.map((s) => s.sessionId)).toEqual([STALE_SESSION]);
    },
    undefined,
    seedStaleSession,
  );
});
