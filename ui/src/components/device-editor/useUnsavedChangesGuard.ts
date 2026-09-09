import { useCallback, useState } from 'react';
import { createPath, useBeforeUnload, useBlocker, useNavigate } from 'react-router';

export interface UnsavedChangesGuard {
  pendingPath: string | null;
  pending: boolean;
  requestAction: (action: () => void) => void;
  requestNavigate: (path: string) => void;
  confirmNavigate: () => void;
  cancelNavigate: () => void;
}

export const useUnsavedChangesGuard = (isDirty: boolean): UnsavedChangesGuard => {
  const navigate = useNavigate();
  const blocker = useBlocker(
    ({ currentLocation, nextLocation }) =>
      isDirty &&
      (currentLocation.pathname !== nextLocation.pathname ||
        currentLocation.search !== nextLocation.search),
  );
  const [pendingAction, setPendingAction] = useState<{ run: () => void } | null>(null);
  useBeforeUnload(
    useCallback(
      (event) => {
        if (!isDirty) return;
        event.preventDefault();
        event.returnValue = '';
      },
      [isDirty],
    ),
  );

  const confirmNavigate = () => {
    setPendingAction(null);
    if (blocker.state === 'blocked') blocker.proceed();
    else pendingAction?.run();
  };
  const cancelNavigate = () => {
    setPendingAction(null);
    if (blocker.state === 'blocked') blocker.reset();
  };
  return {
    pendingPath: blocker.state === 'blocked' ? createPath(blocker.location) : null,
    pending: blocker.state === 'blocked' || pendingAction !== null,
    requestAction: (run) => {
      if (isDirty) setPendingAction({ run });
      else run();
    },
    requestNavigate: (path) => {
      void navigate(path);
    },
    confirmNavigate,
    cancelNavigate,
  };
};
