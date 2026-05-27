/**
 * useLocale — single source for the active i18next language code.
 *
 * Returns a BCP-47 tag suitable for `Intl.*` constructors. i18next
 * gives us the short code ('en' / 'es'); we normalise to the regional
 * tags 'en-US' / 'es-ES' so number/date formatters get a sensible
 * default (decimal separators, weekday names) without each call site
 * having to pick.
 *
 * Use this whenever you'd otherwise call `Intl.NumberFormat(undefined,
 * …)` or hardcode `'en-US'`. The hook re-renders consumers when the
 * user switches language, so date strings flip automatically.
 *
 * Lower-level (non-React) code that needs the same value can call
 * `i18n.language` directly — but prefer the hook in components so
 * language changes trigger re-renders.
 */

import { useTranslation } from 'react-i18next';

const REGION_BY_LANGUAGE: Record<string, string> = {
  en: 'en-US',
  es: 'es-ES',
};

export function useLocale(): string {
  const { i18n } = useTranslation();
  return REGION_BY_LANGUAGE[i18n.language] ?? i18n.language ?? 'en-US';
}
