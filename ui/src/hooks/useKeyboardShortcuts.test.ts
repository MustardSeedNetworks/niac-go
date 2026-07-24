/**
 * useKeyboardShortcuts tests — guard the `?` shortcut, which previously
 * fired a `niac:show-shortcuts` CustomEvent that nothing listened for, so
 * pressing `?` silently did nothing. These pin that `?` now invokes the
 * injected onShowHelp callback, and that the chord shortcuts still navigate.
 */

import { renderHook } from '@testing-library/react';
import { createElement } from 'react';
import { MemoryRouter } from 'react-router';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { useKeyboardShortcuts } from './useKeyboardShortcuts';

const mockNavigate = vi.fn();
vi.mock('react-router', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router')>();
  return { ...actual, useNavigate: () => mockNavigate };
});

const wrapper = ({ children }: { children: React.ReactNode }): React.ReactElement =>
  createElement(MemoryRouter, null, children);

function press(key: string): void {
  window.dispatchEvent(new KeyboardEvent('keydown', { key, bubbles: true }));
}

afterEach(() => {
  vi.clearAllMocks();
});

describe('useKeyboardShortcuts', () => {
  it('invokes onShowHelp when "?" is pressed (was a dead CustomEvent)', () => {
    const onShowHelp = vi.fn();
    renderHook(() => useKeyboardShortcuts(onShowHelp), { wrapper });

    press('?');

    expect(onShowHelp).toHaveBeenCalledTimes(1);
  });

  it('always calls the latest onShowHelp (ref-stable, no stale closure)', () => {
    const first = vi.fn();
    const second = vi.fn();
    const { rerender } = renderHook(({ cb }) => useKeyboardShortcuts(cb), {
      wrapper,
      initialProps: { cb: first },
    });

    rerender({ cb: second });
    press('?');

    expect(first).not.toHaveBeenCalled();
    expect(second).toHaveBeenCalledTimes(1);
  });

  it('navigates the "g a" chord to the canonical /traffic route', () => {
    renderHook(() => useKeyboardShortcuts(vi.fn()), { wrapper });

    press('g');
    press('a');

    expect(mockNavigate).toHaveBeenCalledWith('/traffic');
  });
});
