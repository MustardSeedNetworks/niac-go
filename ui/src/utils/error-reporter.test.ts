import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AUTH_FAILURE_EVENT } from '../api/requestCore';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

const MAX_BATCH_SIZE = 10;

function fillBatch(reportError: (error: unknown, context?: string) => void) {
  for (let i = 0; i < MAX_BATCH_SIZE; i += 1) {
    reportError(new Error(`boom ${i}`), 'error-boundary');
  }
}

function callTo(path: string) {
  return mockFetch.mock.calls.find(([url]) => String(url).includes(path));
}

describe('client error reporter', () => {
  beforeEach(async () => {
    vi.resetModules();
    mockFetch.mockReset();
    vi.stubEnv('DEV', false);
    const { clearRuntimeAPIToken, setRuntimeAPIToken } = await import('../api/requestCore');
    clearRuntimeAPIToken();
    setRuntimeAPIToken('operator-token');
  });

  it('sends the report with the bearer token and the CSRF header', async () => {
    mockFetch.mockImplementation(async (url: string) =>
      String(url).includes('/api/v1/csrf-token')
        ? new Response(JSON.stringify({ token: 'csrf-abc' }), { status: 200 })
        : new Response('', { status: 204 }),
    );

    const { reportError } = await import('./error-reporter');
    fillBatch(reportError);
    await vi.waitFor(() => expect(callTo('/api/v1/client-errors')).toBeDefined());

    const [, init] = callTo('/api/v1/client-errors') as [string, RequestInit];
    const headers = init.headers as Headers;
    expect(init.method).toBe('POST');
    expect(headers.get('Authorization')).toBe('Bearer operator-token');
    expect(headers.get('X-Csrf-Token')).toBe('csrf-abc');
    expect(JSON.parse(String(init.body)).errors).toHaveLength(MAX_BATCH_SIZE);
  });

  it('does not sign the operator out when the report is rejected with 401', async () => {
    const authFailure = vi.fn();
    window.addEventListener(AUTH_FAILURE_EVENT, authFailure);
    mockFetch.mockImplementation(async (url: string) =>
      String(url).includes('/api/v1/csrf-token')
        ? new Response(JSON.stringify({ token: 'csrf-abc' }), { status: 200 })
        : new Response('unauthorized', { status: 401 }),
    );

    try {
      const { reportError } = await import('./error-reporter');
      fillBatch(reportError);
      await vi.waitFor(() => expect(callTo('/api/v1/client-errors')).toBeDefined());
      await vi.waitFor(() => expect(mockFetch).toHaveBeenCalledTimes(2));
      expect(authFailure).not.toHaveBeenCalled();
    } finally {
      window.removeEventListener(AUTH_FAILURE_EVENT, authFailure);
    }
  });

  it('does not sign the operator out when the CSRF token itself is refused', async () => {
    const authFailure = vi.fn();
    window.addEventListener(AUTH_FAILURE_EVENT, authFailure);
    mockFetch.mockImplementation(async () => new Response('unauthorized', { status: 401 }));

    try {
      const { reportError } = await import('./error-reporter');
      fillBatch(reportError);
      await vi.waitFor(() => expect(mockFetch).toHaveBeenCalled());
      expect(authFailure).not.toHaveBeenCalled();
    } finally {
      window.removeEventListener(AUTH_FAILURE_EVENT, authFailure);
    }
  });
});
