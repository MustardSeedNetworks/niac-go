/**
 * RequireScope — render children only if the current token meets a
 * minimum scope. Hide-variant gating, parallel to <RequireFeature>
 * for license tier (#762).
 *
 * Use this for entire pages, drawer sections, nav items, or any UI
 * surface that should simply not exist below the threshold:
 *
 *   <RequireScope min="admin">
 *     <ConfigImportPanel />
 *   </RequireScope>
 *
 * For controls that should remain visible but disabled with a
 * tooltip, read `useScope().canWrite` / `isAdmin` directly and apply
 * `disabled` + a `title` attribute.
 *
 * Fail-closed: while the initial scope fetch is in flight `canWrite`
 * and `isAdmin` are false so children render the `fallback` (default
 * null), matching <RequireFeature>'s behavior.
 */

import type { ReactElement, ReactNode } from 'react';
import { type Scope, useScope } from '../../contexts/ScopeContext';

interface RequireScopeProps {
  /** Minimum scope required for children to render. */
  min: Scope;
  children: ReactNode;
  /** Rendered when the active scope ranks below `min` (or unknown). */
  fallback?: ReactNode;
}

const SCOPE_RANK: Record<Scope, number> = {
  'read-only': 1,
  'read-write': 2,
  admin: 3,
};

export function RequireScope({ min, children, fallback = null }: RequireScopeProps): ReactElement {
  const { scope } = useScope();
  if (scope === null || SCOPE_RANK[scope] < SCOPE_RANK[min]) {
    return <>{fallback}</>;
  }
  return <>{children}</>;
}
