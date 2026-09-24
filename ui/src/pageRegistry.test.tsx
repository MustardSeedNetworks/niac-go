import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
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

/**
 * CI's phone-width job visits a hand-listed set of routes: the reusable
 * workflow cannot read this registry. A page added here but not there would
 * ship without ever being checked at 390px, with the job still green.
 */
describe('pageRegistry <-> phone-width routes', () => {
  it('checks every registered page at phone width', () => {
    const ci = readFileSync(resolve(import.meta.dirname, '../../.github/workflows/ci.yml'), 'utf8');
    const job = ci.slice(ci.indexOf('\n  phone-width:\n'));
    const routes = /^ {6}routes: '(.+)'$/m.exec(job)?.[1];
    expect(routes, 'phone-width job has no routes input').toBeDefined();

    const byPath = (a: string, b: string) => a.localeCompare(b);
    const { result: pages, unmount } = renderHook(() => usePages());
    expect((JSON.parse(routes ?? '[]') as string[]).sort(byPath)).toEqual(
      pages.current.map((page) => page.path).sort(byPath),
    );
    unmount();
  });
});
