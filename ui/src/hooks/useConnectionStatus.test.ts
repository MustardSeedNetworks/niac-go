import { cleanup, renderHook } from '@testing-library/react';
import { act } from 'react';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import { request } from '../api/requestCore';
import { useConnectionStatus } from './useConnectionStatus';

vi.mock('../api/requestCore', () => ({ request: vi.fn() }));
const mockedRequest = vi.mocked(request);

beforeEach(() => {
  vi.useFakeTimers();
  mockedRequest.mockReset().mockResolvedValue({});
});
afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

it('checks once per interval and reports recovery', async () => {
  mockedRequest.mockRejectedValueOnce(new Error('offline'));
  const { result } = renderHook(useConnectionStatus);
  const initialCheck: Promise<void> = act(async () => {
    await vi.advanceTimersByTimeAsync(0);
  });
  await initialCheck;
  expect(result.current).toBe('disconnected');
  expect(mockedRequest).toHaveBeenCalledTimes(1);
  const nextCheck: Promise<void> = act(async () => {
    await vi.advanceTimersByTimeAsync(15_000);
  });
  await nextCheck;
  expect(result.current).toBe('connected');
  expect(mockedRequest).toHaveBeenCalledTimes(2);
});

it('aborts its pending request and removes every timer on unmount', async () => {
  mockedRequest.mockImplementation(
    (_path, options) =>
      new Promise((_resolve, reject) => {
        options?.signal?.addEventListener('abort', () =>
          reject(new DOMException('Aborted', 'AbortError')),
        );
      }),
  );
  const { unmount } = renderHook(useConnectionStatus);
  const signal = mockedRequest.mock.calls[0]?.[1]?.signal;
  expect(signal?.aborted).toBe(false);
  unmount();
  expect(signal?.aborted).toBe(true);
  const remainingTimers: Promise<void> = act(async () => {
    await vi.advanceTimersByTimeAsync(30_000);
  });
  await remainingTimers;
  expect(mockedRequest).toHaveBeenCalledTimes(1);
  expect(vi.getTimerCount()).toBe(0);
});
