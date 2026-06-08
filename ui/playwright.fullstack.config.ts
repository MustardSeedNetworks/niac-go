import { defineConfig, devices } from '@playwright/test';

if (process.env.FORCE_COLOR) {
  delete process.env.NO_COLOR;
}

// Fullstack tier runs against a real niac daemon. Since #663 (Require HTTPS
// unconditionally), the daemon is HTTPS-only with an auto-generated self-signed
// cert. We keep port 18080 dedicated to the fullstack tier so it can coexist
// with a dev daemon on canonical 8445.
const port = process.env.E2E_FULLSTACK_PORT ?? '18080';
const host = '127.0.0.1';
const baseURL = process.env.E2E_BASE_URL ?? `https://${host}:${port}`;

export default defineConfig({
  testDir: './e2e/fullstack',
  // Keep commit metadata in reports, but avoid CI-only base-SHA fetches.
  captureGitInfo: { commit: true, diff: false },
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  timeout: 30000,
  expect: {
    timeout: 10000,
  },
  reporter: [
    ['html', { outputFolder: 'playwright-report/fullstack' }],
    ['list'],
    ['json', { outputFile: 'playwright-report/fullstack/results.json' }],
  ],
  use: {
    baseURL,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'on-first-retry',
    // Self-signed cert from auto-generation; required even in CI for this tier.
    ignoreHTTPSErrors: true,
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: process.env.E2E_BASE_URL
    ? undefined
    : {
        command: `cd .. && CGO_LDFLAGS_ALLOW='-Wl,-no_warn_duplicate_libraries' CGO_LDFLAGS='-Wl,-no_warn_duplicate_libraries' NIAC_E2E_DRY_RUN_SIMULATION=1 go run ./cmd/niac daemon --listen ${host}:${port} --storage disabled`,
        url: `${baseURL}/__version`,
        reuseExistingServer: !process.env.CI,
        timeout: 120000,
        ignoreHTTPSErrors: true,
      },
});
