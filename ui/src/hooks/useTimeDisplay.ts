import { useCallback, useState } from 'react';
import { safeGetItem, safeSetItem } from '../utils/storage';
import { type TimeDisplayMode, nextTimeDisplayMode } from '../utils/time-display';

const STORAGE_KEY = 'niac-time-display-mode';

function loadMode(): TimeDisplayMode {
  const stored = safeGetItem(STORAGE_KEY);
  if (stored === 'absolute' || stored === 'relative' || stored === 'delta') {
    return stored;
  }
  return 'absolute';
}

/**
 * Hook for managing time display mode with localStorage persistence.
 * Cycles through: absolute -> relative -> delta
 */
export function useTimeDisplay() {
  const [mode, setMode] = useState<TimeDisplayMode>(loadMode);

  const cycleMode = useCallback(() => {
    setMode((prev) => {
      const next = nextTimeDisplayMode(prev);
      safeSetItem(STORAGE_KEY, next);
      return next;
    });
  }, []);

  return { mode, cycleMode } as const;
}
