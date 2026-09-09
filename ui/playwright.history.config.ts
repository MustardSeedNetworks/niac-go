import { defineConfig, devices } from '@playwright/test';
import base from './playwright.config';

const baseURL = 'https://127.0.0.1:20445';

export default defineConfig({
  ...base,
  testMatch: 'run-history.acceptance.ts',
  globalSetup: undefined,
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  workers: 1,
  retries: 0,
  timeout: 120000,
  reporter: [['list'], ['json', { outputFile: 'playwright-report/results-history.json' }]],
  use: { ...base.use, baseURL, storageState: undefined, ignoreHTTPSErrors: true },
  webServer: {
    command:
      `cd .. && history_root=$(mktemp -d "\${TMPDIR:-/tmp}/niac-history.XXXXXX") && ` +
      "trap 'rm -rf \"$history_root\"' EXIT && trap 'exit 143' TERM && " +
      'go run ./tests/fixtures/history "$history_root/runs.db" && ' +
      'NIAC_LIBRARY_ROOT="$history_root/library" NIAC_E2E_DRY_RUN_SIMULATION=1 ./niac daemon ' +
      '--listen 127.0.0.1:20445 --storage "$history_root/runs.db" ' +
      '--attachment-policy e2e-dry-run0=access:200',
    url: `${baseURL}/__version`,
    reuseExistingServer: false,
    gracefulShutdown: { signal: 'SIGTERM', timeout: 5000 },
    ignoreHTTPSErrors: true,
  },
});
