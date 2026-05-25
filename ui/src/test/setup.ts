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
