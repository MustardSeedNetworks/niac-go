import { defineConfig } from '@playwright/test';
import baseConfig from './playwright.config';

const authPort = '18446';
const authToken = 'niac-e2e-browser-auth-token'; // gitleaks:allow — local test fixture

export default defineConfig({
  ...baseConfig,
  testMatch: 'browser-auth.auth.ts',
  fullyParallel: false,
  workers: 1,
  use: {
    ...baseConfig.use,
    baseURL: `https://127.0.0.1:${authPort}`,
    ignoreHTTPSErrors: true,
    storageState: undefined,
  },
  webServer: {
    command: `cd .. && NIAC_API_TOKEN=${authToken} ./niac daemon --listen 127.0.0.1:${authPort} --storage disabled`,
    url: `https://127.0.0.1:${authPort}/__version`,
    reuseExistingServer: false,
    timeout: 120000,
    ignoreHTTPSErrors: true,
  },
});
