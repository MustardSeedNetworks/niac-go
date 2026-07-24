import { ApiError, NetworkError, TimeoutError } from './errors';
import {
  buildRequestHeaders,
  buildUrl,
  notifyIfAuthenticationFailed,
  parseApiError,
  toCamelCase,
  toSnakeCase,
} from './requestCore';

/**
 * requestJsonWithProgress is the XHR-based cousin of requestJson() (see
 * requestCore.ts), split into its own module to keep requestCore under the
 * file-size cap. Used only where upload progress needs to be observable
 * (e.g. the PCAP uploader) — fetch() has no cross-browser way to report
 * upload progress, so this goes through XMLHttpRequest instead, sharing
 * the same CSRF/auth header construction, snake_case/camelCase conversion,
 * and error-parsing as the rest of the client. Unlike request(), a single
 * attempt is made (no 5xx retry): retrying would silently restart the
 * transfer and reset the caller's progress bar.
 */

// Large captures over slow links need more headroom than a regular JSON
// request; the server enforces its own hard cap (MaxPCAPUploadBodySize),
// this is just how long the client waits for that transfer to complete.
const UPLOAD_TIMEOUT_MS = 5 * 60_000;

export function requestJsonWithProgress<T>(
  path: string,
  payload: unknown,
  onProgress: (percent: number) => void,
  signal?: AbortSignal,
): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    if (signal?.aborted) {
      reject(new DOMException('Request aborted', 'AbortError'));
      return;
    }

    buildRequestHeaders(path, { method: 'POST' })
      .then((headers) => {
        headers.set('Content-Type', 'application/json');

        const xhr = new XMLHttpRequest();
        xhr.open('POST', buildUrl(path));
        xhr.withCredentials = true;
        xhr.timeout = UPLOAD_TIMEOUT_MS;
        headers.forEach((value, key) => {
          xhr.setRequestHeader(key, value);
        });

        const onAbort = () => xhr.abort();
        signal?.addEventListener('abort', onAbort);
        const cleanup = () => signal?.removeEventListener('abort', onAbort);

        xhr.upload.onprogress = (event) => {
          if (event.lengthComputable) {
            onProgress(Math.round((event.loaded / event.total) * 100));
          }
        };

        xhr.onload = () => {
          cleanup();
          if (xhr.status >= 200 && xhr.status < 300) {
            try {
              resolve(toCamelCase(JSON.parse(xhr.responseText) as T));
            } catch {
              reject(new ApiError('Invalid JSON response', xhr.status));
            }
            return;
          }
          notifyIfAuthenticationFailed(xhr.status);
          reject(parseApiError(xhr.responseText, xhr.status, xhr.statusText));
        };

        xhr.onerror = () => {
          cleanup();
          reject(new NetworkError());
        };
        xhr.ontimeout = () => {
          cleanup();
          reject(new TimeoutError());
        };
        xhr.onabort = () => {
          cleanup();
          reject(new DOMException('Request aborted', 'AbortError'));
        };

        xhr.send(JSON.stringify(toSnakeCase(payload)));
      })
      .catch(reject);
  });
}
