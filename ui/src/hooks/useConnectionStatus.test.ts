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

it('times out a stalled request and recovers on the next check', async () => {
  mockedRequest.mockImplementationOnce(
    (_path, options) =>
      new Promise((_resolve, reject) => {
        options?.signal?.addEventListener('abort', () =>
          reject(new DOMException('Aborted', 'AbortError')),
        );
      }),
  );
  const { result } = renderHook(useConnectionStatus);
  const signal = mockedRequest.mock.calls[0]?.[1]?.signal;
  expect(result.current).toBe('checking');
  const deadline: Promise<void> = act(async () => {
    await vi.advanceTimersByTimeAsync(5_000);
  });
  await deadline;
  expect(signal?.aborted).toBe(true);
  expect(result.current).toBe('disconnected');
  const recovery: Promise<void> = act(async () => {
    await vi.advanceTimersByTimeAsync(10_000);
  });
  await recovery;
  expect(result.current).toBe('connected');
  expect(mockedRequest).toHaveBeenCalledTimes(2);
});
