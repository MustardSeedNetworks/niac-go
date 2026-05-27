/**
 * i18n TypeScript Types
 *
 * Provides type-safe translation keys for react-i18next via declaration
 * merging. Source of truth is the English locale files.
 */

import type enCommon from '@locales/en/common.json';
import type enDevices from '@locales/en/devices.json';
import type enErrors from '@locales/en/errors.json';
import type enHelp from '@locales/en/help.json';
import type enPages from '@locales/en/pages.json';
import type enProtocols from '@locales/en/protocols.json';
import type enSettings from '@locales/en/settings.json';

export type CommonTranslations = typeof enCommon;
export type DevicesTranslations = typeof enDevices;
export type ErrorsTranslations = typeof enErrors;
export type HelpTranslations = typeof enHelp;
export type PagesTranslations = typeof enPages;
export type ProtocolsTranslations = typeof enProtocols;
export type SettingsTranslations = typeof enSettings;

export interface Translations {
  common: CommonTranslations;
  devices: DevicesTranslations;
  errors: ErrorsTranslations;
  help: HelpTranslations;
  pages: PagesTranslations;
  protocols: ProtocolsTranslations;
  settings: SettingsTranslations;
}

declare module 'i18next' {
  interface CustomTypeOptions {
    defaultNS: 'common';
    resources: Translations;
  }
}
