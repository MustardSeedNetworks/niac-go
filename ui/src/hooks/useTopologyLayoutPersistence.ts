import { useCallback, useEffect, useMemo, useRef } from 'react';
import {
  clearSavedPositions,
  readSavedPositions,
  type SavedPositions,
  writeSavedPositions,
} from '../pages/topology/persistence';

// Rapid node drags (or a multi-select move) can fire many position
// updates in quick succession; debouncing the localStorage write
// avoids a write-per-frame without changing user-visible behavior —
// the last position always wins once dragging settles.
const SAVE_DEBOUNCE_MS = 300;

export interface TopologyLayoutPersistence {
  /** Reads the currently-saved positions. SSR-safe, never throws. */
  loadPositions: () => SavedPositions;
  /** Debounced write — safe to call on every drag-stop event. */
  savePositions: (positions: SavedPositions) => void;
  /** Cancels any pending debounced write and clears saved positions. */
  resetPositions: () => void;
}

/**
 * useTopologyLayoutPersistence wraps the localStorage-backed topology
 * position helpers (../pages/topology/persistence.ts) with a debounced
 * save so drag-heavy sessions don't hit localStorage on every event.
 * Browser-local only — there is no backend route or server sync; a
 * different browser or profile starts from the default layout.
 */
export function useTopologyLayoutPersistence(): TopologyLayoutPersistence {
  const timerRef = useRef<number | null>(null);

  useEffect(
    () => () => {
      if (timerRef.current !== null) {
        window.clearTimeout(timerRef.current);
      }
    },
    [],
  );

  const loadPositions = useCallback((): SavedPositions => readSavedPositions(), []);

  const savePositions = useCallback((positions: SavedPositions) => {
    if (timerRef.current !== null) {
      window.clearTimeout(timerRef.current);
    }
    timerRef.current = window.setTimeout(() => {
      writeSavedPositions(positions);
      timerRef.current = null;
    }, SAVE_DEBOUNCE_MS);
  }, []);

  const resetPositions = useCallback(() => {
    if (timerRef.current !== null) {
      window.clearTimeout(timerRef.current);
      timerRef.current = null;
    }
    clearSavedPositions();
  }, []);

  // Memoized so consumers can safely depend on the returned object
  // (e.g. in a useCallback dep array) without it changing every render.
  return useMemo(
    () => ({ loadPositions, savePositions, resetPositions }),
    [loadPositions, savePositions, resetPositions],
  );
}
