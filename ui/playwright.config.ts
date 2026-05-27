import { defineConfig, devices } from '@playwright/test';

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
 * Browsers: Chromium (covers Chrome + Edge) and WebKit (covers Safari).
 * Per msn-docs-internal/05-Engineering/E2E_CONVENTIONS.md, no other browsers
 * or viewports are supported.
 */
export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  // retries 1 (not 2) — one retry is enough to dodge transient flakes; the
  //   second retry was costing ~30s × N flaky tests with no incremental signal.
  // workers 2 in CI (was 1) — GH Actions runners are 4-vCPU; 1 worker wastes
  //   75% of the box. Mirrors seed #1080 and the cross-repo perf push.
  retries: process.env.CI ? 1 : 0,
  workers: process.env.CI ? 2 : undefined,
  timeout: 30000,
  expect: {
    timeout: 10000,
  },
  reporter: [
    ['html', { outputFolder: 'playwright-report' }],
    ['list'],
    ['json', { outputFile: 'playwright-report/results.json' }],
  ],
  use: {
    baseURL: process.env.E2E_BASE_URL || 'http://localhost:5173',
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
    // Per msn-docs-internal/05-Engineering/E2E_CONVENTIONS.md, only chromium
    // (covers Chrome and Edge) and webkit (covers Safari) are supported.
    // Firefox/mobile/tablet/visual projects were deleted in 2026q2 — they
    // were configured but never run in CI, and the visual tier's macOS-only
    // *-darwin.png snapshots couldn't work on Linux CI.
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },
  ],
  // Run local dev server before tests if not in CI
  webServer: process.env.CI
    ? undefined
    : {
        command: 'npm run dev',
        url: 'http://localhost:5173',
        reuseExistingServer: !process.env.CI,
        timeout: 120000,
      },
});
