/**
 * useUnsavedChangesGuard.test.tsx
 *
 * Data-safety regression test: the device editor previously had no
 * unsaved-changes guard at all — navigating away (in-app link click, the
 * editor's own Back button, or closing/refreshing the tab) silently
 * discarded edits. This pins the three escape hatches:
 *   1. requestNavigate() — used by the editor's "Back" button — queues a
 *      confirmation instead of navigating immediately while dirty, and
 *      navigates straight through when clean.
 *   2. An in-app `<a href>` click is intercepted while dirty and only
 *      proceeds after confirmNavigate(); cancelNavigate() discards the
 *      pending navigation and never calls the navigate callback.
 *   3. beforeunload is guarded (preventDefault + returnValue set) while
 *      dirty, and left alone when clean.
 */
import { act, renderHook } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { useUnsavedChangesGuard } from './useUnsavedChangesGuard';

afterEach(() => {
  vi.restoreAllMocks();
});

describe('useUnsavedChangesGuard', () => {
  it('navigates immediately via requestNavigate when not dirty', () => {
    const navigate = vi.fn();
    const { result } = renderHook(() => useUnsavedChangesGuard(false, navigate));

    act(() => result.current.requestNavigate('/device-config'));

    expect(navigate).toHaveBeenCalledWith('/device-config');
    expect(result.current.pendingPath).toBeNull();
  });

  it('queues a confirmation via requestNavigate when dirty, and only navigates on confirm', () => {
    const navigate = vi.fn();
    const { result } = renderHook(() => useUnsavedChangesGuard(true, navigate));

    act(() => result.current.requestNavigate('/device-config'));
    expect(navigate).not.toHaveBeenCalled();
    expect(result.current.pendingPath).toBe('/device-config');

    act(() => result.current.confirmNavigate());
    expect(navigate).toHaveBeenCalledWith('/device-config');
    expect(result.current.pendingPath).toBeNull();
  });

  it('cancelNavigate discards the pending navigation without calling navigate', () => {
    const navigate = vi.fn();
    const { result } = renderHook(() => useUnsavedChangesGuard(true, navigate));

    act(() => result.current.requestNavigate('/device-config'));
    act(() => result.current.cancelNavigate());

    expect(navigate).not.toHaveBeenCalled();
    expect(result.current.pendingPath).toBeNull();
  });

  it('intercepts an in-app link click while dirty and proceeds only on confirm', () => {
    const navigate = vi.fn();
    const { result } = renderHook(() => useUnsavedChangesGuard(true, navigate));

    const anchor = document.createElement('a');
    anchor.setAttribute('href', '/device-config');
    document.body.appendChild(anchor);

    const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true, button: 0 });
    act(() => {
      anchor.dispatchEvent(clickEvent);
    });

    expect(clickEvent.defaultPrevented).toBe(true);
    expect(navigate).not.toHaveBeenCalled();
    expect(result.current.pendingPath).toBe('/device-config');

    act(() => result.current.confirmNavigate());
    expect(navigate).toHaveBeenCalledWith('/device-config');

    document.body.removeChild(anchor);
  });

  it('does not intercept an in-app link click when not dirty', () => {
    const navigate = vi.fn();
    const { result } = renderHook(() => useUnsavedChangesGuard(false, navigate));

    const anchor = document.createElement('a');
    anchor.setAttribute('href', '/device-config');
    document.body.appendChild(anchor);

    const clickEvent = new MouseEvent('click', { bubbles: true, cancelable: true, button: 0 });
    act(() => {
      anchor.dispatchEvent(clickEvent);
    });

    expect(clickEvent.defaultPrevented).toBe(false);
    expect(result.current.pendingPath).toBeNull();

    document.body.removeChild(anchor);
  });

  it('guards beforeunload while dirty', () => {
    renderHook(() => useUnsavedChangesGuard(true, vi.fn()));

    const event = new Event('beforeunload', { cancelable: true }) as BeforeUnloadEvent;
    const preventDefaultSpy = vi.spyOn(event, 'preventDefault');
    act(() => {
      window.dispatchEvent(event);
    });

    expect(preventDefaultSpy).toHaveBeenCalled();
  });

  it('does not guard beforeunload when clean', () => {
    renderHook(() => useUnsavedChangesGuard(false, vi.fn()));

    const event = new Event('beforeunload', { cancelable: true }) as BeforeUnloadEvent;
    const preventDefaultSpy = vi.spyOn(event, 'preventDefault');
    act(() => {
      window.dispatchEvent(event);
    });

    expect(preventDefaultSpy).not.toHaveBeenCalled();
  });
});
