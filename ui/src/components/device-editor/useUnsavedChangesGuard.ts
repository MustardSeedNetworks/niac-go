import { useCallback, useEffect, useRef, useState } from 'react';

export interface UnsavedChangesGuard {
  /** Path awaiting confirmation, or null when no navigation is pending. */
  pendingPath: string | null;
  pending: boolean;
  requestAction: (action: () => void) => void;
  /** Navigate now if there are no unsaved changes; otherwise queue a confirmation. */
  requestNavigate: (path: string) => void;
  /** User confirmed leaving — discard the guard and navigate to the pending path. */
  confirmNavigate: () => void;
  /** User cancelled — stay on the page, no navigation happens. */
  cancelNavigate: () => void;
}

/**
 * Guards navigation away from a dirty form.
 *
 * Two escape hatches are covered:
 * - Same-tab navigation: a document-level capture-phase click listener
 *   intercepts internal links and marked navigation buttons while dirty,
 *   so clicking away from the
 *   editor mid-edit surfaces a confirmation instead of silently discarding
 *   the in-progress changes. `requestNavigate` covers the editor's own
 *   "Back" button, which isn't an anchor.
 * - Cross-document navigation: tab close, refresh, and address-bar
 *   navigation are guarded by the standard `beforeunload` prompt, which the
 *   browser (not this code) renders.
 *
 * Deliberately does not depend on `useBlocker` — the app boots a plain
 * `BrowserRouter` (see ui/src/main.tsx), and `useBlocker` only works under
 * a data router (`createBrowserRouter`). Converting the whole app to a data
 * router is out of scope for this fix.
 */
export const useUnsavedChangesGuard = (
  isDirty: boolean,
  navigate: (path: string) => void,
): UnsavedChangesGuard => {
  const isDirtyRef = useRef(isDirty);
  isDirtyRef.current = isDirty;
  const [pendingPath, setPendingPath] = useState<string | null>(null);
  const [pendingAction, setPendingAction] = useState<{ run: () => void } | null>(null);

  const requestAction = useCallback((run: () => void) => {
    if (isDirtyRef.current) setPendingAction({ run });
    else run();
  }, []);

  useEffect(() => {
    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      if (!isDirtyRef.current) {
        return;
      }
      event.preventDefault();
      // Chrome requires returnValue to be set to show its confirmation.
      event.returnValue = '';
    };
    window.addEventListener('beforeunload', handleBeforeUnload);
    return () => window.removeEventListener('beforeunload', handleBeforeUnload);
  }, []);

  useEffect(() => {
    const handleClick = (event: MouseEvent) => {
      if (!isDirtyRef.current || event.defaultPrevented || event.button !== 0) {
        return;
      }
      if (event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) {
        return;
      }
      const target = event.target as HTMLElement | null;
      const anchor = target?.closest?.('a[href], button[data-navigation-path]');
      if (!anchor || anchor.hasAttribute('target') || anchor.hasAttribute('download')) {
        return;
      }
      const href = anchor.getAttribute('href') ?? anchor.getAttribute('data-navigation-path');
      if (!href || href.startsWith('#')) {
        return;
      }
      let url: URL;
      try {
        url = new URL(href, window.location.origin);
      } catch {
        return;
      }
      if (url.origin !== window.location.origin || url.pathname === window.location.pathname) {
        return;
      }
      event.preventDefault();
      event.stopPropagation();
      setPendingPath(`${url.pathname}${url.search}${url.hash}`);
    };
    document.addEventListener('click', handleClick, true);
    return () => document.removeEventListener('click', handleClick, true);
  }, []);

  const requestNavigate = useCallback(
    (path: string) => {
      if (isDirtyRef.current) {
        setPendingPath(path);
      } else {
        navigate(path);
      }
    },
    [navigate],
  );

  const confirmNavigate = useCallback(() => {
    if (pendingAction) {
      setPendingAction(null);
      pendingAction.run();
      return;
    }
    if (pendingPath) {
      const target = pendingPath;
      setPendingPath(null);
      navigate(target);
    }
  }, [pendingPath, pendingAction, navigate]);

  const cancelNavigate = useCallback(() => {
    setPendingPath(null);
    setPendingAction(null);
  }, []);

  return {
    pendingPath,
    pending: pendingPath !== null || pendingAction !== null,
    requestAction,
    requestNavigate,
    confirmNavigate,
    cancelNavigate,
  };
};
