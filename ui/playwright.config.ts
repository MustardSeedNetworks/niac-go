import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright E2E Test Configuration
 *
 * End-to-end testing for NIAC user flows:
 * - Device management
 * - SNMP capture
 * - Template editing
 * - Replay functionality
 * - Network simulation
 *
 * Browsers: Chromium, Firefox, WebKit (Safari)
 * Viewports: Desktop, Tablet, Mobile
 */
export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  timeout: 30000,
  expect: {
    timeout: 10000,
  },
  reporter: [
    ['html', { outputFolder: 'playwright-report' }],
    ['list'],
    ['json', { outputFile: 'playwright-report/results.json' }],
  ],
  use: {
    baseURL: process.env.E2E_BASE_URL || 'http://localhost:5173',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'on-first-retry',
    ignoreHTTPSErrors: true,
  },
  projects: [
    // Desktop browsers
    {
      name: 'chromium',
      testIgnore: /visual\//,
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'firefox',
      testIgnore: /visual\//,
      use: { ...devices['Desktop Firefox'] },
    },
    {
      name: 'webkit',
      testIgnore: /visual\//,
      use: { ...devices['Desktop Safari'] },
    },
    // Mobile viewports
    {
      name: 'mobile-chrome',
      testIgnore: /visual\//,
      use: { ...devices['Pixel 5'] },
    },
    {
      name: 'mobile-safari',
      testIgnore: /visual\//,
      use: { ...devices['iPhone 12'] },
    },
    // Tablet viewport
    {
      name: 'tablet',
      testIgnore: /visual\//,
      use: { ...devices['iPad (gen 7)'] },
    },
    // Visual regression testing
    {
      name: 'visual',
      testDir: './e2e/visual',
      use: {
        ...devices['Desktop Chrome'],
        // Consistent viewport for visual comparison
        viewport: { width: 1280, height: 720 },
      },
      // Stricter comparison for visual tests
      expect: {
        toHaveScreenshot: {
          maxDiffPixels: 100,
          threshold: 0.1,
        },
      },
    },
  ],
  // Run local dev server before tests if not in CI
  webServer: process.env.CI
    ? undefined
    : {
        command: 'npm run dev',
        url: 'http://localhost:5173',
        reuseExistingServer: !process.env.CI,
        timeout: 120000,
      },
});
