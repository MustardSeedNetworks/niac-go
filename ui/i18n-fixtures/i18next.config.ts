import { defineConfig } from 'i18next-cli';

export default defineConfig({
  locales: ['en'],
  extract: {
    input: ['i18n-fixtures/source.tsx'],
    output: 'i18n-fixtures/locales/{{language}}/{{namespace}}.json',
    defaultNS: 'common',
    primaryLanguage: 'en',
    preservePatterns: ['common:dynamic.*', 'pages:dynamic.*'],
    removeUnusedKeys: true,
    sort: true,
  },
});
