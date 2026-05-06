import { defineConfig, devices } from '@playwright/test';

const port = process.env.E2E_FULLSTACK_PORT ?? '18080';
const baseURL = process.env.E2E_BASE_URL ?? `http://localhost:${port}`;

export default defineConfig({
  testDir: './e2e/fullstack',
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
        command: `cd .. && go run ./cmd/niac daemon --listen :${port} --storage disabled`,
        url: `${baseURL}/__version`,
        reuseExistingServer: !process.env.CI,
        timeout: 120000,
      },
});
