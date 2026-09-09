import { mkdtempSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { defineConfig, devices } from '@playwright/test';
import base from './playwright.config';

const baseURL = 'https://127.0.0.1:20446';

export default defineConfig({
  ...base,
  testMatch: 'unsaved-changes.acceptance.ts',
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  workers: 1,
  retries: 0,
  reporter: [['list'], ['json', { outputFile: 'playwright-report/results-unsaved.json' }]],
  use: { ...base.use, baseURL, ignoreHTTPSErrors: true },
  webServer: {
    env: { NIAC_LIBRARY_ROOT: mkdtempSync(join(tmpdir(), 'niac-unsaved-library-')) },
    command:
      'cd .. && NIAC_E2E_DRY_RUN_SIMULATION=1 ./niac daemon --listen 127.0.0.1:20446 --storage disabled --attachment-policy e2e-dry-run0=access:200',
    url: `${baseURL}/__version`,
    reuseExistingServer: false,
    ignoreHTTPSErrors: true,
  },
});
