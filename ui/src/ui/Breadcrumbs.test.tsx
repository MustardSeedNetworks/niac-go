/**
 * Breadcrumbs.test.tsx — the trail names the page the way the rest of the
 * app does (UI-NIAC-5, niac-go#2189).
 *
 * The labels used to come from a hand-kept map of 11 of the 16 routes. The
 * five it missed fell through to the raw URL slug — "walk analyzer",
 * "new-simulation" — in English only, while the sidebar and the page header
 * showed a translated Title Case name for the same route. One route, three
 * names. They now all come from the page registry.
 */

import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { afterEach, describe, expect, it } from 'vitest';
import i18n from '../i18n';
import { Breadcrumbs } from './Breadcrumbs';

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Breadcrumbs />
    </MemoryRouter>,
  );
}

function trail(): string[] {
  return Array.from(
    screen.getByRole('navigation', { name: 'Breadcrumb' }).querySelectorAll('[data-crumb]'),
  ).map((node) => node.textContent?.trim() ?? '');
}

describe('Breadcrumbs', () => {
  afterEach(async () => {
    await i18n.changeLanguage('en');
  });

  it.each([
    ['/runtime', ['Simulation']],
    ['/new-simulation', ['New Simulation']],
    ['/walk-analyzer', ['Walk Analyzer']],
    ['/segments', ['Segments']],
  ])('names %s from the registry, not the URL slug', (path, expected) => {
    renderAt(path);
    expect(trail()).toEqual(expected);
  });

  it('names a nested route and the group above it', () => {
    renderAt('/library/walks');
    // "/library" routes nowhere, so it is the sidebar group's name and not a
    // link: a crumb that 404s is worse than a crumb you cannot click.
    expect(trail()).toEqual(['Library', 'Walks']);
    expect(screen.queryByRole('link', { name: 'Library' })).toBeNull();
  });

  it('translates the trail', async () => {
    await i18n.changeLanguage('es');
    // /segments was one of the five routes the old map missed, so its crumb
    // was the English slug in every language.
    renderAt('/segments');
    expect(trail()).toEqual(['Segmentos']);
  });

  it('translates the group above a nested route', async () => {
    await i18n.changeLanguage('es');
    renderAt('/library/walks');
    // ("Walks" is the same word in the es catalog, as it is inside
    // "Analizador de Walk" — a domain term, not a missed crumb.)
    expect(trail()[0]).toBe('Biblioteca');
  });

  it('renders nothing at the dashboard, which is the root', () => {
    renderAt('/');
    expect(screen.queryByRole('navigation', { name: 'Breadcrumb' })).toBeNull();
  });
});
