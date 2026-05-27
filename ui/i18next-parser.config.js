/**
 * i18next-parser config — scans src/ for every t('namespace.key',
 * { …interpolation }) call site and produces the expected EN locale
 * shape, which CI compares against the committed
 * internal/i18n/locales/en/*.json.
 *
 * Why: catches typos in key names (you write t('packets.foobar') and
 * the validator doesn't see that the key doesn't exist), dead keys
 * (a t() call removed but the JSON entry forgotten), and missing
 * interpolation placeholders (e.g. key has {{count}} but the t() call
 * doesn't pass count).
 *
 * Two modes:
 *   - npm run i18n:extract — REWRITES locale files with the parsed
 *     shape; useful when adding new keys.
 *   - npm run i18n:check — fails on any drift (used in CI). It does
 *     NOT rewrite; just exits non-zero if the locale files don't match.
 *
 * Keep in sync with the namespaces list in src/i18n/index.ts and the
 * @locales Vite alias in vite.config.ts.
 */
export default {
  // Languages to extract for. We only generate EN keys; ES is human-
  // translated and validated separately by scripts/i18n/validate.sh
  // (key-parity check).
  locales: ['en'],

  // Where to write/read the locale files. Matches the @locales Vite
  // alias which points at ../internal/i18n/locales (relative to the
  // ui/ directory where this config file lives).
  output: '../internal/i18n/locales/$LOCALE/$NAMESPACE.json',

  // Which files to scan.
  input: ['src/**/*.{ts,tsx}'],

  // Skip generated/vendored content and tests.
  // i18n/index.ts and i18n/types.ts reference the namespaces themselves
  // but don't make t() calls; excluding them avoids noise.
  // *.stories.tsx render fixtures, not the real app.
  // *.test.tsx and __tests__ are unit tests; their t() calls (if any)
  // should be testing translation behavior, not introducing new keys.
  // help-content.ts is a large data module used by the help drawer —
  // its English prose isn't translatable via t() yet (queued for a
  // separate help.* namespace migration).
  // contextLines.ts / contextLines.tsx are deprecated test scaffolds.
  // Lazy-loaded route modules under pages/ are covered by the glob
  // unless explicitly excluded.

  // Namespaces — must match the ns: array in src/i18n/index.ts.
  // i18next-parser auto-detects ns from t('ns:key') or
  // useTranslation('ns'); we don't need to enumerate them here, but
  // we DO need defaultNamespace to know where t('key') without an
  // explicit ns prefix should land.
  defaultNamespace: 'common',

  // Recognize useTranslation('ns') as a scope marker.
  useKeysAsDefaultValue: false,

  // When a t() call has an unknown interpolation variable, leave the
  // existing value alone instead of clobbering with the empty default.
  // Pairs with the validator's interpolation-parity check.
  keepRemoved: false,

  // Pluralization config. Matches react-i18next's default _one/_other
  // postfixes that the validator's plural-completeness check enforces.
  defaultValue: '__STRING_NOT_TRANSLATED__',
  pluralSeparator: '_',
  contextSeparator: '_',
  keySeparator: '.',
  namespaceSeparator: ':',

  // Sort keys deterministically so the diff stays small.
  sort: true,

  // Skip the default lookup of keys without arguments — our codebase
  // consistently uses `t('foo.bar')` literal strings.
  lexers: {
    ts: ['JsxLexer'],
    tsx: ['JsxLexer'],
    default: ['JavascriptLexer'],
  },

  // Don't write key-value JSON files when running `--dry-run`. CI uses
  // `npm run i18n:check` which sets --fail-on-update so any drift fails
  // the build instead of silently rewriting.
  createOldCatalogs: false,
  resetDefaultValueLocale: null,

  verbose: false,
};
