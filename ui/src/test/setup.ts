/**
 * Test Setup
 *
 * Provides shared test configuration and JSDoM polyfills for Vitest.
 * Universal polyfill baseline shared across seed/stem/niac.
 */

import '@testing-library/jest-dom';
import i18next from 'i18next';
import { afterEach, vi } from 'vitest';

// ============================================================
// Real i18n
// ============================================================
// Initialising the real i18next means a test asserting on user-visible text is
// asserting on the actual locale files. Without it, <Trans> and t() render
// nothing and a test that checks copy silently checks the absence of copy.
import '../i18n';

// ============================================================
// Unresolved i18n keys fail the test that provoked them
// ============================================================
// Loading the real locale files is only half the guarantee. i18next renders the
// KEY itself on a miss, so `settings.mode.reflector` appears in the UI looking
// like a label and every assertion still passes — seed measured this by
// corrupting all 4,290 strings in both locales and still saw 285 of 311 tests
// pass (#1669, seed#1942).
//
// Rather than assert copy string by string — a human reviews wording — every
// existing test becomes a missing-key detector: i18next reports each miss and
// the test that provoked it fails.
//
// missingKeyHandler, NOT the `missingKey` event. Only the handler is told
// whether a defaultValue was supplied, and an optional lookup with an explicit
// default is a deliberate absence, not a defect — pageRegistry.ts reads an
// optional eyebrow per page that way. Seed's first attempt used the event and
// reported eleven keys of which seven were non-problems; a gate that cries wolf
// on its first run is one people learn to ignore.
const missingI18nKeys = new Set<string>();

i18next.options.saveMissing = true;
i18next.options.missingKeyHandler = (_lngs, namespace, key, _fallback, _updateMissing, options) => {
  if (options?.defaultValue !== undefined) {
    return; // deliberate optional lookup
  }
  missingI18nKeys.add(`${namespace}:${key}`);
};

afterEach(() => {
  if (missingI18nKeys.size === 0) {
    return;
  }
  const keys = [...missingI18nKeys].sort();
  missingI18nKeys.clear(); // do not blame the next test for this one's misses
  throw new Error(
    `Unresolved i18n key(s) rendered during this test:\n  ${keys.join('\n  ')}\n` +
      'i18next renders the key itself on a miss, so this would have shipped as ' +
      'visible UI text. Add the key to the locale files, or pass an explicit ' +
      'defaultValue if the lookup is genuinely optional.',
  );
});

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

// ResizeObserver: used by xyflow, codemirror, recharts, headlessui.
// Must be a real class (not an arrow-function mockImplementation) — some
// consumers (CodeMirror's EditorView) construct it with `new`, which throws
// "is not a constructor" against an arrow function.
class MockResizeObserver implements ResizeObserver {
  observe = vi.fn();
  unobserve = vi.fn();
  disconnect = vi.fn();
}
global.ResizeObserver = MockResizeObserver;

// IntersectionObserver: used by lazy loading, infinite scroll, and
// CodeMirror's DOMObserver (@codemirror/view). Must be a real class, not an
// arrow-function mockImplementation — see the ResizeObserver comment above.
class MockIntersectionObserver implements IntersectionObserver {
  readonly root: Element | Document | null = null;
  readonly rootMargin: string = '';
  readonly scrollMargin: string = '';
  readonly thresholds: ReadonlyArray<number> = [];
  observe = vi.fn();
  unobserve = vi.fn();
  disconnect = vi.fn();
  takeRecords = vi.fn(() => []);
}
global.IntersectionObserver = MockIntersectionObserver;

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
