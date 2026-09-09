import { waitFor } from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { clearRuntimeAPIToken, deduplicatedGet, setRuntimeAPIToken } from './requestCore';
import { requestJsonWithProgress } from './requestUpload';

class FakeXMLHttpRequest {
  static latest: FakeXMLHttpRequest | null = null;

  readonly upload = { onprogress: null as ((event: ProgressEvent) => void) | null };
  status = 401;
  statusText = 'Unauthorized';
  responseText = 'Unauthorized';
  timeout = 0;
  withCredentials = false;
  onabort: (() => void) | null = null;
  onerror: (() => void) | null = null;
  onload: (() => void) | null = null;
  ontimeout: (() => void) | null = null;

  constructor() {
    FakeXMLHttpRequest.latest = this;
  }

  open() {}
  setRequestHeader() {}
  abort() {
    this.onabort?.();
  }
  send() {
    this.onload?.();
  }
}

describe('requestJsonWithProgress', () => {
  beforeEach(() => {
    clearRuntimeAPIToken();
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ token: 'csrf-token' }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
      ),
    );
    vi.stubGlobal('XMLHttpRequest', FakeXMLHttpRequest);
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it('does not reuse a library read started before a successful upload', async () => {
    let finishRead!: (response: Response) => void;
    const pendingRead = new Promise<Response>((resolve) => {
      finishRead = resolve;
    });
    const mockFetch = vi
      .fn<typeof fetch>()
      .mockReturnValueOnce(pendingRead)
      .mockResolvedValueOnce(new Response(JSON.stringify({ token: 'csrf' })))
      .mockResolvedValueOnce(new Response(JSON.stringify([{ name: 'new.pcap' }])));
    vi.stubGlobal('fetch', mockFetch);
    vi.spyOn(FakeXMLHttpRequest.prototype, 'send').mockImplementation(function (
      this: FakeXMLHttpRequest,
    ) {
      this.status = 201;
      this.responseText = '{}';
      this.onload?.();
    });
    const first = deduplicatedGet('/api/v1/library/pcaps');
    await waitFor(() => expect(mockFetch).toHaveBeenCalledTimes(1));
    await requestJsonWithProgress('/api/v1/library/pcaps', {}, vi.fn());
    const afterUpload = deduplicatedGet('/api/v1/library/pcaps');
    try {
      expect(afterUpload).not.toBe(first);
      await expect(afterUpload).resolves.toEqual([{ name: 'new.pcap' }]);
      expect(mockFetch).toHaveBeenCalledTimes(3);
    } finally {
      finishRead(new Response('[]'));
      await first;
    }
  });

  it('raises the authentication event when an upload is unauthorized', async () => {
    setRuntimeAPIToken('revoked-token');
    const listener = vi.fn();
    window.addEventListener('niac:authentication-required', listener);

    await expect(requestJsonWithProgress('/api/v1/uploads', {}, vi.fn())).rejects.toThrow(
      'Unauthorized',
    );
    expect(listener).toHaveBeenCalledOnce();

    window.removeEventListener('niac:authentication-required', listener);
  });
});
