import { type ChildProcess, spawn } from 'node:child_process';
import { once } from 'node:events';
import { mkdtemp, rm } from 'node:fs/promises';
import { createServer } from 'node:net';
import { tmpdir } from 'node:os';
import { resolve } from 'node:path';
import { type APIRequestContext, expect } from '@playwright/test';

async function unusedPort(): Promise<number> {
  const server = createServer();
  server.listen(0, '127.0.0.1');
  await once(server, 'listening');
  const address = server.address();
  if (!address || typeof address === 'string') throw new Error('No TCP address assigned');
  await new Promise<void>((done) => server.close(() => done()));
  return address.port;
}

export async function withIsolatedDaemon(
  request: APIRequestContext,
  check: (baseURL: string) => Promise<void>,
) {
  const root = await mkdtemp(resolve(tmpdir(), 'niac-browser-isolated-'));
  try {
    const baseURL = `https://127.0.0.1:${await unusedPort()}`;
    const daemon = spawn(
      resolve('../niac'),
      [
        'daemon',
        '--listen',
        new URL(baseURL).host,
        '--storage',
        'disabled',
        '--cert-dir',
        resolve('../certs'),
        '--attachment-policy',
        'e2e-dry-run0=access:200',
      ],
      {
        env: {
          ...process.env,
          NIAC_CONFIGS_DIR: resolve(root, 'configs'),
          NIAC_LIBRARY_ROOT: resolve(root, 'library'),
          NIAC_E2E_DRY_RUN_SIMULATION: '1',
        },
        stdio: 'ignore',
      },
    );
    const closed = new Promise<void>((done) => daemon.once('close', () => done()));
    let processError: Error | undefined;
    daemon.on('error', (error: Error) => {
      processError = error;
    });
    try {
      await once(daemon, 'spawn');
      await expect
        .poll(async () => {
          if (processError) throw processError;
          if (daemon.exitCode !== null)
            throw new Error(`Isolated daemon exited: ${daemon.exitCode}`);
          try {
            return (await request.get(`${baseURL}/__version`)).ok();
          } catch {
            return false;
          } // Listener startup is asynchronous.
        })
        .toBe(true);
      await check(baseURL);
    } finally {
      await stopIsolatedDaemon(daemon, closed);
    }
  } finally {
    await rm(root, { recursive: true });
  }
}

async function stopIsolatedDaemon(daemon: ChildProcess, closed: Promise<void>) {
  daemon.kill('SIGTERM');
  if (await closedWithinDeadline(closed)) return;
  daemon.kill('SIGKILL');
  await closedWithinDeadline(closed);
  throw new Error('Isolated daemon shutdown exceeded five seconds');
}

async function closedWithinDeadline(closed: Promise<void>): Promise<boolean> {
  let timer: ReturnType<typeof setTimeout> | undefined;
  try {
    return await Promise.race([
      closed.then(() => true),
      new Promise<boolean>((done) => {
        timer = setTimeout(() => done(false), 5000);
      }),
    ]);
  } finally {
    clearTimeout(timer);
  }
}
