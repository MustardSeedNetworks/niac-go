import type {
  StackStatsResponse,
  DeviceSummary,
  HistoryRecord,
  NeighborRecord,
  ConfigDocument,
  ConfigUpdateRequest,
  ReplayState,
  ReplayRequest,
  AlertConfig,
  FileEntry,
  VersionInfo,
  TopologyGraph,
  ErrorInjectionInfo,
  InterfacesResponse,
  RuntimeStatus,
  SimulationStatus,
  SimulationRequest,
} from './types';

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
  try {
    const headers = new Headers(init.headers);
    headers.set('Accept', 'application/json');
    if (API_TOKEN) {
      headers.set('Authorization', `Bearer ${API_TOKEN}`);
    }

    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 30000); // 30s timeout

    try {
      const response = await fetch(buildURL(path), {
        ...init,
        headers,
        credentials: 'same-origin',
        signal: controller.signal,
      });

      clearTimeout(timeout);

      if (!response.ok) {
        const text = await response.text();
        throw new Error(text || response.statusText);
      }

      return response.json() as Promise<T>;
    } finally {
      clearTimeout(timeout);
    }
  } catch (err) {
    // Handle different error types
    if (err instanceof TypeError) {
      throw new Error('Network error: Unable to reach the server');
    }
    if (err instanceof DOMException && err.name === 'AbortError') {
      throw new Error('Request timeout');
    }
    throw err;
  }
}

export const fetchStats = () => request<StackStatsResponse>('/api/v1/stats');
export const fetchDevices = () => request<DeviceSummary[]>('/api/v1/devices');
export const fetchHistory = () => request<HistoryRecord[]>('/api/v1/history');
export const fetchNeighbors = () => request<NeighborRecord[]>('/api/v1/neighbors');
export const fetchConfig = () => request<ConfigDocument>('/api/v1/config');
export const updateConfig = (payload: ConfigUpdateRequest) =>
  request<ConfigDocument>('/api/v1/config', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
export const fetchReplayStatus = () => request<ReplayState>('/api/v1/replay');
export const startReplay = (payload: ReplayRequest) =>
  request<ReplayState>('/api/v1/replay', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
export const stopReplay = () =>
  request<ReplayState>('/api/v1/replay', {
    method: 'DELETE',
  });
export const fetchAlerts = () => request<AlertConfig>('/api/v1/alerts');
export const updateAlerts = (payload: AlertConfig) =>
  request<AlertConfig>('/api/v1/alerts', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
export const fetchFiles = (kind: 'pcaps' | 'walks') =>
  request<FileEntry[]>(`/api/v1/files?kind=${kind}`);
export const fetchVersion = () => request<VersionInfo>('/api/v1/version');
export const fetchTopology = () => request<TopologyGraph>('/api/v1/topology');
export const fetchErrorTypes = () => request<ErrorInjectionInfo>('/api/v1/errors');

export const injectError = (payload: {
  device_ip: string;
  interface: string;
  error_type: string;
  value: number;
}) =>
  request<{ success: boolean; message: string; device_ip: string; interface: string; error_type: string; value: number }>('/api/v1/errors', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });

export const clearError = (deviceIP: string, iface: string) =>
  request<{ success: boolean; message: string; device_ip: string; interface: string }>(
    `/api/v1/errors?device_ip=${encodeURIComponent(deviceIP)}&interface=${encodeURIComponent(iface)}`,
    { method: 'DELETE' }
  );

export const clearAllErrors = () =>
  request<{ success: boolean; message: string }>('/api/v1/errors', {
    method: 'DELETE',
  });

export const fetchInterfaces = () => request<InterfacesResponse>('/api/v1/interfaces');
export const fetchRuntimeStatus = () => request<RuntimeStatus>('/api/v1/runtime');
export const fetchSimulationStatus = () => request<SimulationStatus>('/api/v1/simulation');
export const startSimulation = (payload: SimulationRequest) =>
  request<SimulationStatus>('/api/v1/simulation', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
export const stopSimulation = () =>
  request<{ status: string }>('/api/v1/simulation', {
    method: 'DELETE',
  });
