/**
 * AppPageHeader.test.tsx -- the trail and the kicker every page opens with
 * (UI-NIAC-5, niac-go#2189; UI-NIAC-28).
 *
 * The labels used to come from a hand-kept map of 11 of the 16 routes, so the
 * five it missed fell through to the raw URL slug. They all come from the page
 * registry now, and every page, the dashboard and the 404 included, carries a
 * kicker and a trail.
 */

import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import { afterEach, describe, expect, it } from 'vitest';
import i18n from '../i18n';
import { usePages } from '../pageRegistry';
import { AppPageHeader } from './AppPageHeader';

function Header() {
  const page = usePages().find((candidate) => candidate.path === window.location.pathname);
  return <AppPageHeader kicker="Kicker" title={page?.title ?? 'Unknown'} />;
}

function renderAt(path: string) {
  window.history.replaceState({}, '', path);
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Header />
    </MemoryRouter>,
  );
}

function trail(): string[] {
  const nav = screen.getByRole('navigation', { name: i18n.t('common:accessibility.breadcrumb') });
  return Array.from(nav.querySelectorAll('li')).map((node) => node.textContent?.trim() ?? '');
}

describe('AppPageHeader trail', () => {
  afterEach(async () => {
    await i18n.changeLanguage('en');
  });

  it.each([
    ['/runtime', ['Dashboard', 'Simulation']],
    ['/new-simulation', ['Dashboard', 'New simulation']],
    ['/walk-analyzer', ['Dashboard', 'Walk analyzer']],
    ['/segments', ['Dashboard', 'Segments']],
  ])('names %s from the registry, not the URL slug', (path, expected) => {
    renderAt(path);
    expect(trail()).toEqual(expected);
  });

  it('names a nested route and the group above it', () => {
    renderAt('/library/walks');
    // "/library" routes nowhere, so it is the sidebar group's name and not a
    // link: a crumb that 404s is worse than a crumb you cannot click.
    expect(trail()).toEqual(['Dashboard', 'Library', 'Walk library']);
    expect(screen.queryByRole('link', { name: 'Library' })).toBeNull();
  });

  it('translates the trail', async () => {
    await i18n.changeLanguage('es');
    renderAt('/segments');
    expect(trail()).toEqual(['Panel', 'Segmentos']);
  });

  it('translates the group above a nested route', async () => {
    await i18n.changeLanguage('es');
    renderAt('/library/walks');
    expect(trail()[1]).toBe('Biblioteca');
  });

  it('shows a trail of one at the dashboard, which is the root', () => {
    renderAt('/');
    expect(trail()).toEqual(['Dashboard']);
    expect(screen.getByTestId('page-header-eyebrow')).toHaveTextContent('Kicker');
  });
});
