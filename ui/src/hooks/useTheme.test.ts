import { act, renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { resetThemeStore, useTheme } from './useTheme';

type Listener = () => void;

function stubSystem(dark: boolean) {
  const state = { dark };
  const listeners = new Set<Listener>();
  vi.stubGlobal('matchMedia', (query: string) => ({
    get matches() {
      return state.dark;
    },
    media: query,
    addEventListener: (_: string, listener: Listener) => listeners.add(listener),
    removeEventListener: (_: string, listener: Listener) => listeners.delete(listener),
  }));
  window.matchMedia = globalThis.matchMedia;
  return {
    flip(next: boolean) {
      state.dark = next;
      for (const listener of listeners) listener();
    },
  };
}

describe('useTheme', () => {
  beforeEach(() => {
    window.localStorage.clear();
    document.documentElement.classList.remove('dark');
  });

  afterEach(() => {
    vi.unstubAllGlobals();
    window.localStorage.clear();
  });

  it('follows the OS when nothing is stored', () => {
    stubSystem(true);
    resetThemeStore();
    const { result } = renderHook(() => useTheme());
    expect(result.current.theme).toBe('system');
    expect(result.current.isDark).toBe(true);
    expect(document.documentElement.classList.contains('dark')).toBe(true);
  });

  it('re-applies when the OS changes while the preference is system', () => {
    const system = stubSystem(false);
    resetThemeStore();
    const { result } = renderHook(() => useTheme());
    expect(result.current.isDark).toBe(false);
    act(() => system.flip(true));
    expect(result.current.isDark).toBe(true);
  });

  it('persists an explicit choice and stops following the OS', () => {
    const system = stubSystem(false);
    resetThemeStore();
    const { result } = renderHook(() => useTheme());
    act(() => result.current.setTheme('dark'));
    expect(window.localStorage.getItem('niac-theme')).toBe('dark');
    act(() => system.flip(false));
    expect(result.current.isDark).toBe(true);
  });

  it('is one store: a second caller sees a change made by the first', () => {
    stubSystem(false);
    resetThemeStore();
    const settings = renderHook(() => useTheme());
    const rail = renderHook(() => useTheme());
    act(() => rail.result.current.toggleTheme());
    expect(settings.result.current.isDark).toBe(true);
    expect(settings.result.current.theme).toBe('dark');
  });
});
