import { beforeEach, describe, expect, it, vi } from 'vitest';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

describe('runtime bearer authentication', () => {
  beforeEach(async () => {
    mockFetch.mockReset();
    const { clearRuntimeAPIToken } = await import('./requestCore');
    clearRuntimeAPIToken();
    localStorage.clear();
    sessionStorage.clear();
  });

  it('adds the runtime token to REST requests without persistent storage', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ scope: 'admin' }),
    });

    const { request, setRuntimeAPIToken } = await import('./requestCore');
    setRuntimeAPIToken('browser-token');
    await request('/api/v1/auth/scope');

    const headers = mockFetch.mock.calls[0][1]?.headers as Headers;
    expect(headers.get('Authorization')).toBe('Bearer browser-token');
    expect(localStorage.length).toBe(0);
    expect(sessionStorage.length).toBe(0);
  });

  it('clears the bearer and cached CSRF token together', async () => {
    mockFetch
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ token: 'old-csrf' }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({}),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({ token: 'new-csrf' }),
      })
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({}),
      });

    const { clearRuntimeAPIToken, request, setRuntimeAPIToken } = await import('./requestCore');
    setRuntimeAPIToken('first-token');
    await request('/api/v1/alerts', { method: 'PUT' });
    clearRuntimeAPIToken();
    setRuntimeAPIToken('second-token');
    await request('/api/v1/alerts', { method: 'PUT' });

    expect(mockFetch.mock.calls[0][0]).toContain('/api/v1/csrf-token');
    expect(mockFetch.mock.calls[2][0]).toContain('/api/v1/csrf-token');
    const secondHeaders = mockFetch.mock.calls[3][1]?.headers as Headers;
    expect(secondHeaders.get('Authorization')).toBe('Bearer second-token');
    expect(secondHeaders.get('X-Csrf-Token')).toBe('new-csrf');
  });

  it('raises the authentication event when CSRF retrieval is unauthorized', async () => {
    mockFetch.mockResolvedValueOnce(
      new Response('Unauthorized', { status: 401, statusText: 'Unauthorized' }),
    );
    const listener = vi.fn();
    window.addEventListener('niac:authentication-required', listener);

    const { request, setRuntimeAPIToken } = await import('./requestCore');
    setRuntimeAPIToken('revoked-token');

    await expect(request('/api/v1/alerts', { method: 'PUT' })).rejects.toThrow('Unauthorized');
    expect(listener).toHaveBeenCalledOnce();

    window.removeEventListener('niac:authentication-required', listener);
  });

  it('does not report session expiry during the initial authentication check', async () => {
    mockFetch.mockResolvedValueOnce(new Response('Unauthorized', { status: 401 }));
    const listener = vi.fn();
    window.addEventListener('niac:authentication-required', listener);

    const { validateRuntimeAuthentication } = await import('./requestCore');
    await expect(validateRuntimeAuthentication()).rejects.toThrow('Unauthorized');
    expect(listener).not.toHaveBeenCalled();

    window.removeEventListener('niac:authentication-required', listener);
  });
});
