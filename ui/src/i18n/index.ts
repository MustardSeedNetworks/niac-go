/**
 * i18n Configuration — NIAC
 *
 * Configures react-i18next for internationalization. Translations are
 * loaded from the shared Go-embedded locales directory at
 * /internal/i18n/locales/{lang}/{namespace}.json via the @locales Vite alias.
 *
 * Mirrors seed/stem so cross-product convention stays uniform — see
 * msn-docs-internal/05-Engineering/I18N_CONVENTIONS.md.
 *
 * Supported languages:
 * - English (en) — primary, fallback
 * - Spanish (es) — full parity required
 *
 * Usage in components:
 * ```tsx
 * import { useTranslation } from 'react-i18next';
 *
 * function MyComponent() {
 *   const { t } = useTranslation('common');
 *   return <button>{t('buttons.save')}</button>;
 * }
 * ```
 */

// Import English locale files
import enCommon from '@locales/en/common.json';
import enDevices from '@locales/en/devices.json';
import enErrors from '@locales/en/errors.json';
import enHelp from '@locales/en/help.json';
import enPages from '@locales/en/pages.json';
import enProtocols from '@locales/en/protocols.json';
import enSettings from '@locales/en/settings.json';
// Import Spanish locale files
import esCommon from '@locales/es/common.json';
import esDevices from '@locales/es/devices.json';
import esErrors from '@locales/es/errors.json';
import esHelp from '@locales/es/help.json';
import esPages from '@locales/es/pages.json';
import esProtocols from '@locales/es/protocols.json';
import esSettings from '@locales/es/settings.json';
import i18n, { type Resource } from 'i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import { initReactI18next } from 'react-i18next';

/**
 * Available languages configuration.
 */
export const languages = [
  { code: 'en', label: 'English', nativeLabel: 'English' },
  { code: 'es', label: 'Spanish', nativeLabel: 'Español' },
] as const;

export type LanguageCode = (typeof languages)[number]['code'];

/**
 * Translation namespaces. Order doesn't affect resolution but keeps
 * the bundle/types output deterministic.
 */
export const namespaces = [
  'common',
  'devices',
  'errors',
  'help',
  'pages',
  'protocols',
  'settings',
] as const;

export type Namespace = (typeof namespaces)[number];

/**
 * Default namespace used when none is specified.
 */
export const defaultNs: Namespace = 'common';

/**
 * Resources organized by language and namespace.
 */
const resources: Resource = {
  en: {
    common: enCommon,
    devices: enDevices,
    errors: enErrors,
    help: enHelp,
    pages: enPages,
    protocols: enProtocols,
    settings: enSettings,
  },
  es: {
    common: esCommon,
    devices: esDevices,
    errors: esErrors,
    help: esHelp,
    pages: esPages,
    protocols: esProtocols,
    settings: esSettings,
  },
};

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources,
    fallbackLng: 'en',
    supportedLngs: ['en', 'es'],
    defaultNS: defaultNs,
    ns: namespaces,

    detection: {
      // Order of language detection. localStorage wins so a user's
      // explicit Settings choice survives across sessions.
      order: ['localStorage', 'navigator', 'htmlTag'],
      // Cache user language preference
      caches: ['localStorage'],
      // localStorage key — namespaced to NIAC to avoid colliding with
      // a co-hosted seed/stem dev shell.
      lookupLocalStorage: 'niac-language',
    },

    interpolation: {
      // React already escapes values
      escapeValue: false,
    },

    debug: import.meta.env.DEV,
  })
  .catch(() => {
    // i18n initialization failure is non-recoverable; app will use
    // English key-fallbacks via the in-memory resources above.
  });

// Keep the document's lang attribute in sync with the active locale.
// Required for screen readers, search engines, browser spell-check, and
// CSS :lang() selectors. WCAG 3.1.1 + 3.1.2.
// Per msn-docs-internal/05-Engineering/I18N_CONVENTIONS.md.
if (typeof document !== 'undefined') {
  document.documentElement.lang = i18n.language;
  i18n.on('languageChanged', (lng) => {
    document.documentElement.lang = lng;
  });
}

export default i18n;

export type { TFunction } from 'i18next';
export { useTranslation } from 'react-i18next';
