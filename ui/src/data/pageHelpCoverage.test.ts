import enHelp from '@locales/en/help.json';
import esHelp from '@locales/es/help.json';
import { renderHook } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import i18n from '../i18n';
import { usePages } from '../pageRegistry';
import { getPageHelp, pageHelpRoutes } from './page-help';

const leaves = (value: object, prefix = ''): string[] =>
  Object.entries(value).flatMap(([key, child]) =>
    typeof child === 'object' && child !== null
      ? leaves(child, `${prefix}${key}.`)
      : [`${prefix}${key}`],
  );

describe.each(['en', 'es'] as const)('page help — %s', (language) => {
  it('covers every route in both directions', () => {
    const { result } = renderHook(() => usePages());
    const paths = result.current.map((page) => page.path).sort((a, b) => a.localeCompare(b));
    const help = getPageHelp(i18n.getFixedT(language, 'help'));
    expect(Object.keys(help).sort((a, b) => a.localeCompare(b))).toEqual(paths);
    expect([...pageHelpRoutes].sort((a, b) => a.localeCompare(b))).toEqual(paths);
    for (const blocks of Object.values(help)) expect(blocks.length).toBeGreaterThan(0);
  });

  it('renders every short page-help key without orphans or fallback keys', () => {
    const translation = { t: i18n.getFixedT(language, 'help') };
    const spy = vi.spyOn(translation, 't');
    getPageHelp(translation.t);
    const used = new Set(spy.mock.calls.map(([key]) => String(key)));
    const catalog = language === 'en' ? enHelp : esHelp;
    expect([...used].sort((a, b) => a.localeCompare(b))).toEqual(
      leaves(catalog.pageHelp, 'pageHelp.').sort((a, b) => a.localeCompare(b)),
    );
    for (const key of used)
      expect(i18n.exists(key, { lng: language, ns: 'help', fallbackLng: false })).toBe(true);
  });
});

it('keeps English and Spanish short-help keys in parity', () => {
  expect(leaves(esHelp.pageHelp).sort((a, b) => a.localeCompare(b))).toEqual(
    leaves(enHelp.pageHelp).sort((a, b) => a.localeCompare(b)),
  );
  expect(leaves(esHelp.glossary).sort((a, b) => a.localeCompare(b))).toEqual(
    leaves(enHelp.glossary).sort((a, b) => a.localeCompare(b)),
  );
});
