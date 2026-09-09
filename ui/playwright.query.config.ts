import { defineConfig } from '@playwright/test';
import baseConfig from './playwright.config';

const baseURL = 'https://127.0.0.1:22445';
const desktopProjects = new Set(['chromium', 'webkit', 'firefox', 'chrome', 'edge']);

export default defineConfig({
  ...baseConfig,
  testMatch: 'shared-resources.query.ts',
  globalSetup: undefined,
  fullyParallel: false,
  workers: 1,
  reporter: [['list'], ['json', { outputFile: 'playwright-report/results-query.json' }]],
  projects: (baseConfig.projects ?? [])
    .filter((project) => desktopProjects.has(project.name ?? ''))
    .map(({ testMatch: _testMatch, testIgnore: _testIgnore, ...project }) => project),
  use: { ...baseConfig.use, baseURL, storageState: undefined, ignoreHTTPSErrors: true },
  webServer: {
    command:
      `cd .. && query_library=$(mktemp -d "\${TMPDIR:-/tmp}/niac-query.XXXXXX") && ` +
      "trap 'rm -rf \"$query_library\"' EXIT && trap 'exit 143' TERM && " +
      'NIAC_LIBRARY_ROOT="$query_library" NIAC_E2E_DRY_RUN_SIMULATION=1 ./niac daemon --listen 127.0.0.1:22445 ' +
      '--storage disabled --attachment-policy e2e-query-a=access:200 ' +
      '--attachment-policy e2e-query-b=access:201',
    url: `${baseURL}/__version`,
    reuseExistingServer: false,
    gracefulShutdown: { signal: 'SIGTERM', timeout: 5000 },
    timeout: 120000,
    ignoreHTTPSErrors: true,
  },
});
