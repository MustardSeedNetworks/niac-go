import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

// Mock fetch globally
const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

// Mock import.meta.env
vi.stubGlobal('import', { meta: { env: { VITE_API_BASE: '', VITE_API_TOKEN: 'test-token' } } });

describe('API Client', () => {
  beforeEach(() => {
    mockFetch.mockReset();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('toCamelCase / toSnakeCase helpers', () => {
    it('converts snake_case keys to camelCase', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ device_name: 'test', ip_address: '1.2.3.4' }),
      });

      // Import dynamically to use mocked environment
      const { fetchVersion } = await import('./client');
      const result = await fetchVersion();
      expect(result).toHaveProperty('deviceName');
      expect(result).toHaveProperty('ipAddress');
    });

    it('handles nested objects', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ outer_key: { inner_key: 'value' } }),
      });

      const { fetchVersion } = await import('./client');
      const result = await fetchVersion();
      const typed = result as Record<string, Record<string, string>>;
      expect(typed.outerKey).toBeDefined();
      expect(typed.outerKey.innerKey).toBe('value');
    });

    it('handles arrays', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve([{ device_name: 'a' }, { device_name: 'b' }]),
      });

      const { fetchDevices } = await import('./client');
      const result = await fetchDevices();
      expect(result).toHaveLength(2);
    });
  });

  describe('URL building', () => {
    it('prepends API base to relative paths', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({}),
      });

      const { fetchVersion } = await import('./client');
      await fetchVersion();

      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining('/api/v1/version'),
        expect.any(Object),
      );
    });
  });

  describe('error handling', () => {
    it('throws on non-OK responses', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 404,
        statusText: 'Not Found',
        text: () => Promise.resolve('Not Found'),
      });

      const { fetchVersion } = await import('./client');
      await expect(fetchVersion()).rejects.toThrow('Not Found');
    });

    it('throws network error for TypeError', async () => {
      // Skip backoff delays by resolving setTimeout immediately
      const origSetTimeout = globalThis.setTimeout;
      vi.stubGlobal('setTimeout', (fn: () => void) => origSetTimeout(fn, 0));

      // Mock all 4 attempts (initial + 3 retries) to reject with TypeError
      mockFetch
        .mockRejectedValueOnce(new TypeError('Failed to fetch'))
        .mockRejectedValueOnce(new TypeError('Failed to fetch'))
        .mockRejectedValueOnce(new TypeError('Failed to fetch'))
        .mockRejectedValueOnce(new TypeError('Failed to fetch'));

      const { fetchVersion } = await import('./client');
      await expect(fetchVersion()).rejects.toThrow('Network error');
      vi.stubGlobal('setTimeout', origSetTimeout);
    });

    it('throws timeout error for AbortError', async () => {
      mockFetch.mockRejectedValueOnce(new DOMException('Aborted', 'AbortError'));

      const { fetchVersion } = await import('./client');
      await expect(fetchVersion()).rejects.toThrow('Request timeout');
    });
  });

  describe('auth header', () => {
    it('includes Authorization header when token is set', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({}),
      });

      const { fetchVersion } = await import('./client');
      await fetchVersion();

      const callArgs = mockFetch.mock.calls[0];
      const headers = callArgs[1]?.headers;
      expect(headers).toBeDefined();
    });
  });

  describe('retry logic', () => {
    let origSetTimeout: typeof globalThis.setTimeout;

    beforeEach(() => {
      // Skip backoff delays by making setTimeout resolve immediately
      origSetTimeout = globalThis.setTimeout;
      vi.stubGlobal('setTimeout', (fn: () => void) => origSetTimeout(fn, 0));
    });

    afterEach(() => {
      vi.stubGlobal('setTimeout', origSetTimeout);
    });

    it('retries on 5xx errors', async () => {
      // First two calls return 500, third succeeds
      mockFetch
        .mockResolvedValueOnce({ ok: false, status: 500, statusText: 'Internal Server Error' })
        .mockResolvedValueOnce({ ok: false, status: 503, statusText: 'Service Unavailable' })
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({ version: '1.0' }),
        });

      const { fetchVersion } = await import('./client');
      const result = await fetchVersion();
      expect(result).toHaveProperty('version', '1.0');
      expect(mockFetch).toHaveBeenCalledTimes(3);
    });

    it('does not retry on 4xx errors', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: false,
        status: 400,
        statusText: 'Bad Request',
        text: () => Promise.resolve('Bad Request'),
      });

      const { fetchVersion } = await import('./client');
      await expect(fetchVersion()).rejects.toThrow('Bad Request');
      expect(mockFetch).toHaveBeenCalledTimes(1);
    });

    it('retries on network errors (TypeError)', async () => {
      mockFetch
        .mockRejectedValueOnce(new TypeError('Failed to fetch'))
        .mockRejectedValueOnce(new TypeError('Failed to fetch'))
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({}),
        });

      const { fetchVersion } = await import('./client');
      const result = await fetchVersion();
      expect(result).toBeDefined();
      expect(mockFetch).toHaveBeenCalledTimes(3);
    });
  });
});
