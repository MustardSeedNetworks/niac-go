import { useEffect, useMemo, useRef } from 'react';
import { useNavigate } from 'react-router-dom';

interface Shortcut {
  keys: string[];
  action: () => void;
  description: string;
}

/**
 * Keyboard shortcuts for power users.
 * Supports chord sequences (e.g., 'g' then 'd' for go-to-devices).
 * Shortcuts are disabled when focus is in an input/textarea.
 *
 * @param onShowHelp opens the help drawer (the `?` shortcut). Wired through
 *   React state from AppShell — previously this fired a `niac:show-shortcuts`
 *   CustomEvent that nothing listened for, so `?` silently did nothing.
 */
export function useKeyboardShortcuts(onShowHelp: () => void): Shortcut[] {
  const navigate = useNavigate();
  const navigateRef = useRef(navigate);
  navigateRef.current = navigate;

  // Keep the latest callback in a ref so the memoised `shortcuts` array stays
  // stable while still calling the current handler.
  const onShowHelpRef = useRef(onShowHelp);
  onShowHelpRef.current = onShowHelp;

  const pendingRef = useRef<string | null>(null);
  const timerRef = useRef<ReturnType<typeof setTimeout>>(null);

  const shortcuts: Shortcut[] = useMemo(
    () => [
      {
        keys: ['g', 'd'],
        action: () => navigateRef.current('/device-config'),
        description: 'Go to Devices',
      },
      {
        keys: ['g', 't'],
        action: () => navigateRef.current('/topology'),
        description: 'Go to Topology',
      },
      {
        keys: ['g', 'r'],
        action: () => navigateRef.current('/runtime'),
        description: 'Go to Runtime',
      },
      {
        keys: ['g', 'p'],
        action: () => navigateRef.current('/packets'),
        description: 'Go to Packets',
      },
      {
        keys: ['g', 'l'],
        action: () => navigateRef.current('/debug'),
        description: 'Go to Debug Console',
      },
      { keys: ['g', 'h'], action: () => navigateRef.current('/'), description: 'Go to Home' },
      {
        keys: ['g', 'a'],
        action: () => navigateRef.current('/traffic'),
        description: 'Go to Traffic',
      },
      { keys: ['?'], action: () => onShowHelpRef.current(), description: 'Show shortcuts' },
    ],
    [],
  );

  useEffect(() => {
    function handleKeyDown(event: KeyboardEvent): void {
      const target = event.target as HTMLElement;

      if (
        target.tagName === 'INPUT' ||
        target.tagName === 'TEXTAREA' ||
        target.tagName === 'SELECT' ||
        target.isContentEditable
      ) {
        return;
      }

      if (event.ctrlKey || event.metaKey || event.altKey) {
        return;
      }

      const key = event.key.toLowerCase();

      if (pendingRef.current) {
        const chord = pendingRef.current;
        pendingRef.current = null;
        if (timerRef.current) {
          clearTimeout(timerRef.current);
        }

        const match = shortcuts.find(
          (s) => s.keys.length === 2 && s.keys[0] === chord && s.keys[1] === key,
        );
        if (match) {
          event.preventDefault();
          match.action();
        }
      } else {
        const singleMatch = shortcuts.find((s) => s.keys.length === 1 && s.keys[0] === key);
        if (singleMatch) {
          event.preventDefault();
          singleMatch.action();
          return;
        }

        const couldBeChord = shortcuts.some((s) => s.keys.length === 2 && s.keys[0] === key);
        if (couldBeChord) {
          pendingRef.current = key;
          timerRef.current = setTimeout(() => {
            pendingRef.current = null;
          }, 500);
        }
      }
    }

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [shortcuts]);

  return shortcuts;
}
