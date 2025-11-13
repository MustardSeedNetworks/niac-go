import type { StackStatsResponse, DeviceSummary, HistoryRecord, NeighborRecord } from './types';

const API_BASE = import.meta.env.VITE_API_BASE ?? '';
const API_TOKEN = import.meta.env.VITE_API_TOKEN ?? '';

function buildURL(path: string) {
  if (path.startsWith('http')) {
    return path;
  }
  if (path.startsWith('/')) {
    return `${API_BASE}${path}`;
  }
  return `${API_BASE}/${path}`;
}

async function request<T>(path: string, init: RequestInit = {}) {
  const headers = new Headers(init.headers);
  headers.set('Accept', 'application/json');
  if (API_TOKEN) {
    headers.set('Authorization', `Bearer ${API_TOKEN}`);
  }

  const response = await fetch(buildURL(path), {
    ...init,
    headers,
    credentials: 'same-origin',
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(text || response.statusText);
  }

  return response.json() as Promise<T>;
}

export const fetchStats = () => request<StackStatsResponse>('/api/v1/stats');
export const fetchDevices = () => request<DeviceSummary[]>('/api/v1/devices');
export const fetchHistory = () => request<HistoryRecord[]>('/api/v1/history');
export const fetchNeighbors = () => request<NeighborRecord[]>('/api/v1/neighbors');
