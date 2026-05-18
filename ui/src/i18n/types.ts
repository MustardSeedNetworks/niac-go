/**
 * i18n TypeScript Types
 *
 * Provides type-safe translation keys for react-i18next. The types
 * are derived directly from the English locale JSON files, so adding
 * a new key to `en/<namespace>.json` automatically widens the
 * `t()` autocomplete set.
 *
 * Usage:
 * ```tsx
 * import { useTranslation } from 'react-i18next';
 *
 * function MyComponent() {
 *   const { t } = useTranslation('common');
 *   // TypeScript will autocomplete 'buttons.save', 'status.connected', etc.
 *   return <button>{t('buttons.save')}</button>;
 * }
 * ```
 */

import type enCommon from '@locales/en/common.json';
import type enDevices from '@locales/en/devices.json';
import type enErrors from '@locales/en/errors.json';
import type enHelp from '@locales/en/help.json';
import type enPages from '@locales/en/pages.json';
import type enProtocols from '@locales/en/protocols.json';
import type enSettings from '@locales/en/settings.json';

/**
 * Type definitions for each namespace.
 */
export type CommonTranslations = typeof enCommon;
export type ErrorsTranslations = typeof enErrors;
export type PagesTranslations = typeof enPages;
export type HelpTranslations = typeof enHelp;
export type SettingsTranslations = typeof enSettings;
export type DevicesTranslations = typeof enDevices;
export type ProtocolsTranslations = typeof enProtocols;

/**
 * All translations combined.
 */
export interface Translations {
  common: CommonTranslations;
  errors: ErrorsTranslations;
  pages: PagesTranslations;
  help: HelpTranslations;
  settings: SettingsTranslations;
  devices: DevicesTranslations;
  protocols: ProtocolsTranslations;
}

/**
 * Declaration merging for react-i18next.
 * This enables autocomplete for translation keys.
 */
declare module 'i18next' {
  interface CustomTypeOptions {
    defaultNS: 'common';
    resources: Translations;
  }
}
