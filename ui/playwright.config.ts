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
// The synthetic interface the dry-run daemon binds. Shared with the specs so
// the attachment policy below and the interface they select cannot drift.
export const e2eSimInterface = process.env.E2E_SIM_INTERFACE ?? 'e2e-dry-run0';
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
 * Engines: Chromium and WebKit — the two the product targets. Actual Safari
 * remains a manual release-candidate gate because Playwright drives WebKit
 * rather than Safari itself.
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
    // ── Supported breakpoint matrix ──────────────────────────────────────
    // This list is the contract, not an accumulation. The product is expected
    // to work on desktop, tablet and phone; before this, every project used a
    // `Desktop *` preset, so no phone or tablet layout was exercised on any
    // run and a mobile-only regression shipped green (#1320).
    //
    //   desktop  chromium · webkit · firefox   full suite
    //   tablet   iPad (gen 7)                  smoke subset
    //   phone    Pixel 7 · iPhone 15           smoke subset
    //
    // Real device presets, not `setViewportSize` inside a desktop project: a
    // narrow window is not a phone. The presets bring the right user agent,
    // touch support and input modality, which is what decides whether a
    // control is reachable at all.
    //
    // The small screens run only `*.mobile.spec.ts` — the app shell, primary
    // navigation and the main journey. Running the full suite on five
    // projects would multiply E2E wall-clock for coverage that is mostly
    // viewport-independent.
    {
      name: 'chromium',
      testIgnore: /.*\.mobile\.spec\.ts/,
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'webkit',
      // three-way-authoring drives a *started* simulation, and the daemon
      // serves one at a time -- a second concurrent start answers 409. Running
      // it on more than one engine at once would have the projects stopping
      // each other's session, which is a race, not coverage. The UI half of
      // the same journey runs on every engine via wizard-authoring.spec.ts.
      testIgnore: [/.*\.mobile\.spec\.ts/, /three-way-authoring\.spec\.ts/],
      use: { ...devices['Desktop Safari'] },
    },
    // Gecko. docs/WEBUI.md lists Firefox under "Engine CI — critical journeys
    // on relevant pull requests", but it was never in this list, so the only
    // independent-engine coverage the table promised did not exist. Chromium
    // and WebKit are Blink and WebKit; nothing here exercised a third engine.
    {
      name: 'firefox',
      // See the webkit note: one started simulation at a time.
      testIgnore: [/.*\.mobile\.spec\.ts/, /three-way-authoring\.spec\.ts/],
      use: { ...devices['Desktop Firefox'] },
    },
    // Edge, because the authoring plan's Definition of Complete names Chrome,
    // Edge and Safari as first-class authoring browsers and only two of the
    // three were covered. `channel: 'msedge'` drives the real installed Edge
    // rather than Chromium — the point is Edge's own integration, since the
    // engine underneath is already exercised by the chromium project.
    //
    // Scoped to the authoring journey the criterion actually names — compose a
    // scenario, edit a device, author a behaviour timeline — rather than the
    // whole suite. Edge and Chromium share Blink, so running everything twice
    // buys little; what is worth checking is that the journey completes in
    // Edge's own shell.
    {
      name: 'edge',
      testMatch: /(behavior-timeline|scenario-pack|device-editor)\.spec\.ts/,
      use: { ...devices['Desktop Edge'], channel: 'msedge' },
    },
    {
      name: 'tablet-safari',
      testMatch: /.*\.mobile\.spec\.ts/,
      use: { ...devices['iPad (gen 7)'] },
    },
    {
      name: 'mobile-chrome',
      testMatch: /.*\.mobile\.spec\.ts/,
      use: { ...devices['Pixel 7'] },
    },
    {
      name: 'mobile-safari',
      testMatch: /.*\.mobile\.spec\.ts/,
      use: { ...devices['iPhone 15'] },
    },
  ],
  // Explicit E2E_BASE_URL adopts an operator-managed daemon (CI uses 8445).
  // Otherwise Playwright owns the make-built HTTPS daemon and tears it down.
  webServer: process.env.E2E_BASE_URL
    ? undefined
    : {
        command:
          // The attachment policy is what lets an E2E reach *start*: a
          // binding with no approving policy fails preflight with
          // attachment_policy_denied, by design, so without this the
          // authoring journeys could only be driven as far as review.
          `cd .. && NIAC_E2E_DRY_RUN_SIMULATION=1 ./niac daemon --listen ${e2eHost}:${e2ePort} ` +
          `--storage disabled --attachment-policy ${e2eSimInterface}=access:200`,
        url: `${baseURL}/__version`,
        reuseExistingServer: !process.env.CI,
        timeout: 120000,
        ignoreHTTPSErrors: true,
      },
});
