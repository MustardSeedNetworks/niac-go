/**
 * useApiResource.test.tsx — locks the opt-in `errorToast` seam (PR
 * "3a — global error surfacing via toasts").
 *
 * useApiResource is the shared fetch/poll hook behind most pages. Most
 * callers don't want every poll failure to spam a toast (e.g. a
 * background poller that expects the daemon to be offline sometimes),
 * so toasting is opt-in via `errorToast: true`. When enabled:
 *   - a fetch failure pushes exactly one error notification into the
 *     shared ui-store (the seam ToastContainer renders from)
 *   - repeated ticks with the *same* error message don't re-toast
 *   - a new/changed error message (or a recovery-then-refail) toasts again
 * When not enabled, failures still populate `error` for the caller's own
 * UI, but never touch the notification store.
 */
import { renderHook, waitFor } from '@testing-library/react';
import { act } from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { useUIStore } from '../stores/ui-store';
import { useApiResource } from './useApiResource';

describe('useApiResource errorToast option', () => {
  beforeEach(() => {
    useUIStore.getState().clearNotifications();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('does not touch the notification store when errorToast is unset', async () => {
    const fetcher = vi.fn().mockRejectedValue(new Error('boom'));

    const { result } = renderHook(() => useApiResource(fetcher, []));

    await waitFor(() => expect(result.current.error).not.toBeNull());
    expect(useUIStore.getState().notifications).toHaveLength(0);
  });

  it('pushes exactly one error toast into the shared store on failure', async () => {
    const fetcher = vi.fn().mockRejectedValue(new Error('daemon unreachable'));

    const { result } = renderHook(() =>
      useApiResource(fetcher, [], { errorToast: { title: 'Could not load devices' } }),
    );

    await waitFor(() => expect(result.current.error).not.toBeNull());

    const notifications = useUIStore.getState().notifications;
    expect(notifications).toHaveLength(1);
    expect(notifications[0]).toMatchObject({
      type: 'error',
      title: 'Could not load devices',
      message: 'daemon unreachable',
    });
  });

  it('does not re-toast an identical error on repeated poll ticks', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const fetcher = vi.fn().mockRejectedValue(new Error('daemon unreachable'));

    renderHook(() => useApiResource(fetcher, [], { intervalMs: 1000, errorToast: true }));

    await waitFor(() => expect(fetcher).toHaveBeenCalledTimes(1));
    expect(useUIStore.getState().notifications).toHaveLength(1);

    await act(async () => {
      vi.advanceTimersByTime(3000);
      await Promise.resolve();
    });

    await waitFor(() => expect(fetcher).toHaveBeenCalledTimes(4));
    // Same error message on every tick — still just the one toast.
    expect(useUIStore.getState().notifications).toHaveLength(1);
  });

  it('re-toasts after recovering and failing again with a new message', async () => {
    const fetcher = vi
      .fn()
      .mockRejectedValueOnce(new Error('first failure'))
      .mockResolvedValueOnce('ok')
      .mockRejectedValueOnce(new Error('second failure'));

    const { result, rerender } = renderHook(
      ({ n }: { n: number }) => useApiResource(fetcher, [n], { errorToast: true }),
      { initialProps: { n: 0 } },
    );

    await waitFor(() => expect(result.current.error?.message).toBe('first failure'));
    expect(useUIStore.getState().notifications).toHaveLength(1);

    rerender({ n: 1 });
    await waitFor(() => expect(result.current.data).toBe('ok'));

    rerender({ n: 2 });
    await waitFor(() => expect(result.current.error?.message).toBe('second failure'));

    expect(useUIStore.getState().notifications).toHaveLength(2);
  });
});
