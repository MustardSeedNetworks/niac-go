/**
 * Tests for the localStorage wrappers.
 *
 * Both exist purely to swallow the throwing cases — a Safari private-mode
 * setItem, or a getItem behind a blocked-storage policy — so the throwing
 * paths are the point of the module, not an edge case.
 */

import { afterEach, describe, expect, it, vi } from 'vitest';
import { safeGetItem, safeSetItem } from './storage';

afterEach(() => {
  vi.restoreAllMocks();
  localStorage.clear();
});

describe('safeGetItem', () => {
  it('returns the stored value', () => {
    localStorage.setItem('k', 'v');
    expect(safeGetItem('k')).toBe('v');
  });

  it('returns null for a missing key', () => {
    expect(safeGetItem('absent')).toBeNull();
  });

  it('returns null instead of propagating a thrown access', () => {
    localStorage.setItem('k', 'v');
    // Spy on the instance: src/test/setup.ts swaps in a MemoryStorage object,
    // so Storage.prototype is not on this call path.
    vi.spyOn(localStorage, 'getItem').mockImplementation(() => {
      throw new Error('access denied');
    });

    expect(safeGetItem('k')).toBeNull();
  });
});

describe('safeSetItem', () => {
  it('stores the value and reports success', () => {
    expect(safeSetItem('k', 'v')).toBe(true);
    expect(localStorage.getItem('k')).toBe('v');
  });

  it('reports failure when the quota is exceeded', () => {
    vi.spyOn(localStorage, 'setItem').mockImplementation(() => {
      throw new DOMException('quota', 'QuotaExceededError');
    });

    expect(safeSetItem('k', 'v')).toBe(false);
  });

  it('reports failure for any other thrown error', () => {
    vi.spyOn(localStorage, 'setItem').mockImplementation(() => {
      throw new Error('storage disabled');
    });

    expect(safeSetItem('k', 'v')).toBe(false);
  });
});
