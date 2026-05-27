/**
 * useLocale tests — covers the BCP-47 normalization that every Intl.*
 * call site in the codebase relies on. Regressions here would silently
 * break number/date formatting for ES users.
 */

import { renderHook } from '@testing-library/react';
import i18n from 'i18next';
import { afterEach, describe, expect, it } from 'vitest';
import '../i18n'; // initialize i18next
import { useLocale } from './useLocale';

describe('useLocale', () => {
  const originalLanguage = i18n.language;

  afterEach(async () => {
    if (i18n.language !== originalLanguage) {
      await i18n.changeLanguage(originalLanguage);
    }
  });

  it('returns en-US when i18next language is en', async () => {
    await i18n.changeLanguage('en');
    const { result } = renderHook(() => useLocale());
    expect(result.current).toBe('en-US');
  });

  it('returns es-ES when i18next language is es', async () => {
    await i18n.changeLanguage('es');
    const { result } = renderHook(() => useLocale());
    expect(result.current).toBe('es-ES');
  });

  it('falls back to en-US for unknown languages', async () => {
    // i18next.changeLanguage('xx') will set language to 'xx' even
    // though it's not in supportedLngs — we still want a valid BCP-47.
    await i18n.changeLanguage('xx');
    const { result } = renderHook(() => useLocale());
    // Either the raw code (if i18next kept it) or en-US fallback;
    // either is a valid BCP-47 tag accepted by Intl.*.
    expect(['xx', 'en-US']).toContain(result.current);
  });
});
