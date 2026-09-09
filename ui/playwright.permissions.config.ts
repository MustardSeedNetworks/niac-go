import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './e2e',
  testMatch: 'permissions.auth.ts',
  workers: 1,
  retries: 0,
  reporter: [['list'], ['json', { outputFile: 'playwright-report/results-permissions.json' }]],
  use: { baseURL: 'https://127.0.0.1:18447', ignoreHTTPSErrors: true, trace: 'retain-on-failure' },
  projects: [{ name: 'chromium', use: devices['Desktop Chrome'] }],
  webServer: {
    command: 'node scripts/start-scoped-daemon.ts',
    url: 'https://127.0.0.1:18447/__version',
    ignoreHTTPSErrors: true,
    reuseExistingServer: false,
  },
});
