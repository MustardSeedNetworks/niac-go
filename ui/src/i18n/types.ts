/**
 * i18n TypeScript Augmentation — NIAC
 *
 * Declaration-merges the loaded resources into the 'i18next' module so
 * the `t()` function gets compile-time autocomplete for keys and
 * namespaces. English is the source of truth; types are inferred from
 * `@locales/en/*.json`.
 *
 * Mirrors seed/stem so cross-product convention stays uniform — see
 * msn-docs-internal/05-Engineering/I18N_CONVENTIONS.md.
 */

import type common from '@locales/en/common.json';
import type devices from '@locales/en/devices.json';
import type errors from '@locales/en/errors.json';
import type help from '@locales/en/help.json';
import type pages from '@locales/en/pages.json';
import type protocols from '@locales/en/protocols.json';
import type settings from '@locales/en/settings.json';
import type { defaultNs, namespaces } from './index';

declare module 'i18next' {
  interface CustomTypeOptions {
    defaultNS: typeof defaultNs;
    resources: {
      common: typeof common;
      devices: typeof devices;
      errors: typeof errors;
      help: typeof help;
      pages: typeof pages;
      protocols: typeof protocols;
      settings: typeof settings;
    };
  }
}

export type { LanguageCode, Namespace } from './index';
export type Namespaces = typeof namespaces;
