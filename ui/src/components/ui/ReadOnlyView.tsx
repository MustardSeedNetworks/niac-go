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
 * Read-write and admin tokens see no chrome change: children render
 * directly, no banner, no fieldset.
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
  const { canWrite } = useScope();
  if (canWrite) {
    return <>{children}</>;
  }
  return (
    <div className="stack-sm">
      <div
        role="status"
        className="rounded-lg border border-status-info/30 bg-status-info/5 pad-sm text-sm text-status-info"
      >
        {notice ?? 'Read-only — your token does not allow changes on this panel.'}
      </div>
      <fieldset disabled className="contents">
        {children}
      </fieldset>
    </div>
  );
}
