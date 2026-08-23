import { ApiError, type ApiErrorDetail, NetworkError, TimeoutError } from './errors';

/**
 * Core HTTP request infrastructure for the API client. Everything that
 * is shared by every endpoint lives here:
 *
 *   - case conversion (toCamelCase in; nothing converts on the way out)
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
export const AUTH_FAILURE_EVENT = 'niac:authentication-required';

let runtimeAPIToken = '';

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

export function setRuntimeAPIToken(token: string) {
  runtimeAPIToken = token;
  csrfTokenPromise = null;
}

export function clearRuntimeAPIToken() {
  runtimeAPIToken = '';
  csrfTokenPromise = null;
}

function activeAPIToken() {
  return runtimeAPIToken || API_TOKEN;
}

export function notifyAuthenticationRequired() {
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new Event(AUTH_FAILURE_EVENT));
  }
}

export function notifyIfAuthenticationFailed(status: number) {
  if (status === 401) {
    notifyAuthenticationRequired();
  }
}

export interface RetryConfig {
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
    const token = activeAPIToken();
    if (token) {
      headers.set('Authorization', `Bearer ${token}`);
    }

    const response = await fetch(buildUrl(CSRF_TOKEN_PATH), {
      headers,
      credentials: 'same-origin',
    });
    if (!response.ok) {
      notifyIfAuthenticationFailed(response.status);
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
  const token = activeAPIToken();
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }
  if (isStateChangingMethod(init.method) && path !== CSRF_TOKEN_PATH) {
    headers.set('X-Csrf-Token', await fetchCSRFToken());
  }

  return headers;
}

export async function validateRuntimeAuthentication() {
  const path = '/api/v1/auth/scope';
  const headers = await buildRequestHeaders(path, { method: 'GET' });
  const response = await fetch(buildUrl(path), {
    headers,
    credentials: 'same-origin',
  });
  if (!response.ok) {
    throw parseApiError(await response.text(), response.status, response.statusText);
  }
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
  timeoutMs = REQUEST_TIMEOUT_MS,
) {
  const externalSignal = init.signal;
  let networkRetries = 0;
  let csrfRetries = 0;
  let retryDelay = 0;

  while (true) {
    if (retryDelay > 0) {
      await new Promise((resolve) => setTimeout(resolve, retryDelay));
      retryDelay = 0;
    }

    // Don't retry if the caller's signal was already aborted
    if (externalSignal?.aborted) {
      throw new DOMException('Request aborted', 'AbortError');
    }

    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), timeoutMs);

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
        if (isRetryableStatus(response.status) && networkRetries < retry.maxRetries) {
          retryDelay = retry.baseDelay * 2 ** networkRetries;
          networkRetries += 1;
          continue;
        }
        const text = await response.text();
        notifyIfAuthenticationFailed(response.status);
        // If the server rejected a stale CSRF token (typically because the
        // daemon was restarted and rotated its in-memory secret), drop the
        // cached token and retry once. Without this the SPA would keep
        // sending the stale token on every state-changing request until a
        // full reload.
        if (
          response.status === 403 &&
          isStateChangingMethod(init.method) &&
          /csrf_token_invalid/i.test(text) &&
          csrfRetries === 0
        ) {
          resetCSRFToken();
          csrfRetries += 1;
          continue;
        }
        throw parseApiError(text, response.status, response.statusText);
      }

      // 204 No Content (e.g. DELETE /api/v1/library/networks/{name}) has no
      // body — calling response.json() on it throws a SyntaxError.
      if (response.status === 204) {
        return undefined as T;
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
      if (isRetryableError(err) && networkRetries < retry.maxRetries) {
        retryDelay = retry.baseDelay * 2 ** networkRetries;
        networkRetries += 1;
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
}

/**
 * requestJson is the JSON-bodied cousin of request(). The payload is sent
 * exactly as given: per ADR-0007 the API speaks camelCase in both directions
 * with no exceptions, so there is nothing to convert on the way out.
 *
 * This used to snake_case the body, which was correct while the server still
 * had snake_case request tags. Those are gone, and the transform silently
 * broke every payload with a multi-word key — device create/clone,
 * error injection, PCAP replay, alert saving (D11). Responses are still
 * camel-cased on the way in by request(), for endpoints that predate the ADR.
 */
export const requestJson = <T>(
  path: string,
  payload: unknown,
  init: RequestInit = {},
  retry: RetryConfig = DEFAULT_RETRY,
  timeoutMs = REQUEST_TIMEOUT_MS,
) =>
  request<T>(
    path,
    {
      ...init,
      headers: { 'Content-Type': 'application/json', ...(init.headers ?? {}) },
      body: JSON.stringify(payload),
    },
    retry,
    timeoutMs,
  );

// requestJsonCamelCase is retained as an alias so the many call sites migrated
// during the ADR-0007 rollout keep working. It is now identical to requestJson;
// prefer requestJson for new code.
export const requestJsonCamelCase = <T>(
  path: string,
  payload: unknown,
  init: RequestInit = {},
  retry: RetryConfig = DEFAULT_RETRY,
  timeoutMs = REQUEST_TIMEOUT_MS,
) =>
  request<T>(
    path,
    {
      ...init,
      headers: { 'Content-Type': 'application/json', ...(init.headers ?? {}) },
      body: JSON.stringify(payload),
    },
    retry,
    timeoutMs,
  );

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
      if (response.status === 401) {
        notifyAuthenticationRequired();
      }
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
