import { request } from './requestCore';
import type { CaptureFilterResponse } from './types';

/** Get the current capture filter state. */
export function getCaptureFilter(): Promise<CaptureFilterResponse> {
  return request<CaptureFilterResponse>('/api/v1/capture/filter');
}

/** Set a BPF capture filter. */
export function setCaptureFilter(filter: string): Promise<CaptureFilterResponse> {
  return request<CaptureFilterResponse>('/api/v1/capture/filter', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ filter }),
  });
}

/** Clear the active capture filter. */
export function clearCaptureFilter(): Promise<CaptureFilterResponse> {
  return request<CaptureFilterResponse>('/api/v1/capture/filter', {
    method: 'DELETE',
  });
}
