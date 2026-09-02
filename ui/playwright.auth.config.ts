import { defineConfig } from '@playwright/test';
import baseConfig from './playwright.config';

const authPort = '18446';
const authToken = 'niac-e2e-browser-auth-token'; // gitleaks:allow — local test fixture

export default defineConfig({
  ...baseConfig,
  // Spreading baseConfig carries its reporter array over, which would send this
  // suite's JSON to the same results.json the main run writes -- the second run
  // would overwrite the first and hide its flakes. Distinct file, so the flake
  // budget can glob both and see the whole picture.
  reporter: [['list'], ['json', { outputFile: 'playwright-report/results-auth.json' }]],
  testMatch: 'browser-auth.auth.ts',
  // Spreading baseConfig also carries its form-factor matrix, and a project's
  // OWN testMatch beats the config-level one above — so the small-screen
  // projects dragged e2e/app-shell.mobile.spec.ts into this suite, where it ran
  // against the token-gated daemon with storageState cleared and could not get
  // past the auth gate. This is a browser-auth check across the three engines;
  // it has no business running on a phone.
  //
  // Derived from baseConfig rather than restated, so adding an engine there
  // still reaches this suite. Projects carrying their own testMatch are the
  // form-factor ones and are dropped; testIgnore is stripped because the
  // config-level testMatch above already scopes this run to one file.
  projects: (baseConfig.projects ?? [])
    .filter((project) => !('testMatch' in project && project.testMatch))
    .map(({ testIgnore: _testIgnore, ...project }) => project),
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
