/**
 * i18n Configuration
 *
 * Configures react-i18next for internationalization.
 * Translations are loaded from the shared locales directory
 * (`internal/i18n/locales`) via the `@locales` Vite alias — the
 * same directory the Go backend embeds via `go:embed`, so the
 * frontend and backend share a single source of truth.
 *
 * Supported languages:
 * - English (en) — default
 *
 * Industry-standard terms (protocol names like ARP, ICMP, DHCP,
 * DNS, SNMP; standards like RFC 2544, Y.1564; units like ms, Mbps;
 * abbreviations like IP, MAC, MTU, VLAN, CIDR) are NOT translated —
 * they are passed as `{{token}}` placeholders or kept verbatim in
 * every locale.
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

// Import English translations
import enCommon from '@locales/en/common.json';
import enDevices from '@locales/en/devices.json';
import enErrors from '@locales/en/errors.json';
import enHelp from '@locales/en/help.json';
import enPages from '@locales/en/pages.json';
import enProtocols from '@locales/en/protocols.json';
import enSettings from '@locales/en/settings.json';
import i18n, { type Resource } from 'i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import { initReactI18next } from 'react-i18next';

/**
 * Available languages configuration.
 *
 * Additional locales will be added under `internal/i18n/locales/<code>/`
 * and imported here. The English locale is the canonical source for
 * key shape; missing keys in other locales fall back to English.
 */
export const languages = [{ code: 'en', label: 'English', nativeLabel: 'English' }] as const;

export type LanguageCode = (typeof languages)[number]['code'];

/**
 * Translation namespaces.
 *
 * Each namespace maps to a JSON file under
 * `internal/i18n/locales/<lang>/<namespace>.json`.
 */
export const namespaces = [
  'common',
  'errors',
  'pages',
  'help',
  'settings',
  'devices',
  'protocols',
] as const;

export type Namespace = (typeof namespaces)[number];

/**
 * Default namespace used when none is specified.
 */
export const defaultNs: Namespace = 'common';

/**
 * Resources organised by language and namespace.
 */
const resources: Resource = {
  en: {
    common: enCommon,
    errors: enErrors,
    pages: enPages,
    help: enHelp,
    settings: enSettings,
    devices: enDevices,
    protocols: enProtocols,
  },
};

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    resources,
    fallbackLng: 'en',
    supportedLngs: ['en'],
    defaultNS: defaultNs,
    ns: namespaces,

    // Detection options
    detection: {
      // Order of language detection
      order: ['localStorage', 'navigator', 'htmlTag'],
      // Cache user language preference
      caches: ['localStorage'],
      // localStorage key
      lookupLocalStorage: 'niac-language',
    },

    interpolation: {
      // React already escapes values
      escapeValue: false,
    },

    // Debug mode in development
    debug: import.meta.env.DEV,
  })
  .catch(() => {
    // i18n initialisation failure is non-recoverable; app will use
    // the inline English fallback passed to t() as the second arg.
  });

export default i18n;

export type { TFunction } from 'i18next';
export { useTranslation } from 'react-i18next';
