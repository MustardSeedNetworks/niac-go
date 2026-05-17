import { DEFAULT_LAYOUT_MODE, LAYOUT_MODES, type LayoutMode } from './layout';

/**
 * localStorage-backed persistence helpers for the topology page.
 *
 * Two pieces of state survive page reload:
 *  1. User-dragged node positions, keyed by device name
 *  2. The chosen layout mode (hierarchical / concentric / grid)
 *
 * All helpers are SSR-safe — they no-op when `window` is undefined.
 * Lives in its own file so TopologyPage.tsx stays under the
 * 800-line file-size red-flag threshold.
 */

export const TOPOLOGY_POSITIONS_KEY = 'niac.topology.positions';
export const TOPOLOGY_LAYOUT_MODE_KEY = 'niac.topology.layoutMode';

export type SavedPositions = Record<string, { x: number; y: number }>;

export function readSavedPositions(): SavedPositions {
  if (typeof window === 'undefined') return {};
  try {
    const raw = window.localStorage.getItem(TOPOLOGY_POSITIONS_KEY);
    if (!raw) return {};
    return JSON.parse(raw) as SavedPositions;
  } catch {
    return {};
  }
}

export function writeSavedPositions(positions: SavedPositions): void {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(TOPOLOGY_POSITIONS_KEY, JSON.stringify(positions));
  } catch {
    // Quota / privacy mode — silently skip; positions will reset on reload.
  }
}

export function clearSavedPositions(): void {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.removeItem(TOPOLOGY_POSITIONS_KEY);
  } catch {
    // No-op on quota / privacy mode.
  }
}

export function readSavedLayoutMode(): LayoutMode {
  if (typeof window === 'undefined') return DEFAULT_LAYOUT_MODE;
  try {
    const raw = window.localStorage.getItem(TOPOLOGY_LAYOUT_MODE_KEY);
    if (LAYOUT_MODES.some((m) => m.mode === raw)) {
      return raw as LayoutMode;
    }
  } catch {
    // SSR / quota — fall through.
  }
  return DEFAULT_LAYOUT_MODE;
}

export function writeSavedLayoutMode(mode: LayoutMode): void {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(TOPOLOGY_LAYOUT_MODE_KEY, mode);
  } catch {
    // No-op on quota / privacy mode.
  }
}
