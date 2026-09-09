import { spawn } from 'node:child_process';
import { once } from 'node:events';
import { mkdtemp, rm } from 'node:fs/promises';
import { createServer } from 'node:net';
import { tmpdir } from 'node:os';
import { resolve } from 'node:path';
import { expect, test } from '@playwright/test';

async function unusedPort(): Promise<number> {
  const server = createServer();
  server.listen(0, '127.0.0.1');
  await once(server, 'listening');
  const address = server.address();
  if (!address || typeof address === 'string') throw new Error('No TCP address assigned');
  await new Promise<void>((resolve) => server.close(() => resolve()));
  return address.port;
}

test('an idle daemon is neutral rather than all-clear', async ({ page, request }) => {
  // Other specs start scenarios. This daemon belongs only to this test.
  const root = await mkdtemp(resolve(tmpdir(), 'niac-idle-'));
  const baseURL = `https://127.0.0.1:${await unusedPort()}`;
  const daemon = spawn(
    resolve('../niac'),
    ['daemon', '--listen', new URL(baseURL).host, '--storage', 'disabled'],
    {
      env: {
        ...process.env,
        NIAC_CONFIGS_DIR: resolve(root, 'configs'),
        NIAC_LIBRARY_ROOT: resolve(root, 'library'),
      },
      stdio: 'ignore',
    },
  );
  const exited = once(daemon, 'exit');
  try {
    await expect
      .poll(async () => {
        if (daemon.exitCode !== null) throw new Error(`Idle daemon exited: ${daemon.exitCode}`);
        try {
          return (await request.get(`${baseURL}/__version`)).ok();
        } catch {
          return false;
        } // The listener is not bound during process startup.
      })
      .toBe(true);
    const response = await request.get(`${baseURL}/api/v1/simulation`);
    expect(response.ok()).toBe(true);
    expect(await response.json()).toMatchObject({ running: false });
    await page.goto(`${baseURL}/runtime`);
    const rollup = page.getByTestId('status-rollup');
    await expect(rollup).toHaveAttribute('data-state', 'idle');
    await expect(rollup).toContainText('No simulation is running');
    await expect(rollup).not.toContainText('All clear');
  } finally {
    daemon.kill('SIGTERM');
    await exited;
    await rm(root, { recursive: true });
  }
});
