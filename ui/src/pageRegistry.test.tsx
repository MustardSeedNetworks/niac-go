import { renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import i18n from './i18n';
import { usePages } from './pageRegistry';

const unresolvedInterpolation = /\{\{[^}]+}}/;

describe('page registry translations', () => {
  afterEach(async () => {
    await i18n.changeLanguage('en');
  });

  it.each(['en', 'es'])('resolves page metadata interpolation in %s', async (language) => {
    await i18n.changeLanguage(language);
    const { result, unmount } = renderHook(() => usePages());

    for (const page of result.current) {
      expect(page.label, `${page.path} label`).not.toMatch(unresolvedInterpolation);
      expect(page.title, `${page.path} title`).not.toMatch(unresolvedInterpolation);
      expect(page.description, `${page.path} description`).not.toMatch(unresolvedInterpolation);
    }
    unmount();
  });
});
