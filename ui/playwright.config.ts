import { defineConfig, devices } from '@playwright/test';

if (process.env.FORCE_COLOR) {
  delete process.env.NO_COLOR;
}

const e2ePort = process.env.E2E_PORT ?? '18445';
const e2ePortNumber = Number(e2ePort);
if (!/^\d+$/.test(e2ePort) || e2ePortNumber < 1 || e2ePortNumber > 65535) {
  throw new Error('E2E_PORT must be a numeric TCP port between 1 and 65535');
}
const e2eHost = '127.0.0.1';
const baseURL = process.env.E2E_BASE_URL ?? `https://${e2eHost}:${e2ePort}`;

/**
 * Playwright E2E Test Configuration
 *
 * End-to-end testing for NIAC user flows:
 * - Device management
 * - SNMP capture
 * - Template editing
 * - Replay functionality
 * - Network simulation
 *
 * Engines: Chromium, WebKit, and Firefox. Installed Chrome and Edge channels
 * use playwright.channels.config.ts; actual Safari remains a manual release
 * candidate gate because Playwright controls WebKit rather than Safari.
 */
export default defineConfig({
  testDir: './e2e',
  // Playwright 1.60 captures git diffs by default in CI. On PRs this runs a
  // shallow fetch for the base SHA, which can fail before tests execute.
  captureGitInfo: { commit: true, diff: false },
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  // retries 1 (not 2) — one retry is enough to dodge transient flakes; the
  //   second retry was costing ~30s × N flaky tests with no incremental signal.
  // workers 4 in CI (bumped from 2 in PR-N1) — GH Actions ubuntu-latest is
  //   4-vCPU. fullyParallel + workers=4 fills the box and roughly halves
  //   per-shard wall-clock under the seed cross-repo perf pattern.
  retries: process.env.CI ? 1 : 0,
  workers: process.env.CI ? 4 : undefined,
  timeout: 30000,
  expect: {
    timeout: 10000,
  },
  globalSetup: './e2e/global-setup.ts',
  reporter: [
    ['html', { outputFolder: 'playwright-report' }],
    ['list'],
    ['json', { outputFile: 'playwright-report/results.json' }],
  ],
  use: {
    baseURL,
    storageState: 'playwright/.auth/user.json',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'on-first-retry',
    // Default: gated to local dev only. CI MUST hit real TLS per
    // E2E_CONVENTIONS. The PLAYWRIGHT_IGNORE_HTTPS_ERRORS env var is
    // the documented escape hatch — used in CI when the backend's
    // self-signed cert can't be added to the runner's trust store
    // (most cases today, since the daemon auto-generates a self-signed
    // cert on first start with no CA-signing step). The CI workflow
    // sets this env var explicitly; locally it's unset and dev
    // ignores HTTPS errors implicitly.
    ignoreHTTPSErrors: process.env.PLAYWRIGHT_IGNORE_HTTPS_ERRORS === 'true' || !process.env.CI,
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
  ],
  // Explicit E2E_BASE_URL adopts an operator-managed daemon (CI uses 8445).
  // Otherwise Playwright owns the make-built HTTPS daemon and tears it down.
  webServer: process.env.E2E_BASE_URL
    ? undefined
    : {
        command: `cd .. && NIAC_E2E_DRY_RUN_SIMULATION=1 ./niac daemon --listen ${e2eHost}:${e2ePort} --storage disabled`,
        url: `${baseURL}/__version`,
        reuseExistingServer: !process.env.CI,
        timeout: 120000,
        ignoreHTTPSErrors: true,
      },
});
