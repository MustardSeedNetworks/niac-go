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

/**
 * Nav regroup (naming + nav IA, PR 2a):
 *   - /license moves out of Overview into a bottom System group.
 *   - Tools groups config-diff, walk-validator, walk-analyzer, traffic,
 *     and automation (folded from the old single-item Alerts group).
 *   - Live View shrinks to Devices + Topology; segments moves to Library;
 *     Inspect shrinks to Packets + Debug.
 * Locks the structure so a future edit can't silently re-scatter these
 * without a deliberate test update.
 */
describe('navGroups regroup (Tools/System split)', () => {
  const { result } = renderHook(() => useNavGroups());
  const byPath = (path: string) => result.current.find((g) => g.items.some((i) => i.path === path));

  it('places /license in a group of its own at the bottom (System)', () => {
    const group = byPath('/license');
    expect(group?.items.map((i) => i.path)).toEqual(['/license']);
    expect(result.current[result.current.length - 1]).toBe(group);
  });

  it('groups config-diff, walk-validator, walk-analyzer, traffic, and automation under Tools', () => {
    const group = byPath('/config-diff');
    expect(group?.items.map((i) => i.path)).toEqual([
      '/config-diff',
      '/walk-validator',
      '/walk-analyzer',
      '/traffic',
      '/automation',
    ]);
  });

  it('no longer has a standalone single-item Alerts group', () => {
    const automationGroup = byPath('/automation');
    expect(automationGroup?.items.length).toBeGreaterThan(1);
  });

  it('keeps Live View to Devices + Topology', () => {
    const group = byPath('/devices');
    expect(group?.items.map((i) => i.path)).toEqual(['/devices', '/topology']);
  });

  it('keeps Inspect to Packets + Debug', () => {
    const group = byPath('/packets');
    expect(group?.items.map((i) => i.path)).toEqual(['/packets', '/debug']);
  });

  it('moves segments into Library alongside the device library and file browsers', () => {
    const group = byPath('/segments');
    expect(group?.items.map((i) => i.path)).toEqual([
      '/device-config',
      '/library/walks',
      '/library/pcaps',
      '/segments',
    ]);
  });
});
