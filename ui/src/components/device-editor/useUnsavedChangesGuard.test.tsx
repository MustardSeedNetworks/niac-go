import { act, renderHook } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { MemoryDataRouter } from '../../test/MemoryDataRouter';
import { useUnsavedChangesGuard } from './useUnsavedChangesGuard';

describe('useUnsavedChangesGuard local actions', () => {
  it.each([false, true])('guards document unload when dirty=%s', (dirty) => {
    renderHook(() => useUnsavedChangesGuard(dirty), { wrapper: MemoryDataRouter });
    const event = new Event('beforeunload', { cancelable: true });
    window.dispatchEvent(event);
    expect(event.defaultPrevented).toBe(dirty);
  });
  it('executes a selection action immediately when clean', () => {
    const run = vi.fn();
    const { result } = renderHook(() => useUnsavedChangesGuard(false), {
      wrapper: MemoryDataRouter,
    });
    act(() => result.current.requestAction(run));
    expect(run).toHaveBeenCalledOnce();
    expect(result.current.pending).toBe(false);
  });
  it('cancels or confirms a dirty selection action without changing route', () => {
    const run = vi.fn();
    const { result } = renderHook(() => useUnsavedChangesGuard(true), {
      wrapper: MemoryDataRouter,
    });
    act(() => result.current.requestAction(run));
    expect(run).not.toHaveBeenCalled();
    expect(result.current.pending).toBe(true);
    act(() => result.current.cancelNavigate());
    expect(result.current.pending).toBe(false);
    act(() => result.current.requestAction(run));
    act(() => result.current.confirmNavigate());
    expect(run).toHaveBeenCalledOnce();
  });
});
