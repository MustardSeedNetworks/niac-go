/**
 * ReadOnlyView — wraps a panel so a read-only-scoped token sees the
 * current state but cannot mutate it (#762).
 *
 * Renders an explanatory banner when the active scope lacks write
 * access, and disables every descendant native form control via a
 * `<fieldset disabled>`. The fieldset trick works because the HTML
 * spec propagates `disabled` to every nested input, select, textarea,
 * and button — one wrap on a panel root locks down the whole surface
 * regardless of how many sub-controls exist.
 *
 * Read-write and admin tokens see no chrome change: the wrapper is
 * `display: contents`, the banner is not rendered and the fieldset is
 * enabled, so the page below lays out exactly as it would unwrapped.
 *
 * For controls that should remain visible-but-disabled with a custom
 * tooltip, prefer the inline `useScope().canWrite` + `disabled`
 * pattern so the title can compose with other gates.
 */

import type { ReactElement, ReactNode } from 'react';
import { useScope } from '../../contexts/ScopeContext';

interface ReadOnlyViewProps {
  children: ReactNode;
  /** Optional override for the banner copy. */
  notice?: string;
}

export function ReadOnlyView({ children, notice }: ReadOnlyViewProps): ReactElement {
  const { canWrite, loading } = useScope();
  // Both scopes render the same element at the same position on purpose
  // (#1941). The scope is unknown until GET /auth/scope answers, so every
  // session starts read-only for a moment; swapping the tree shape when the
  // answer arrived remounted the whole routed page under it and threw away
  // whatever the operator had already started -- a chosen file, a half-filled
  // wizard step. Toggling `disabled` and the wrapper's class leaves the
  // subtree in place.
  return (
    <div className={canWrite ? 'contents' : 'stack-sm'}>
      {/* Fail-closed disables the page while /auth/scope is in flight, but
          saying the token disallows changes before the token has been read
          would be a claim the app cannot yet make. */}
      {canWrite || loading ? null : (
        <div
          role="status"
          className="rounded-lg border border-status-info/30 bg-status-info/5 pad-sm text-sm text-status-info"
        >
          {notice ?? 'Read-only — your token does not allow changes on this panel.'}
        </div>
      )}
      <fieldset disabled={!canWrite} className="contents">
        {children}
      </fieldset>
    </div>
  );
}
