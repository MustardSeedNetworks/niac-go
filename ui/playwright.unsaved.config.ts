import { defineConfig, devices } from '@playwright/test';
import base from './playwright.config';

const baseURL = 'https://127.0.0.1:20446';

export default defineConfig({
  ...base,
  testMatch: 'unsaved-changes.acceptance.ts',
  globalSetup: undefined,
  fullyParallel: false,
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'] } },
    { name: 'chrome', use: { ...devices['Desktop Chrome'], channel: 'chrome' } },
    { name: 'edge', use: { ...devices['Desktop Edge'], channel: 'msedge' } },
    { name: 'webkit', use: { ...devices['Desktop Safari'] } },
    { name: 'firefox', use: { ...devices['Desktop Firefox'] } },
  ],
  workers: 1,
  retries: 0,
  reporter: [['list'], ['json', { outputFile: 'playwright-report/results-unsaved.json' }]],
  use: { ...base.use, baseURL, storageState: undefined, ignoreHTTPSErrors: true },
  webServer: {
    command:
      `cd .. && unsaved_root=$(mktemp -d "\${TMPDIR:-/tmp}/niac-unsaved.XXXXXX") && ` +
      "trap 'rm -rf \"$unsaved_root\"' EXIT && trap 'exit 143' TERM && " +
      'NIAC_LIBRARY_ROOT="$unsaved_root/library" NIAC_CONFIGS_DIR="$unsaved_root/configs" ' +
      'NIAC_E2E_DRY_RUN_SIMULATION=1 ./niac daemon ' +
      '--listen 127.0.0.1:20446 --storage disabled --attachment-policy e2e-dry-run0=access:200',
    url: `${baseURL}/__version`,
    reuseExistingServer: false,
    gracefulShutdown: { signal: 'SIGTERM', timeout: 5000 },
    ignoreHTTPSErrors: true,
  },
});
