/**
 * useTopologyLayoutPersistence tests — pin the browser-local
 * save/load/reset contract: a debounced save survives a "reload"
 * (a fresh hook instance reading the same localStorage), and reset
 * both cancels any pending save and clears what's stored.
 */

import { renderHook } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { TOPOLOGY_POSITIONS_KEY } from '../pages/topology/persistence';
import { useTopologyLayoutPersistence } from './useTopologyLayoutPersistence';

beforeEach(() => {
  window.localStorage.clear();
  vi.useFakeTimers();
});

afterEach(() => {
  vi.useRealTimers();
});

describe('useTopologyLayoutPersistence', () => {
  it('save is debounced — nothing is written until the timer fires', () => {
    const { result } = renderHook(() => useTopologyLayoutPersistence());

    result.current.savePositions({ router1: { x: 10, y: 20 } });

    expect(window.localStorage.getItem(TOPOLOGY_POSITIONS_KEY)).toBeNull();

    vi.runAllTimers();

    expect(JSON.parse(window.localStorage.getItem(TOPOLOGY_POSITIONS_KEY) ?? '{}')).toEqual({
      router1: { x: 10, y: 20 },
    });
  });

  it('save -> reload restores positions', () => {
    const { result: first } = renderHook(() => useTopologyLayoutPersistence());
    first.current.savePositions({ switch1: { x: 5, y: 5 } });
    vi.runAllTimers();

    // A fresh hook instance simulates a page reload — it must read
    // back what the previous instance persisted.
    const { result: second } = renderHook(() => useTopologyLayoutPersistence());
    expect(second.current.loadPositions()).toEqual({ switch1: { x: 5, y: 5 } });
  });

  it('rapid saves collapse to the last call (only one write)', () => {
    const { result } = renderHook(() => useTopologyLayoutPersistence());

    result.current.savePositions({ a: { x: 1, y: 1 } });
    result.current.savePositions({ a: { x: 2, y: 2 } });
    result.current.savePositions({ a: { x: 3, y: 3 } });
    vi.runAllTimers();

    expect(result.current.loadPositions()).toEqual({ a: { x: 3, y: 3 } });
  });

  it('reset clears saved positions', () => {
    const { result } = renderHook(() => useTopologyLayoutPersistence());
    result.current.savePositions({ fw1: { x: 1, y: 1 } });
    vi.runAllTimers();
    expect(result.current.loadPositions()).toEqual({ fw1: { x: 1, y: 1 } });

    result.current.resetPositions();

    expect(result.current.loadPositions()).toEqual({});
    expect(window.localStorage.getItem(TOPOLOGY_POSITIONS_KEY)).toBeNull();
  });

  it('reset cancels a pending debounced save', () => {
    const { result } = renderHook(() => useTopologyLayoutPersistence());
    result.current.savePositions({ ap1: { x: 9, y: 9 } });

    result.current.resetPositions();
    vi.runAllTimers();

    expect(result.current.loadPositions()).toEqual({});
  });
});
