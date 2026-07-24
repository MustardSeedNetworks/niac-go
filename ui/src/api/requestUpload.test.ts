import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { clearRuntimeAPIToken, setRuntimeAPIToken } from './requestCore';
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
    vi.unstubAllGlobals();
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
