import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { storybookTest } from '@storybook/addon-vitest/vitest-plugin';
import { playwright } from '@vitest/browser-playwright';
import { defineConfig } from 'vitest/config';

const currentDir = dirname(fileURLToPath(import.meta.url));

/**
 * Story files that do not yet pass the interaction/a11y run, excluded by path
 * so every other story is gated by default.
 *
 * A deny-list, not an allow-list. seed shipped the `tags: { include:
 * ['test-ready'] }` allow-list first and found that exactly one of its 88 story
 * files carried the tag — so the job proved the harness worked while covering
 * no real component, and every story written afterwards was ungated by
 * default. A deny-list inverts that: a new story is gated the moment it is
 * written, and anything skipped is visible here rather than invisible by
 * omission.
 *
 * Shrink this list; do not grow it.
 */
const NOT_YET_PASSING: string[] = [];

export default defineConfig({
  // aria-query is CommonJS, and @storybook/addon-vitest's setup file imports
  // `elementRoles` from it as a named ESM export. Browser mode serves it
  // unbundled unless Vite is told to pre-bundle it, and the run dies before a
  // single story mounts with "does not provide an export named 'elementRoles'".
  optimizeDeps: {
    include: [
      'aria-query',
      'lz-string',
      'pretty-format',
      '@testing-library/dom',
      '@testing-library/jest-dom',
      '@testing-library/react',
      '@testing-library/user-event',
    ],
  },
  test: {
    projects: [
      {
        extends: true,
        plugins: [
          storybookTest({
            configDir: resolve(currentDir, '.storybook'),
          }),
        ],
        test: {
          name: 'storybook',
          exclude: NOT_YET_PASSING,
          browser: {
            enabled: true,
            headless: true,
            provider: playwright({}),
            instances: [{ browser: 'chromium' }],
          },
        },
      },
    ],
  },
});
