/**
 * i18n Configuration
 *
 * Configures react-i18next for NIAC. Translations are loaded from the
 * shared locales directory (internal/i18n/locales) via the @locales Vite
 * alias.
 *
 * Supported languages:
 * - English (en) - default
 * - Spanish (es)
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

import enCommon from '@locales/en/common.json';
import enDevices from '@locales/en/devices.json';
import enErrors from '@locales/en/errors.json';
import enHelp from '@locales/en/help.json';
import enPages from '@locales/en/pages.json';
import enProtocols from '@locales/en/protocols.json';
import enSettings from '@locales/en/settings.json';
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

export const languages = [
  { code: 'en', label: 'English', nativeLabel: 'English' },
  { code: 'es', label: 'Spanish', nativeLabel: 'Español' },
] as const;

export type LanguageCode = (typeof languages)[number]['code'];

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

export const defaultNs: Namespace = 'common';

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
      order: ['localStorage', 'navigator', 'htmlTag'],
      caches: ['localStorage'],
      lookupLocalStorage: 'niac-language',
    },

    interpolation: {
      escapeValue: false,
    },

    debug: import.meta.env.DEV,
  })
  .catch(() => {
    // i18n initialization failure is non-recoverable, app will use fallback strings
  });

// Keep the document's lang attribute in sync with the active locale.
// Required for screen readers, search engines, browser spell-check, and
// CSS :lang() selectors. WCAG 3.1.1/3.1.2.
if (typeof document !== 'undefined') {
  document.documentElement.lang = i18n.language;
  i18n.on('languageChanged', (lng) => {
    document.documentElement.lang = lng;
  });
}

export default i18n;

export type { TFunction } from 'i18next';
export { useTranslation } from 'react-i18next';
