import { ApiError, type ApiErrorDetail, NetworkError, TimeoutError } from './errors';

/**
 * Core HTTP request infrastructure for the API client. Everything that
 * is shared by every endpoint lives here:
 *
 *   - case conversion (toSnakeCase out, toCamelCase in)
 *   - bearer token + CSRF header injection
 *   - timeout + signal forwarding
 *   - retry-with-backoff for 5xx and network errors
 *   - in-flight GET deduplication
 *
 * Endpoint wrappers live in endpoints.ts and import request / requestJson
 * / deduplicatedGet from here.
 */

const API_BASE = import.meta.env.VITE_API_BASE ?? '';
// API_TOKEN is only used in development mode to avoid embedding secrets in production bundles.
const API_TOKEN = import.meta.env.DEV ? (import.meta.env.VITE_API_TOKEN ?? '') : '';

const isPlainObject = (value: unknown): value is Record<string, unknown> =>
  value !== null &&
  typeof value === 'object' &&
  !Array.isArray(value) &&
  !(value instanceof Date) &&
  !(value instanceof File) &&
  !(value instanceof Blob) &&
  !(value instanceof FormData);

const toCamelKey = (key: string) =>
  key.replace(/_([a-z0-9])/g, (_, char: string) => char.toUpperCase());

export const toCamelCase = <T>(value: T): T => {
  if (Array.isArray(value)) {
    return value.map((item) => toCamelCase(item)) as T;
  }
  if (isPlainObject(value)) {
    const result: Record<string, unknown> = {};
    for (const [key, entry] of Object.entries(value)) {
      const camelKey = key.includes('_') ? toCamelKey(key) : key;
      result[camelKey] = toCamelCase(entry);
    }
    return result as T;
  }
  return value;
};

const toSnakeKey = (key: string) =>
  key
    .replace(/([a-z0-9])([A-Z])/g, '$1_$2')
    .replace(/([A-Z]+)([A-Z][a-z])/g, '$1_$2')
    .toLowerCase();

export const toSnakeCase = <T>(value: T): T => {
  if (Array.isArray(value)) {
    return value.map((item) => toSnakeCase(item)) as T;
  }
  if (isPlainObject(value)) {
    const result: Record<string, unknown> = {};
    for (const [key, entry] of Object.entries(value)) {
      result[toSnakeKey(key)] = toSnakeCase(entry);
    }
    return result as T;
  }
  return value;
};

export function buildUrl(path: string) {
  if (path.startsWith('http')) {
    return path;
  }
  if (path.startsWith('/')) {
    return `${API_BASE}${path}`;
  }
  return `${API_BASE}/${path}`;
}

const REQUEST_TIMEOUT_MS = 30_000;
const MAX_RETRIES = 3;
const BASE_DELAY_MS = 1_000;
const CSRF_TOKEN_PATH = '/api/v1/csrf-token';

let csrfTokenPromise: Promise<string> | null = null;

interface RetryConfig {
  readonly maxRetries: number;
  readonly baseDelay: number;
}

const DEFAULT_RETRY: RetryConfig = { maxRetries: MAX_RETRIES, baseDelay: BASE_DELAY_MS };

function isStateChangingMethod(method: string | undefined) {
  const normalized = (method ?? 'GET').toUpperCase();
  return (
    normalized === 'POST' ||
    normalized === 'PUT' ||
    normalized === 'PATCH' ||
    normalized === 'DELETE'
  );
}

async function fetchCSRFToken() {
  if (csrfTokenPromise) return csrfTokenPromise;

  csrfTokenPromise = (async () => {
    const headers = new Headers();
    headers.set('Accept', 'application/json');
    if (API_TOKEN) {
      headers.set('Authorization', `Bearer ${API_TOKEN}`);
    }

    const response = await fetch(buildUrl(CSRF_TOKEN_PATH), {
      headers,
      credentials: 'same-origin',
    });
    if (!response.ok) {
      const text = await response.text();
      throw new ApiError(text || response.statusText, response.status);
    }

    const data = (await response.json()) as { token?: string };
    if (!data.token) {
      throw new ApiError('CSRF token response did not include a token', response.status);
    }

    return data.token;
  })();

  try {
    return await csrfTokenPromise;
  } catch (error) {
    csrfTokenPromise = null;
    throw error;
  }
}

/**
 * Force the next CSRF token use to refetch from the server. Called when a
 * state-changing request fails with csrf_token_invalid — typically because
 * the daemon was restarted and rotated its in-memory token, leaving the
 * client with a stale cached promise.
 */
function resetCSRFToken() {
  csrfTokenPromise = null;
}

export async function buildRequestHeaders(path: string, init: RequestInit) {
  const headers = new Headers(init.headers);
  headers.set('Accept', 'application/json');
  if (API_TOKEN) {
    headers.set('Authorization', `Bearer ${API_TOKEN}`);
  }
  if (isStateChangingMethod(init.method) && path !== CSRF_TOKEN_PATH) {
    headers.set('X-Csrf-Token', await fetchCSRFToken());
  }

  return headers;
}

/**
 * Server error payloads are JSON like `{error: "config_read_failed",
 * message: "Failed to read configuration", ...}`. Surface the message
 * field as the throwable's .message so callers don't end up rendering
 * the raw JSON blob; copy the error field into ApiError.code so feature
 * pages can branch on the failure kind.
 */
export function parseApiError(text: string, status: number, fallback = ''): ApiError {
  if (text) {
    try {
      const parsed = JSON.parse(text) as {
        error?: string;
        message?: string;
        details?: ApiErrorDetail[];
      };
      const message = parsed.message || fallback || text;
      const code = parsed.error || 'API_ERROR';
      return new ApiError(message, status, code, parsed.details ?? []);
    } catch {
      // Body wasn't JSON — fall through to raw text.
    }
  }
  return new ApiError(text || fallback || 'Request failed', status);
}

// FIX #175: Check if an error is retryable (network errors thrown by fetch()).
// fetch() only ever rejects with a TypeError (network failure) or an
// AbortError; it never rejects with a Response — 5xx responses resolve
// normally and are handled via isRetryableStatus() on response.status below.
function isRetryableError(error: unknown): boolean {
  return error instanceof TypeError;
}

// FIX #175: Check if a response status is retryable.
function isRetryableStatus(status: number): boolean {
  return status >= 500;
}

// FIX #179: Accept optional signal parameter to allow caller-provided AbortController.
export async function request<T>(
  path: string,
  init: RequestInit = {},
  retry: RetryConfig = DEFAULT_RETRY,
) {
  const externalSignal = init.signal;
  let lastError: unknown;

  for (let attempt = 0; attempt <= retry.maxRetries; attempt++) {
    // FIX #175: Exponential backoff between retries
    if (attempt > 0) {
      const delay = retry.baseDelay * 2 ** (attempt - 1);
      await new Promise((resolve) => setTimeout(resolve, delay));
    }

    // Don't retry if the caller's signal was already aborted
    if (externalSignal?.aborted) {
      throw new DOMException('Request aborted', 'AbortError');
    }

    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);

    // If caller provides a signal, abort our controller when it fires
    const onExternalAbort = () => controller.abort();
    externalSignal?.addEventListener('abort', onExternalAbort);

    try {
      const headers = await buildRequestHeaders(path, init);

      const response = await fetch(buildUrl(path), {
        ...init,
        headers,
        credentials: 'same-origin',
        signal: controller.signal,
      });

      clearTimeout(timeout);

      if (!response.ok) {
        // FIX #175: Retry on 5xx, don't retry on 4xx
        if (isRetryableStatus(response.status) && attempt < retry.maxRetries) {
          lastError = new Error(response.statusText);
          continue;
        }
        const text = await response.text();
        // If the server rejected a stale CSRF token (typically because the
        // daemon was restarted and rotated its in-memory secret), drop the
        // cached token and retry once. Without this the SPA would keep
        // sending the stale token on every state-changing request until a
        // full reload.
        if (
          response.status === 403 &&
          isStateChangingMethod(init.method) &&
          /csrf_token_invalid/i.test(text) &&
          attempt < retry.maxRetries
        ) {
          resetCSRFToken();
          lastError = parseApiError(text, response.status);
          continue;
        }
        throw parseApiError(text, response.status, response.statusText);
      }

      const data = (await response.json()) as T;
      return toCamelCase(data);
    } catch (err) {
      clearTimeout(timeout);

      // Never retry aborts from caller's signal
      if (externalSignal?.aborted) {
        throw new DOMException('Request aborted', 'AbortError');
      }

      if (err instanceof DOMException && err.name === 'AbortError') {
        throw new TimeoutError();
      }

      // FIX #175: Retry network errors
      if (isRetryableError(err) && attempt < retry.maxRetries) {
        lastError = err;
        continue;
      }

      if (err instanceof TypeError) {
        throw new NetworkError();
      }
      throw err;
    } finally {
      clearTimeout(timeout);
      externalSignal?.removeEventListener('abort', onExternalAbort);
    }
  }

  // All retries exhausted
  if (lastError instanceof TypeError) {
    throw new NetworkError();
  }
  throw lastError ?? new ApiError('Request failed after retries', 0);
}

/**
 * requestJson is the JSON-bodied cousin of request(). Camel-case in the
 * payload is converted to snake_case so endpoints match the Go API
 * server's struct tags without each call site having to think about it.
 */
export const requestJson = <T>(path: string, payload: unknown, init: RequestInit = {}) =>
  request<T>(path, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...(init.headers ?? {}) },
    body: JSON.stringify(toSnakeCase(payload)),
  });

/**
 * requestText fetches a non-JSON body (e.g. the topology DOT/GraphML export)
 * through the same auth + URL machinery as request(), returning the raw text.
 * It shares buildRequestHeaders/buildUrl so auth stays identical, skips the
 * camel-case pass (the body isn't JSON), and keeps error handling aligned with
 * request(). Idempotent GET exports don't need the retry loop.
 */
export async function requestText(path: string, init: RequestInit = {}): Promise<string> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);

  try {
    const headers = await buildRequestHeaders(path, init);
    const response = await fetch(buildUrl(path), {
      ...init,
      headers,
      credentials: 'same-origin',
      signal: controller.signal,
    });

    if (!response.ok) {
      throw parseApiError(await response.text(), response.status, response.statusText);
    }

    return await response.text();
  } catch (err) {
    if (err instanceof DOMException && err.name === 'AbortError') {
      throw new TimeoutError();
    }
    if (err instanceof TypeError) {
      throw new NetworkError();
    }
    throw err;
  } finally {
    clearTimeout(timeout);
  }
}

// Request deduplication for concurrent identical GET requests.
const inflightRequests = new Map<string, Promise<unknown>>();

/**
 * deduplicatedGet collapses concurrent GETs to the same path into a
 * single in-flight fetch. The promise is removed from the dedupe map
 * when it settles, so subsequent calls go through normally.
 */
export function deduplicatedGet<T>(path: string): Promise<T> {
  const existing = inflightRequests.get(path);
  if (existing) return existing as Promise<T>;

  const promise = request<T>(path).finally(() => {
    inflightRequests.delete(path);
  });
  inflightRequests.set(path, promise);
  return promise;
}
