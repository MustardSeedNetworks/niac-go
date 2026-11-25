/**
 * Theme store.
 *
 * One module-level store shared by every caller, so the Settings control and
 * the rail-footer toggle can never disagree. The preference is 'light',
 * 'dark' or 'system'; with nothing stored it is 'system', so the app follows
 * the operating system until the operator picks a side. The preference is
 * persisted in localStorage (key 'niac-theme'), and while it is 'system' an
 * OS change re-applies immediately.
 *
 * The theme is applied by adding/removing the 'dark' class on the document
 * root; index.css switches between :root (light) and .dark properties.
 */

import { useCallback, useSyncExternalStore } from 'react';

/** Theme preference. */
export type Theme = 'light' | 'dark' | 'system';

type EffectiveTheme = 'light' | 'dark';

const STORAGE_KEY = 'niac-theme';
const DARK_QUERY = '(prefers-color-scheme: dark)';

function systemTheme(): EffectiveTheme {
  if (typeof window !== 'undefined' && window.matchMedia) {
    return window.matchMedia(DARK_QUERY).matches ? 'dark' : 'light';
  }
  return 'light';
}

function readStoredTheme(): Theme {
  try {
    const stored = window.localStorage.getItem(STORAGE_KEY);
    if (stored === 'light' || stored === 'dark' || stored === 'system') {
      return stored;
    }
  } catch {
    // localStorage may be unavailable (private browsing)
  }
  return 'system';
}

function resolve(theme: Theme): EffectiveTheme {
  return theme === 'system' ? systemTheme() : theme;
}

function applyToDocument(effective: EffectiveTheme): void {
  if (typeof document !== 'undefined') {
    document.documentElement.classList.toggle('dark', effective === 'dark');
  }
}

interface ThemeSnapshot {
  theme: Theme;
  effectiveTheme: EffectiveTheme;
}

let snapshot: ThemeSnapshot | undefined;
const listeners = new Set<() => void>();

function current(): ThemeSnapshot {
  if (snapshot === undefined) {
    const theme = readStoredTheme();
    snapshot = { theme, effectiveTheme: resolve(theme) };
  }
  return snapshot;
}

function publish(theme: Theme): void {
  const effectiveTheme = resolve(theme);
  snapshot = { theme, effectiveTheme };
  applyToDocument(effectiveTheme);
  for (const listener of listeners) {
    listener();
  }
}

function subscribe(listener: () => void): () => void {
  listeners.add(listener);
  const query = window.matchMedia(DARK_QUERY);
  const onSystemChange = (): void => {
    if (current().theme === 'system') {
      publish('system');
    }
  };
  query.addEventListener('change', onSystemChange);
  return (): void => {
    listeners.delete(listener);
    query.removeEventListener('change', onSystemChange);
  };
}

function setTheme(theme: Theme): void {
  try {
    window.localStorage.setItem(STORAGE_KEY, theme);
  } catch {
    // Ignore storage failures (private browsing, quota)
  }
  publish(theme);
}

/** Test seam: forget the cached preference so the next read hits storage. */
export function resetThemeStore(): void {
  snapshot = undefined;
  applyToDocument(current().effectiveTheme);
  for (const listener of listeners) {
    listener();
  }
}

export function useTheme(): {
  theme: Theme;
  effectiveTheme: EffectiveTheme;
  setTheme: (newTheme: Theme) => void;
  toggleTheme: () => void;
  isDark: boolean;
} {
  const { theme, effectiveTheme } = useSyncExternalStore(subscribe, current, current);
  const toggleTheme = useCallback((): void => {
    setTheme(current().effectiveTheme === 'dark' ? 'light' : 'dark');
  }, []);
  return { theme, effectiveTheme, setTheme, toggleTheme, isDark: effectiveTheme === 'dark' };
}

/**
 * Apply the stored (or system) theme synchronously, before React mounts, so
 * there is no flash of the wrong theme at boot. Call from main.tsx before
 * `createRoot(...).render(...)`.
 */
export function initThemeFromStorage(): void {
  applyToDocument(current().effectiveTheme);
}
