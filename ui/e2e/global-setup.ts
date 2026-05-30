import { mkdir, writeFile } from 'node:fs/promises';
import { dirname } from 'node:path';
import { AUTH_STORAGE_STATE } from './helpers/auth';

/**
 * Playwright globalSetup — runs ONCE per test process before any
 * test fixtures spawn.
 *
 * niac has no auth surface (loopback-only by default; no token
 * required, no login form), so there is no real "session" to mint.
 * What this file does is write an empty-but-well-formed storage state
 * to `AUTH_STORAGE_STATE` so that `playwright.config.ts`'s
 * `use.storageState` resolves without an EBUSY/ENOENT, AND so the
 * cross-repo specs that mirror the seed/stem shape (`test.use({
 * storageState: AUTH_STORAGE_STATE })`) keep working when ported.
 *
 * No login, no API calls — just a literal empty state file.
 */
export default async function globalSetup(): Promise<void> {
  const path = AUTH_STORAGE_STATE;
  await mkdir(dirname(path), { recursive: true });
  await writeFile(path, JSON.stringify({ cookies: [], origins: [] }, null, 2));
}
