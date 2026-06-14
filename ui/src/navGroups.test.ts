/**
 * navGroups <-> pageRegistry parity guard.
 *
 * Enforces two architecture invariants:
 *   1. Every routable page defined in pageRegistry appears in the sidebar.
 *   2. Every sidebar entry points at a route that exists in pageRegistry.
 *
 * This mirrors the same guard in seed (ui/src/navGroups.test.ts) so both
 * products enforce the same nav-drift invariant.
 *
 * Implementation note: both useNavGroups() and usePages() call
 * useTranslation('pages') internally. Importing '../i18n' initialises
 * i18next synchronously (see useLocale.test.ts for the same pattern),
 * so renderHook works without a wrapper.
 */

import { renderHook } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import './i18n'; // initialise i18next before hooks run
import { useNavGroups } from './navGroups';
import { usePages } from './pageRegistry';

describe('navGroups <-> pageRegistry parity', () => {
  const { result: navResult } = renderHook(() => useNavGroups());
  const { result: pagesResult } = renderHook(() => usePages());

  const navPaths = new Set(navResult.current.flatMap((g) => g.items.map((i) => i.path)));
  const routePaths = new Set(pagesResult.current.map((p) => p.path));

  it('exposes every routable page in the sidebar', () => {
    const missing = pagesResult.current.map((p) => p.path).filter((p) => !navPaths.has(p));
    expect(missing, `pages missing from navGroups: ${missing.join(', ')}`).toEqual([]);
  });

  it('has no sidebar entries pointing at a non-existent route', () => {
    const orphaned = [...navPaths].filter((p) => !routePaths.has(p));
    expect(orphaned, `navGroups entries without a page: ${orphaned.join(', ')}`).toEqual([]);
  });
});
