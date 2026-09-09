import { waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { clearRuntimeAPIToken, deduplicatedGet, request } from './requestCore';

const mockFetch = vi.fn<typeof fetch>();

function deferredResponse() {
  let resolve!: (response: Response) => void;
  const promise = new Promise<Response>((finish) => {
    resolve = finish;
  });
  return { promise, resolve };
}

describe('read-after-write request isolation', () => {
  beforeEach(() => {
    clearRuntimeAPIToken();
    mockFetch.mockReset();
    vi.stubGlobal('fetch', mockFetch);
  });

  afterEach(() => vi.unstubAllGlobals());

  it.each([
    ['POST', 201],
    ['PUT', 200],
    ['PATCH', 200],
    ['DELETE', 204],
  ])('starts a fresh read after successful %s (%i)', async (method, status) => {
    const firstResponse = deferredResponse();
    const nextResponse = deferredResponse();
    mockFetch
      .mockReturnValueOnce(firstResponse.promise)
      .mockResolvedValueOnce(new Response(JSON.stringify({ token: 'csrf' })))
      .mockResolvedValueOnce(new Response(status === 204 ? null : '{}', { status }))
      .mockReturnValueOnce(nextResponse.promise);

    const first = deduplicatedGet('/api/v1/config');
    await waitFor(() => expect(mockFetch).toHaveBeenCalledTimes(1));
    await request('/api/v1/config/devices/router', { method });
    const next = deduplicatedGet('/api/v1/config');
    try {
      expect(next).not.toBe(first);
      await waitFor(() => expect(mockFetch).toHaveBeenCalledTimes(4));
      firstResponse.resolve(new Response(JSON.stringify({ content: 'before' })));
      await first;
      expect(deduplicatedGet('/api/v1/config')).toBe(next);
      nextResponse.resolve(new Response(JSON.stringify({ content: 'after' })));
      await expect(next).resolves.toEqual({ content: 'after' });
    } finally {
      firstResponse.resolve(new Response('{}'));
      nextResponse.resolve(new Response('{}'));
      await Promise.all([first, next]);
    }
  });

  it('keeps sharing pending reads after a rejected write', async () => {
    const pending = deferredResponse();
    mockFetch
      .mockReturnValueOnce(pending.promise)
      .mockResolvedValueOnce(new Response(JSON.stringify({ token: 'csrf' })))
      .mockResolvedValueOnce(new Response('Invalid device', { status: 400 }));
    const first = deduplicatedGet('/api/v1/config');
    await waitFor(() => expect(mockFetch).toHaveBeenCalledTimes(1));
    await expect(request('/api/v1/config/devices/router', { method: 'PUT' })).rejects.toThrow(
      'Invalid device',
    );
    expect(deduplicatedGet('/api/v1/config')).toBe(first);
    pending.resolve(new Response('{}'));
    await first;
  });
});
