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
 * Engines: Chromium, WebKit and Firefox, plus installed Chrome and Edge. Actual Safari
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
  // One retry diagnoses intermittent failures; CI's zero-flake budget still
  // rejects a retry-pass.
  // Leave capacity for browser rendering, trace capture and the real daemon.
  // Four workers delayed WebKit actionability enough to exhaust the wizard
  // journey's deadline even while its API requests remained responsive.
  retries: process.env.CI ? 1 : 0,
  workers: process.env.CI ? 2 : undefined,
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
    // Keep the failing attempt; a trace of a successful retry cannot explain a flake.
    trace: 'retain-on-failure',
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
    // ── The browser contract ─────────────────────────────────────────────
    // Chromium and WebKit, and nothing else. This is not a local preference:
    // `msn-docs-internal/05-Engineering/E2E_CONVENTIONS.md` is the single
    // source of truth for all four products and says so in as many words —
    // "No other browsers. Delete Firefox, mobile-chrome, mobile-safari,
    // tablet, edge from playwright.config.ts projects arrays. If a future
    // customer commitment requires another browser, file an issue and amend
    // this doc first." Chromium covers Chrome and Edge (one engine); WebKit
    // covers Safari. seed, stem and trellis have carried exactly these two
    // for months; niac was the last repo still running eight (#2246).
    //
    // What the extra projects actually bought: firefox contributed two
    // failures that were Gecko reporting `clientWidth: 0` for a span the
    // other two engines measure at 276px, and `make test-e2e-install`
    // installs chromium and webkit only — so on a clean machine the firefox
    // projects produced 119 instant "executable doesn't exist" failures and
    // taught everyone to ignore a red E2E run.
    //
    // Small screens did NOT go away with the phone and tablet projects. The
    // objection the old comment raised is right — a narrow window is not a
    // phone, because the user agent, touch support and input modality are
    // what decide whether a control is reachable — so `*.mobile.spec.ts`
    // keeps a real device preset via `test.use({ ...devices['Pixel 7'] })`,
    // applied inside chromium. The policy bans browser projects, not device
    // emulation, and Pixel 7 is a Chromium device, so that spec runs with the
    // touch and user agent it needs on a browser the policy allows.
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'webkit',
      // three-way-authoring drives a *started* simulation, and the daemon
      // serves one at a time -- a second concurrent start answers 409. Running
      // it on more than one engine at once would have the projects stopping
      // each other's session, which is a race, not coverage. The UI half of
      // the same journey runs on every engine via wizard-authoring.spec.ts.
      // `*.mobile.spec.ts` carries its own Chromium device preset, so it is
      // ignored here rather than being run at a desktop Safari viewport.
      testIgnore: [/.*\.mobile\.spec\.ts/, /three-way-authoring\.spec\.ts/],
      use: { ...devices['Desktop Safari'] },
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
          `cd .. && e2e_root=$(mktemp -d "\${TMPDIR:-/tmp}/niac-browser.XXXXXX") && ` +
          "trap 'rm -rf \"$e2e_root\"' EXIT && trap 'exit 143' TERM && " +
          'NIAC_LIBRARY_ROOT="$e2e_root/library" NIAC_CONFIGS_DIR="$e2e_root/configs" ' +
          `NIAC_E2E_DRY_RUN_SIMULATION=1 ./niac daemon --listen ${e2eHost}:${e2ePort} ` +
          `--storage disabled --attachment-policy ${e2eSimInterface}=access:200`,
        url: `${baseURL}/__version`,
        reuseExistingServer: false,
        gracefulShutdown: { signal: 'SIGTERM', timeout: 5000 },
        timeout: 120000,
        ignoreHTTPSErrors: true,
      },
});
