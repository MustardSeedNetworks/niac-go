import { onlineManager } from '@tanstack/react-query';
import { act, renderHook, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { ResourceProvider } from '../contexts/ResourceProvider';
import { useApiResource } from './useApiResource';

const wrapper = ResourceProvider;

describe('shared resource queries', () => {
  it('can read and refresh the local daemon when the browser reports offline', async () => {
    const wasOnline = onlineManager.isOnline();
    onlineManager.setOnline(false);
    const fetcher = vi.fn().mockResolvedValueOnce('initial').mockResolvedValue('refreshed');
    const { result, unmount } = renderHook(() => useApiResource(fetcher, ['config']), { wrapper });
    try {
      await waitFor(() => expect(result.current.data).toBe('initial'));
      act(() => {
        void result.current.refetch();
      });
      await waitFor(() => expect(result.current.data).toBe('refreshed'));
      expect(fetcher).toHaveBeenCalledTimes(2);
    } finally {
      unmount();
      onlineManager.setOnline(wasOnline);
    }
  });

  it('shares an in-flight resource across consumers', async () => {
    const fetcher = vi.fn().mockResolvedValue('devices');
    const { result } = renderHook(
      () => [
        useApiResource(fetcher, ['devices', 'hospital']),
        useApiResource(fetcher, ['devices', 'hospital']),
      ],
      { wrapper },
    );
    await waitFor(() => expect(result.current[1]?.data).toBe('devices'));
    expect(fetcher).toHaveBeenCalledTimes(1);
  });

  it('clears the previous session while a new session loads', async () => {
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce('hospital')
      .mockReturnValue(new Promise(() => {}));
    const { result, rerender } = renderHook(
      ({ session }) => useApiResource(fetcher, ['devices', session]),
      { initialProps: { session: 'hospital' }, wrapper },
    );
    await waitFor(() => expect(result.current.data).toBe('hospital'));
    rerender({ session: 'warehouse' });
    expect(result.current.data).toBeNull();
    expect(result.current.loading).toBe(true);
  });

  it('keeps data visible during background refresh', async () => {
    let finish: ((value: string) => void) | undefined;
    const fetcher = vi
      .fn()
      .mockResolvedValueOnce('first')
      .mockImplementation(
        () =>
          new Promise<string>((resolve) => {
            finish = resolve;
          }),
      );
    const { result } = renderHook(() => useApiResource(fetcher, ['config']), { wrapper });
    await waitFor(() => expect(result.current.data).toBe('first'));
    act(() => {
      void result.current.refetch();
    });
    expect(result.current.loading).toBe(false);
    expect(result.current.data).toBe('first');
    act(() => {
      finish?.('second');
    });
    await waitFor(() => expect(result.current.data).toBe('second'));
  });
});
