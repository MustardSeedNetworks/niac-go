import { dirname, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import type { StorybookConfig } from '@storybook/react-vite';
import type { UserConfig } from 'vite';

const currentDir: string = dirname(fileURLToPath(import.meta.url));

const config: StorybookConfig = {
  stories: ['../src/**/*.mdx', '../src/**/*.stories.@(js|jsx|mjs|ts|tsx)'],
  addons: [
    '@chromatic-com/storybook',
    '@storybook/addon-vitest',
    '@storybook/addon-a11y',
    '@storybook/addon-docs',
    '@storybook/addon-onboarding',
  ],
  framework: '@storybook/react-vite',
  viteFinal: (viteConfig: UserConfig): UserConfig => {
    // OVERRIDE — don't merge aliases. Mirrors the seed pattern: the
    // runtime vite.config.ts may declare aliases as an array of
    // `{find: RegExp, replacement: string}` entries, but Storybook's
    // rolldown-based builder only accepts the object form. Spread-
    // merging the two crashes with "StringExpected on
    // BindingViteAliasPluginAlias.replacement". The override below
    // gives Storybook the shape it wants; runtime Vite is unaffected.
    return {
      ...viteConfig,
      resolve: {
        ...viteConfig.resolve,
        alias: {
          '@': resolve(currentDir, '../src'),
        },
      },
    };
  },
};
export default config;
