import { defineConfig, devices } from '@playwright/test';
import authConfig from './playwright.auth.config';

export default defineConfig({
  ...authConfig,
  reporter: [
    ['html', { outputFolder: 'playwright-report/browser-channel-auth' }],
    ['list'],
    ['json', { outputFile: 'playwright-report/browser-channel-auth/results.json' }],
  ],
  projects: [
    {
      name: 'chrome',
      use: { ...authConfig.use, ...devices['Desktop Chrome'], channel: 'chrome' },
    },
    {
      name: 'edge',
      use: { ...authConfig.use, ...devices['Desktop Edge'], channel: 'msedge' },
    },
  ],
});
