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

import type { RenderHookOptions } from '@testing-library/react';
import { renderHook as renderRawHook, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { ResourceProvider } from '../contexts/ResourceProvider';
import { renderWithResources } from '../test/renderWithResources';
import '../i18n';
import { useUIStore } from '../stores/ui-store';
import { useApiResource } from './useApiResource';

function renderHook<Result, Props>(
  callback: (props: Props) => Result,
  options?: RenderHookOptions<Props>,
) {
  return renderRawHook(callback, { ...options, wrapper: ResourceProvider });
}

describe('useApiResource errorToast option', () => {
  beforeEach(() => {
    useUIStore.getState().clearNotifications();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('does not touch the notification store when errorToast is unset', async () => {
    const fetcher = vi.fn().mockRejectedValue(new Error('boom'));

    const { result } = renderHook(() => useApiResource(fetcher, ['test-resource']));

    await waitFor(() => expect(result.current.error).not.toBeNull());
    expect(useUIStore.getState().notifications).toHaveLength(0);
  });

  it('pushes exactly one error toast into the shared store on failure', async () => {
    const fetcher = vi.fn().mockRejectedValue(new Error('daemon unreachable'));

    const { result } = renderHook(() =>
      useApiResource(fetcher, ['test-resource'], {
        errorToast: { title: 'Could not load devices' },
      }),
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

  it('does not re-toast an identical error on repeated requests', async () => {
    const fetcher = vi.fn().mockRejectedValue(new Error('daemon unreachable'));

    function Probe() {
      const { refetch } = useApiResource(fetcher, ['test-resource'], { errorToast: true });
      return (
        <button type="button" onClick={refetch}>
          Refresh
        </button>
      );
    }
    renderWithResources(<Probe />);

    await waitFor(() => expect(fetcher).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(useUIStore.getState().notifications).toHaveLength(1));

    const user = userEvent.setup();
    await user.click(screen.getByRole('button', { name: 'Refresh' }));
    await user.click(screen.getByRole('button', { name: 'Refresh' }));
    await user.click(screen.getByRole('button', { name: 'Refresh' }));

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
      ({ n }: { n: number }) => useApiResource(fetcher, ['test-resource', n], { errorToast: true }),
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

/**
 * An inline options object must not put the caller in a fetch loop.
 *
 * Every real call site writes `errorToast: { title: t('...') }` inline, so
 * the options object has a new identity on every render. While `run` named
 * that object as a dependency, `run` changed every render, the fetch effect
 * re-ran every render, and the state it set caused the next render: five
 * pages sat in an unbounded loop against the daemon. The page-story harness
 * measured 39,191 requests in ten seconds on the alerts page.
 */
describe('useApiResource — stable identity across renders', () => {
  it('fetches once when the options object is rebuilt on every render', async () => {
    const fetcher = vi.fn(() => Promise.resolve('value'));
    const { rerender, result } = renderHook(() =>
      // A fresh object literal each render, exactly as the pages write it.
      useApiResource(fetcher, ['test-resource'], { errorToast: { title: 'Failed' } }),
    );

    await waitFor(() => expect(result.current.data).toBe('value'));
    for (let i = 0; i < 5; i += 1) rerender();
    await waitFor(() => expect(result.current.loading).toBe(false));

    expect(fetcher).toHaveBeenCalledTimes(1);
  });

  it('fetches once when transform is declared inline', async () => {
    const fetcher = vi.fn(() => Promise.resolve(2));
    const { rerender, result } = renderHook(() =>
      useApiResource(fetcher, ['test-resource'], { transform: (n: number) => n * 2 }),
    );

    await waitFor(() => expect(result.current.data).toBe(4));
    for (let i = 0; i < 5; i += 1) rerender();

    expect(fetcher).toHaveBeenCalledTimes(1);
  });
});
