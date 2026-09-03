import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { required } from '../test/required';

// Mock fetch globally
const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

// Mock import.meta.env
vi.stubGlobal('import', { meta: { env: { VITE_API_BASE: '', VITE_API_TOKEN: 'test-token' } } });

/**
 * Minimal fake XMLHttpRequest for exercising uploadPcapWithProgress, which
 * uses XHR (not fetch) specifically so it can observe xhr.upload.onprogress
 * — something fetch has no cross-browser way to report.
 */
class FakeXHR {
  static instances: FakeXHR[] = [];

  method = '';
  url = '';
  readonly headers: Record<string, string> = {};
  withCredentials = false;
  timeout = 0;
  status = 0;
  statusText = '';
  responseText = '';
  sentBody: string | undefined;
  upload: { onprogress: ((event: ProgressEvent) => void) | null } = { onprogress: null };
  onload: (() => void) | null = null;
  onerror: (() => void) | null = null;
  ontimeout: (() => void) | null = null;
  onabort: (() => void) | null = null;

  constructor() {
    FakeXHR.instances.push(this);
  }

  open(method: string, url: string) {
    this.method = method;
    this.url = url;
  }

  setRequestHeader(key: string, value: string) {
    this.headers[key] = value;
  }

  send(body?: string) {
    this.sentBody = body;
  }

  abort() {
    this.onabort?.();
  }
}

describe('API Client', () => {
  beforeEach(() => {
    mockFetch.mockReset();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('toCamelCase response conversion', () => {
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
      const typed = result as unknown as Record<string, Record<string, string>>;
      expect(typed.outerKey?.innerKey).toBe('value');
    });

    it('handles arrays', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve([{ device_name: 'a' }, { device_name: 'b' }]),
      });

      const { fetchDevices } = await import('./client');
      const result = await fetchDevices('hospital');
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

      expect(mockFetch.mock.calls[0]?.[1]?.headers).toBeDefined();
    });
  });

  describe('csrf header', () => {
    it('fetches and includes CSRF token for state-changing requests', async () => {
      mockFetch
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({ token: 'csrf-token' }),
        })
        .mockResolvedValueOnce({
          ok: true,
          json: () => Promise.resolve({ packetsThreshold: 100, webhookUrl: '' }),
        });

      const { updateAlerts } = await import('./client');
      await updateAlerts({ packetsThreshold: 100, webhookUrl: '' });

      expect(mockFetch).toHaveBeenCalledTimes(2);
      expect(mockFetch.mock.calls[0]?.[0]).toContain('/api/v1/csrf-token');

      const headers = mockFetch.mock.calls[1]?.[1]?.headers as Headers;
      expect(headers.get('X-Csrf-Token')).toBe('csrf-token');
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

  describe('device create/update payloads', () => {
    beforeEach(() => {
      mockFetch.mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ success: true }),
      });
    });

    it('strips the server-computed protocols field from createDevice', async () => {
      const { createDevice } = await import('./client');
      await createDevice({
        hostname: 'sw1',
        mac: '00:11:22:33:44:55',
        protocols: ['snmp', 'lldp'],
      });

      const [, options] = required(mockFetch.mock.calls[0], 'the createDevice fetch call');
      const sent = JSON.parse((options as RequestInit).body as string);
      expect(sent).not.toHaveProperty('protocols');
      expect(sent).toMatchObject({ hostname: 'sw1', mac: '00:11:22:33:44:55' });
    });

    it('strips the server-computed protocols field from updateDevice', async () => {
      const { updateDevice } = await import('./client');
      await updateDevice('sw1', { mac: '00:11:22:33:44:55', protocols: ['snmp'] });

      const [, options] = required(mockFetch.mock.calls[0], 'the updateDevice fetch call');
      const sent = JSON.parse((options as RequestInit).body as string);
      expect(sent).not.toHaveProperty('protocols');
      expect(sent).toEqual({ mac: '00:11:22:33:44:55' });
    });
  });

  describe('uploadPcapWithProgress', () => {
    beforeEach(() => {
      FakeXHR.instances = [];
      vi.stubGlobal('XMLHttpRequest', FakeXHR);
    });

    it('reports upload progress and resolves with the camel-cased response', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ token: 'csrf-token' }),
      });

      const { uploadPcapWithProgress } = await import('./client');
      const onProgress = vi.fn();

      const promise = uploadPcapWithProgress({ filename: 'x.pcap', data: 'AAAA' }, onProgress);

      await vi.waitFor(() => expect(FakeXHR.instances).toHaveLength(1));
      const xhr = required(FakeXHR.instances[0], 'the upload XHR');
      expect(xhr.method).toBe('POST');
      // Headers normalizes names to lowercase when iterated.
      expect(xhr.headers['x-csrf-token']).toBe('csrf-token');

      xhr.upload.onprogress?.({ lengthComputable: true, loaded: 50, total: 100 } as ProgressEvent);
      expect(onProgress).toHaveBeenCalledWith(50);

      xhr.status = 200;
      xhr.responseText = JSON.stringify({
        success: true,
        analysis_id: 'abc123',
        message: 'Successfully analyzed 3 packets',
      });
      xhr.onload?.();

      await expect(promise).resolves.toEqual({
        success: true,
        analysisId: 'abc123',
        message: 'Successfully analyzed 3 packets',
      });
    });

    it('rejects with the structured over-limit error on 413', async () => {
      mockFetch.mockResolvedValueOnce({
        ok: true,
        json: () => Promise.resolve({ token: 'csrf-token' }),
      });

      const { uploadPcapWithProgress } = await import('./client');
      const promise = uploadPcapWithProgress({ filename: 'big.pcap', data: 'AAAA' }, vi.fn());

      await vi.waitFor(() => expect(FakeXHR.instances).toHaveLength(1));
      const xhr = required(FakeXHR.instances[0], 'the upload XHR');

      xhr.status = 413;
      xhr.statusText = 'Request Entity Too Large';
      xhr.responseText = JSON.stringify({
        error: 'request_too_large',
        message: 'Request body exceeds the 140.0 MB size limit',
        details: [{ field: 'body', issue: 'max_size_exceeded', value: '140.0 MB' }],
      });
      xhr.onload?.();

      await expect(promise).rejects.toMatchObject({
        code: 'request_too_large',
        status: 413,
        details: [{ field: 'body', issue: 'max_size_exceeded', value: '140.0 MB' }],
      });
    });
  });
});
