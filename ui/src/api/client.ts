import type {
  AlertConfig,
  CloneDeviceRequest,
  ConfigDocument,
  ConfigSchema,
  ConfigUpdateRequest,
  // Device Configuration Types
  CreateDeviceRequest,
  Device,
  DeviceDetailResponse,
  DeviceListResponse,
  DeviceMutationResponse,
  DeviceSummary,
  ErrorInjectionInfo,
  FileEntry,
  HistoryRecord,
  InterfacesResponse,
  NeighborRecord,
  PcapAnalysisResult,
  PcapUploadRequest,
  PcapUploadResponse,
  ProtocolDebugLevelsResponse,
  ReplayRequest,
  ReplayState,
  ResetProtocolDebugLevelsResponse,
  RuntimeStatus,
  SimulationRequest,
  SimulationStatus,
  StackStatsResponse,
  Template,
  TemplateContent,
  TopologyGraph,
  UpdateProtocolDebugLevelsRequest,
  UploadTemplateRequest,
  UploadTemplateResponse,
  UseTemplateRequest,
  UseTemplateResponse,
  VersionInfo,
} from './types';

const API_BASE = import.meta.env.VITE_API_BASE ?? '';
const API_TOKEN = import.meta.env.VITE_API_TOKEN ?? '';

// FIX #262: CSRF token management - fetch and cache the token
let csrfToken = '';

async function fetchCsrfToken(): Promise<string> {
  if (csrfToken) return csrfToken;
  try {
    const headers = new Headers();
    headers.set('Accept', 'application/json');
    if (API_TOKEN) {
      headers.set('Authorization', `Bearer ${API_TOKEN}`);
    }
    const response = await fetch(buildUrl('/api/v1/csrf-token'), { headers });
    if (response.ok) {
      const data = (await response.json()) as { token: string };
      csrfToken = data.token;
    }
  } catch {
    // CSRF token fetch failed - state-changing requests will be rejected by server
  }
  return csrfToken;
}

// FIX #270: Export API_BASE for EventSource URL consistency
export function getApiBase(): string {
  return API_BASE;
}

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

const toCamelCase = <T>(value: T): T => {
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

const toSnakeCase = <T>(value: T): T => {
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

function buildUrl(path: string) {
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

    // FIX #262: Include CSRF token for state-changing requests
    const method = (init.method ?? 'GET').toUpperCase();
    if (method === 'POST' || method === 'PUT' || method === 'PATCH' || method === 'DELETE') {
      const token = await fetchCsrfToken();
      if (token) {
        headers.set('X-Csrf-Token', token);
      }
    }

    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 30000); // 30s timeout

    try {
      const response = await fetch(buildUrl(path), {
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

      const data = (await response.json()) as T;
      return toCamelCase(data);
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

const requestJson = <T>(path: string, payload: unknown, init: RequestInit = {}) =>
  request<T>(path, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...(init.headers ?? {}) },
    body: JSON.stringify(toSnakeCase(payload)),
  });

export const fetchStats = () => request<StackStatsResponse>('/api/v1/stats');
export const fetchDevices = () => request<DeviceSummary[]>('/api/v1/devices');
export const fetchHistory = () => request<HistoryRecord[]>('/api/v1/history');
export const fetchNeighbors = () => request<NeighborRecord[]>('/api/v1/neighbors');
export const fetchConfig = () => request<ConfigDocument>('/api/v1/config');
export const updateConfig = (payload: ConfigUpdateRequest) =>
  requestJson<ConfigDocument>('/api/v1/config', payload, { method: 'PUT' });
export const fetchReplayStatus = () => request<ReplayState>('/api/v1/replay');
export const startReplay = (payload: ReplayRequest) =>
  requestJson<ReplayState>('/api/v1/replay', payload, { method: 'POST' });
export const stopReplay = () =>
  request<ReplayState>('/api/v1/replay', {
    method: 'DELETE',
  });
export const fetchAlerts = () => request<AlertConfig>('/api/v1/alerts');
export const updateAlerts = (payload: AlertConfig) =>
  requestJson<AlertConfig>('/api/v1/alerts', payload, { method: 'PUT' });
export const fetchFiles = (kind: 'pcaps' | 'walks') =>
  request<FileEntry[]>(`/api/v1/files?kind=${kind}`);
export const fetchVersion = () => request<VersionInfo>('/api/v1/version');
export const fetchTopology = () => request<TopologyGraph>('/api/v1/topology');
export const fetchErrorTypes = () => request<ErrorInjectionInfo>('/api/v1/errors');

export const injectError = (payload: {
  deviceIp: string;
  interface: string;
  errorType: string;
  value: number;
}) =>
  requestJson<{
    success: boolean;
    message: string;
    deviceIp: string;
    interface: string;
    errorType: string;
    value: number;
  }>('/api/v1/errors', payload, { method: 'POST' });

export const clearError = (deviceIp: string, iface: string) =>
  request<{
    success: boolean;
    message: string;
    deviceIp: string;
    interface: string;
  }>(
    `/api/v1/errors?device_ip=${encodeURIComponent(deviceIp)}&interface=${encodeURIComponent(iface)}`,
    { method: 'DELETE' },
  );

export const clearAllErrors = () =>
  request<{ success: boolean; message: string }>('/api/v1/errors', {
    method: 'DELETE',
  });

export const fetchInterfaces = () => request<InterfacesResponse>('/api/v1/interfaces');
export const fetchRuntimeStatus = () => request<RuntimeStatus>('/api/v1/runtime');
export const fetchSimulationStatus = () => request<SimulationStatus>('/api/v1/simulation');
export const startSimulation = (payload: SimulationRequest) =>
  requestJson<SimulationStatus>('/api/v1/simulation', payload, {
    method: 'POST',
  });
export const stopSimulation = () =>
  request<{ status: string }>('/api/v1/simulation', {
    method: 'DELETE',
  });

// Template API functions
export const fetchTemplates = () => request<Template[]>('/api/v1/templates');

export const fetchTemplateContent = (name: string) =>
  request<TemplateContent>(`/api/v1/templates/${encodeURIComponent(name)}`);

export const applyTemplate = (payload: UseTemplateRequest) =>
  requestJson<UseTemplateResponse>('/api/v1/templates/use', payload, {
    method: 'POST',
  });

export const uploadTemplate = (payload: UploadTemplateRequest) =>
  requestJson<UploadTemplateResponse>('/api/v1/templates', payload, {
    method: 'POST',
  });

export const deleteTemplate = (name: string) =>
  request<{ success: boolean; message: string }>(`/api/v1/templates/${encodeURIComponent(name)}`, {
    method: 'DELETE',
  });

// Protocol Debug Levels API functions
export const fetchProtocolDebugLevels = () =>
  request<ProtocolDebugLevelsResponse>('/api/v1/debug/levels');

export const updateProtocolDebugLevels = (payload: UpdateProtocolDebugLevelsRequest) =>
  requestJson<ProtocolDebugLevelsResponse>('/api/v1/debug/levels', payload, {
    method: 'PUT',
  });

export const resetProtocolDebugLevels = () =>
  request<ResetProtocolDebugLevelsResponse>('/api/v1/debug/levels/reset', {
    method: 'POST',
  });

// PCAP Analyzer API functions
export const uploadPcap = (payload: PcapUploadRequest) =>
  requestJson<PcapUploadResponse>('/api/v1/pcap/upload', payload, {
    method: 'POST',
  });

// FIX #265: Corrected URL to match backend route /api/v1/pcap/{id}
export const analyzePcap = (analysisId: string) =>
  request<PcapAnalysisResult>(`/api/v1/pcap/${encodeURIComponent(analysisId)}`);

export const fetchPcapAnalysis = (analysisId: string) =>
  request<PcapAnalysisResult>(`/api/v1/pcap/${encodeURIComponent(analysisId)}`);

// ============================================================================
// Device Configuration API functions
// ============================================================================

/**
 * Fetch all devices from the configuration
 */
export const fetchConfigDevices = () => request<DeviceListResponse>('/api/v1/config/devices');

/**
 * Fetch a single device by hostname
 */
export const fetchConfigDevice = (hostname: string) =>
  request<DeviceDetailResponse>(`/api/v1/config/devices/${encodeURIComponent(hostname)}`);

/**
 * Create a new device
 * FIX #287: Accept CreateDeviceRequest matching backend DeviceCreateRequest
 */
export const createDevice = (device: CreateDeviceRequest) =>
  requestJson<DeviceMutationResponse>('/api/v1/config/devices', device, {
    method: 'POST',
  });

/**
 * Update an existing device
 */
export const updateDevice = (hostname: string, device: Partial<Device>) =>
  requestJson<DeviceMutationResponse>(
    `/api/v1/config/devices/${encodeURIComponent(hostname)}`,
    device,
    {
      method: 'PUT',
    },
  );

/**
 * Delete a device
 */
export const deleteDevice = (hostname: string) =>
  request<DeviceMutationResponse>(`/api/v1/config/devices/${encodeURIComponent(hostname)}`, {
    method: 'DELETE',
  });

/**
 * Clone a device with a new hostname
 */
export const cloneDevice = (hostname: string, payload: CloneDeviceRequest) =>
  requestJson<DeviceMutationResponse>(
    `/api/v1/config/devices/${encodeURIComponent(hostname)}/clone`,
    payload,
    {
      method: 'POST',
    },
  );

/**
 * Fetch the JSON Schema for device configuration
 * Used for dynamic form generation
 */
export const fetchConfigSchema = () => request<ConfigSchema>('/api/v1/config/schema');

/**
 * Fetch available walk files for SNMP configuration
 */
export const fetchWalkFiles = () => request<FileEntry[]>('/api/v1/files?kind=walks');
