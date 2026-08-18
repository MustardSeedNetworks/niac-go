/**
 * Holds GUI help completeness in CI.
 *
 * The page header's (?) opens the drawer on `pageHelp[page.path]`, so a route
 * without an entry silently loses its help button, and an entry without a
 * route is content nothing can reach. Both are failures here rather than
 * discoveries in the UI.
 */
import { renderHook } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import '../i18n';
import { usePages } from '../pageRegistry';
import { pageHelp } from './page-help';

describe('page help — route coverage', () => {
  const { result } = renderHook(() => usePages());
  const paths = result.current.map((page) => page.path);

  it('every route has page help', () => {
    const missing = paths.filter((path) => !pageHelp[path]);
    expect(missing, `add an entry to data/page-help.ts for: ${missing.join(', ')}`).toEqual([]);
  });

  it('every page-help entry belongs to a route', () => {
    const orphaned = Object.keys(pageHelp).filter((path) => !paths.includes(path));
    expect(orphaned, `these entries match no route: ${orphaned.join(', ')}`).toEqual([]);
  });
});
