/**
 * useDocumentTitle.test.tsx — the browser tab names the page (UI-NIAC-5,
 * niac-go#2189).
 *
 * document.title was set once in index.html and never again, so every route
 * was "NIAC | Mustard Seed Networks": a browser history entry, a bookmark and
 * a second tab were all indistinguishable.
 */

import { renderHook } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { afterEach, describe, expect, it } from 'vitest';
import i18n from '../i18n';
import { useDocumentTitle } from './useDocumentTitle';

function at(path: string) {
  return renderHook(() => useDocumentTitle(), {
    wrapper: ({ children }) => <MemoryRouter initialEntries={[path]}>{children}</MemoryRouter>,
  });
}

describe('useDocumentTitle', () => {
  afterEach(async () => {
    await i18n.changeLanguage('en');
  });

  it.each([
    ['/runtime', 'Simulation'],
    ['/walk-analyzer', 'Walk Analyzer'],
    ['/packets', 'Packets'],
  ])('names %s in the tab', (path, label) => {
    at(path);
    expect(document.title).toBe(`${label} | NIAC`);
  });

  it('is the product alone at the dashboard, not "Dashboard | NIAC"', () => {
    at('/');
    expect(document.title).toBe('NIAC');
  });

  it('leaves the title alone on a route the registry does not know', () => {
    at('/');
    at('/device-config/gateway-1');
    // A dynamic route's page owns its own heading; guessing a tab name from a
    // URL segment is how the breadcrumb ended up showing slugs (#2189).
    expect(document.title).toBe('NIAC');
  });

  it('follows the language', async () => {
    await i18n.changeLanguage('es');
    at('/segments');
    expect(document.title).toBe('Segmentos | NIAC');
  });
});
