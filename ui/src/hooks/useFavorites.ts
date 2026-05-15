import { useCallback, useEffect, useState } from 'react';

/**
 * useFavorites stores a Set of arbitrary string keys in localStorage so
 * the user's "favorite" markings survive reloads. Keyed by storageKey
 * so multiple feature areas (Networks, Devices, …) can keep separate
 * favorite sets without colliding.
 *
 * SSR-safe — if window is undefined the initial set is empty and writes
 * no-op. Storage is read once on mount; subsequent edits update both
 * state and the backing store.
 */
export function useFavorites(storageKey: string) {
  const [favorites, setFavorites] = useState<Set<string>>(() => {
    if (typeof window === 'undefined') return new Set();
    try {
      const raw = window.localStorage.getItem(storageKey);
      if (!raw) return new Set();
      const parsed = JSON.parse(raw) as unknown;
      if (Array.isArray(parsed)) {
        return new Set(parsed.filter((v): v is string => typeof v === 'string'));
      }
    } catch {
      // Corrupted / non-JSON value — start clean.
    }
    return new Set();
  });

  useEffect(() => {
    if (typeof window === 'undefined') return;
    try {
      window.localStorage.setItem(storageKey, JSON.stringify([...favorites]));
    } catch {
      // Quota / privacy mode — just skip persistence.
    }
  }, [storageKey, favorites]);

  const isFavorite = useCallback((key: string) => favorites.has(key), [favorites]);

  const toggleFavorite = useCallback((key: string) => {
    setFavorites((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  }, []);

  return { isFavorite, toggleFavorite };
}
