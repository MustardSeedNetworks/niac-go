import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { defineConfig } from 'i18next-cli';

const repositoryDynamicPatterns = readFileSync(
  resolve(import.meta.dirname, '../scripts/i18n/dynamic-prefixes.txt'),
  'utf8',
)
  .split('\n')
  .map((line) => line.split('#', 1)[0].trim())
  .filter(Boolean)
  .map((prefix) => `${prefix}*`);

export default defineConfig({
  locales: ['en'],
  extract: {
    input: ['src/**/*.{ts,tsx}'],
    output:
      process.env.I18NEXT_OUTPUT ?? '../internal/i18n/locales/{{language}}/{{namespace}}.json',
    ignore: [
      'src/**/*.test.{ts,tsx}',
      'src/**/*.stories.tsx',
      'src/**/__tests__/**',
      'src/data/help-content.ts',
      'src/i18n/index.ts',
      'src/i18n/types.ts',
      // The page registry addresses every key through `pages.{i18nKey}.*`
      // template literals. Resolving those would have the extractor invent an
      // `eyebrow` for every page, and an eyebrow is opt-in — a page has one
      // only where its locale declares it. Every `pages:` prefix is already
      // listed in scripts/i18n/dynamic-prefixes.txt, so these keys are
      // preserved rather than pruned.
      'src/pageRegistry.ts',
      // The generated device-editor forms label every field through
      // `editor.fields.{path}`, where the path comes from the schema manifest.
      // The extractor cannot resolve that template literal and invents a
      // literal `<path>` key for it. `devices:editor.` is already a preserved
      // dynamic prefix, so this file's own static chrome keys (addEntry,
      // removeEntry, mapKey, mapValue, unset) survive the prune.
      'src/components/device-editor/SchemaFields.tsx',
    ],
    functions: ['t', '*.t', 'i18next.t'],
    useTranslationNames: ['useTranslation'],
    defaultNS: 'common',
    nsSeparator: ':',
    keySeparator: '.',
    contextSeparator: '_',
    pluralSeparator: '_',
    preservePatterns: [
      'common:format.*',
      'settings:appearance.*',
      'settings:simulation.*',
      'settings:tabs.*',
      ...repositoryDynamicPatterns,
    ],
    primaryLanguage: 'en',
    defaultValue: '__STRING_NOT_TRANSLATED__',
    removeUnusedKeys: true,
    sort: true,
    indentation: 2,
  },
});
