import { ApiError, NetworkError, TimeoutError } from './errors';
import type { CaptureFilterResponse } from './types';

const API_BASE = import.meta.env.VITE_API_BASE ?? '';
const API_TOKEN = import.meta.env.VITE_API_TOKEN ?? '';
const REQUEST_TIMEOUT_MS = 10_000;

function buildUrl(path: string): string {
  if (path.startsWith('/')) {
    return `${API_BASE}${path}`;
  }
  return `${API_BASE}/${path}`;
}

async function captureRequest<T>(path: string, init: RequestInit = {}): Promise<T> {
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), REQUEST_TIMEOUT_MS);

  try {
    const headers = new Headers(init.headers);
    headers.set('Accept', 'application/json');
    if (API_TOKEN) {
      headers.set('Authorization', `Bearer ${API_TOKEN}`);
    }

    const response = await fetch(buildUrl(path), {
      ...init,
      headers,
      credentials: 'same-origin',
      signal: controller.signal,
    });

    if (!response.ok) {
      const text = await response.text();
      throw new ApiError(text || response.statusText, response.status);
    }

    return (await response.json()) as T;
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

/** Get the current capture filter state. */
export function getCaptureFilter(): Promise<CaptureFilterResponse> {
  return captureRequest<CaptureFilterResponse>('/api/v1/capture/filter');
}

/** Set a BPF capture filter. */
export function setCaptureFilter(filter: string): Promise<CaptureFilterResponse> {
  return captureRequest<CaptureFilterResponse>('/api/v1/capture/filter', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ filter }),
  });
}

/** Clear the active capture filter. */
export function clearCaptureFilter(): Promise<CaptureFilterResponse> {
  return captureRequest<CaptureFilterResponse>('/api/v1/capture/filter', {
    method: 'DELETE',
  });
}
