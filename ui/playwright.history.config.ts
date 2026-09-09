import { mkdtempSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { defineConfig, devices } from '@playwright/test';
import base from './playwright.config';

const directory = mkdtempSync(join(tmpdir(), 'niac-history-'));
const baseURL = 'https://127.0.0.1:20445';

export default defineConfig({
  ...base,
  testMatch: 'run-history.acceptance.ts',
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  workers: 1,
  retries: 0,
  timeout: 120000,
  reporter: [['list'], ['json', { outputFile: 'playwright-report/results-history.json' }]],
  use: { ...base.use, baseURL, ignoreHTTPSErrors: true },
  webServer: {
    command: `cd .. && go run ./tests/fixtures/history '${directory}/runs.db' && NIAC_E2E_DRY_RUN_SIMULATION=1 ./niac daemon --listen 127.0.0.1:20445 --storage '${directory}/runs.db' --attachment-policy e2e-dry-run0=access:200`,
    url: `${baseURL}/__version`,
    reuseExistingServer: false,
    ignoreHTTPSErrors: true,
  },
});
