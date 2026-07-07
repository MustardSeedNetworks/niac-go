/**
 * Test Setup
 *
 * Provides shared test configuration and JSDoM polyfills for Vitest.
 * Universal polyfill baseline shared across seed/stem/niac.
 */

import '@testing-library/jest-dom';
import { vi } from 'vitest';

// ============================================================
// JSDoM polyfills — common browser APIs not implemented by JSDoM
// Universal baseline shared across seed/stem/niac test setups.
// ============================================================

// matchMedia: dark-mode detection, responsive hooks, prefers-reduced-motion
Object.defineProperty(window, 'matchMedia', {
  writable: true,
  value: (query: string) => ({
    matches: false,
    media: query,
    onchange: null,
    addListener: vi.fn(), // legacy API still used by some libs
    removeListener: vi.fn(), // legacy
    addEventListener: vi.fn(),
    removeEventListener: vi.fn(),
    dispatchEvent: vi.fn(),
  }),
});

// ResizeObserver: used by xyflow, codemirror, recharts, headlessui
global.ResizeObserver = vi.fn().mockImplementation(() => ({
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn(),
})) as unknown as typeof ResizeObserver;

// IntersectionObserver: used by lazy loading, infinite scroll
global.IntersectionObserver = vi.fn().mockImplementation(() => ({
  observe: vi.fn(),
  unobserve: vi.fn(),
  disconnect: vi.fn(),
  takeRecords: vi.fn(() => []),
  root: null,
  rootMargin: '',
  thresholds: [],
})) as unknown as typeof IntersectionObserver;

// localStorage: Node 22+'s experimental global `localStorage` shadows
// jsdom's window.localStorage and throws on access without a
// --localstorage-file flag, which breaks zustand `persist` stores (e.g.
// ui-store) in tests. Provide a minimal in-memory implementation so
// persisted stores work the same as they do in the browser.
class MemoryStorage implements Storage {
  #store = new Map<string, string>();
  get length() {
    return this.#store.size;
  }
  clear(): void {
    this.#store.clear();
  }
  getItem(key: string): string | null {
    return this.#store.has(key) ? (this.#store.get(key) ?? null) : null;
  }
  key(index: number): string | null {
    return Array.from(this.#store.keys())[index] ?? null;
  }
  removeItem(key: string): void {
    this.#store.delete(key);
  }
  setItem(key: string, value: string): void {
    this.#store.set(key, value);
  }
}

const memoryStorage = new MemoryStorage();
Object.defineProperty(window, 'localStorage', {
  configurable: true,
  value: memoryStorage,
});
Object.defineProperty(globalThis, 'localStorage', {
  configurable: true,
  value: memoryStorage,
});
