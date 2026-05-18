/**
 * Theme Management Hook
 *
 * Manages application theme with support for light, dark, and system modes.
 *
 * Features:
 * - Light and dark theme modes
 * - Automatic system theme detection via prefers-color-scheme
 * - Persistent theme storage in localStorage (key: 'niac-theme')
 * - Automatic system theme change detection while in 'system' mode
 * - Theme toggling functionality
 *
 * The theme is applied by adding/removing the 'dark' class on the document
 * root element. Tailwind CSS / our index.css uses that class to switch
 * between :root (light) and .dark (dark) custom properties.
 *
 * Default value when no stored preference exists is 'dark' — this matches
 * the seed/stem behaviour and is the intended out-of-the-box appearance.
 *
 * Usage:
 * ```typescript
 * const { theme, isDark, toggleTheme, setTheme } = useTheme();
 *
 * <button onClick={toggleTheme}>Toggle Theme</button>
 * <button onClick={() => setTheme('system')}>Use System Theme</button>
 * ```
 */

import { useCallback, useEffect, useState } from 'react';

/** Theme mode options */
export type Theme = 'light' | 'dark' | 'system';

/** localStorage key for theme persistence */
const STORAGE_KEY = 'niac-theme';

/**
 * Detects the system's preferred color scheme.
 *
 * @returns 'dark' if system prefers dark mode, 'light' otherwise.
 *          Defaults to 'dark' if detection is unavailable.
 */
function getSystemTheme(): 'light' | 'dark' {
  if (typeof window !== 'undefined' && window.matchMedia) {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }
  return 'dark';
}

/**
 * Applies the theme to the document root element.
 * Resolves 'system' theme to actual light/dark preference.
 *
 * @param theme - Theme to apply ('light', 'dark', or 'system')
 */
function applyTheme(theme: Theme): void {
  if (typeof document === 'undefined') {
    return;
  }
  const root: HTMLElement = document.documentElement;
  const effectiveTheme: 'light' | 'dark' = theme === 'system' ? getSystemTheme() : theme;

  if (effectiveTheme === 'dark') {
    root.classList.add('dark');
  } else {
    root.classList.remove('dark');
  }
}

/**
 * Read the stored theme from localStorage. Defaults to 'dark' when no
 * preference is stored, ensuring dark mode is the out-of-the-box default.
 */
function readStoredTheme(): Theme {
  if (typeof window === 'undefined') {
    return 'dark';
  }
  try {
    const stored = window.localStorage.getItem(STORAGE_KEY) as Theme | null;
    if (stored === 'light' || stored === 'dark' || stored === 'system') {
      return stored;
    }
  } catch {
    // localStorage may be unavailable (e.g. private browsing)
  }
  return 'dark';
}

/**
 * Custom hook for managing application theme.
 *
 * Handles theme persistence, system theme detection, and automatic updates
 * when system theme changes.
 *
 * @returns Theme state and control functions
 */
export function useTheme(): {
  theme: Theme;
  effectiveTheme: 'light' | 'dark';
  setTheme: (newTheme: Theme) => void;
  toggleTheme: () => void;
  isDark: boolean;
} {
  const [theme, setThemeState] = useState<Theme>(readStoredTheme);

  const [effectiveTheme, setEffectiveTheme] = useState<'light' | 'dark'>(() =>
    theme === 'system' ? getSystemTheme() : theme,
  );

  const setTheme = useCallback((newTheme: Theme): void => {
    setThemeState(newTheme);
    try {
      window.localStorage.setItem(STORAGE_KEY, newTheme);
    } catch {
      // Ignore storage failures (private browsing, quota, etc.)
    }
    applyTheme(newTheme);
    setEffectiveTheme(newTheme === 'system' ? getSystemTheme() : newTheme);
  }, []);

  const toggleTheme = useCallback((): void => {
    const newTheme: Theme = effectiveTheme === 'dark' ? 'light' : 'dark';
    setTheme(newTheme);
  }, [effectiveTheme, setTheme]);

  // Apply theme on mount and listen for system theme changes
  useEffect(() => {
    applyTheme(theme);

    if (theme === 'system') {
      const mediaQuery: MediaQueryList = window.matchMedia('(prefers-color-scheme: dark)');
      const handler = (e: MediaQueryListEvent): void => {
        setEffectiveTheme(e.matches ? 'dark' : 'light');
        applyTheme('system');
      };
      mediaQuery.addEventListener('change', handler);
      return (): void => mediaQuery.removeEventListener('change', handler);
    }
    return undefined;
  }, [theme]);

  return {
    theme,
    effectiveTheme,
    setTheme,
    toggleTheme,
    isDark: effectiveTheme === 'dark',
  };
}

/**
 * Apply the initial theme synchronously, before React mounts. This avoids
 * a brief flash of light mode at boot when the user has stored 'dark'.
 *
 * Call from main.tsx before `createRoot(...).render(...)`.
 */
export function initThemeFromStorage(): void {
  applyTheme(readStoredTheme());
}
