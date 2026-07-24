import { defineConfig, devices } from '@playwright/test';
import baseConfig from './playwright.config';

export default defineConfig({
  ...baseConfig,
  reporter: [
    ['html', { outputFolder: 'playwright-report/browser-channels' }],
    ['list'],
    ['json', { outputFile: 'playwright-report/browser-channels/results.json' }],
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
