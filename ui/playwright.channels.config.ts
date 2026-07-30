import { defineConfig, devices } from '@playwright/test';
import baseConfig from './playwright.config';

const reportSuffix = process.env.PLAYWRIGHT_REPORT_SUFFIX ?? 'combined';
const reportDirectory = `playwright-report/browser-channels/${reportSuffix}`;

export default defineConfig({
  ...baseConfig,
  // Native channel processes accumulate renderer/GPU state over long suites.
  // One worker plus workflow sharding keeps each browser process bounded.
  workers: 1,
  reporter: [
    ['html', { outputFolder: reportDirectory }],
    ['list'],
    ['json', { outputFile: `${reportDirectory}/results.json` }],
  ],
  projects: [
    {
      name: 'chrome',
      use: { ...devices['Desktop Chrome'], channel: 'chrome' },
    },
    {
      name: 'edge',
      use: { ...devices['Desktop Edge'], channel: 'msedge' },
    },
  ],
});
