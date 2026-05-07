import { ApiError, NetworkError, TimeoutError } from './errors';
import type {
  AlertConfig,
  CloneDeviceRequest,
  ConfigDocument,
  ConfigSchema,
  ConfigUpdateRequest,
  // Device Configuration Types
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
  UploadUserConfigRequest,
  UploadUserConfigResponse,
  UserConfigContent,
  UserConfigsResponse,
  UseTemplateRequest,
  UseTemplateResponse,
  VersionInfo,
  WalkValidationResponse,
} from './types';

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

async function buildRequestHeaders(path: string, init: RequestInit) {
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

// FIX #175: Check if an error is retryable (network errors and 5xx responses).
function isRetryableError(error: unknown): boolean {
  if (error instanceof TypeError) return true; // Network error
  if (error instanceof Response && error.status >= 500) return true;
  return false;
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
        throw new ApiError(text || response.statusText, response.status);
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

const requestJson = <T>(path: string, payload: unknown, init: RequestInit = {}) =>
  request<T>(path, {
    ...init,
    headers: { 'Content-Type': 'application/json', ...(init.headers ?? {}) },
    body: JSON.stringify(toSnakeCase(payload)),
  });

// Request deduplication for concurrent identical GET requests
const inflightRequests = new Map<string, Promise<unknown>>();

function deduplicatedGet<T>(path: string): Promise<T> {
  const existing = inflightRequests.get(path);
  if (existing) return existing as Promise<T>;

  const promise = request<T>(path).finally(() => {
    inflightRequests.delete(path);
  });
  inflightRequests.set(path, promise);
  return promise;
}

export const fetchStats = () => deduplicatedGet<StackStatsResponse>('/api/v1/stats');
export const fetchDevices = () => deduplicatedGet<DeviceSummary[]>('/api/v1/devices');
export const fetchHistory = () => deduplicatedGet<HistoryRecord[]>('/api/v1/history');
export const fetchNeighbors = () => deduplicatedGet<NeighborRecord[]>('/api/v1/neighbors');
export const fetchConfig = () => deduplicatedGet<ConfigDocument>('/api/v1/config');
export const updateConfig = (payload: ConfigUpdateRequest) =>
  requestJson<ConfigDocument>('/api/v1/config', payload, { method: 'PUT' });
export const fetchReplayStatus = () => deduplicatedGet<ReplayState>('/api/v1/replay');
export const startReplay = (payload: ReplayRequest) =>
  requestJson<ReplayState>('/api/v1/replay', payload, { method: 'POST' });
export const stopReplay = () =>
  request<ReplayState>('/api/v1/replay', {
    method: 'DELETE',
  });
export const fetchAlerts = () => deduplicatedGet<AlertConfig>('/api/v1/alerts');
export const updateAlerts = (payload: AlertConfig) =>
  requestJson<AlertConfig>('/api/v1/alerts', payload, { method: 'PUT' });
export const fetchFiles = (kind: 'pcaps' | 'walks') =>
  request<FileEntry[]>(`/api/v1/files?kind=${kind}`);
export const validateWalk = (filename: string) =>
  requestJson<WalkValidationResponse>('/api/v1/walk/validate', { filename }, { method: 'POST' });
export const fixWalk = (filename: string) =>
  requestJson<WalkValidationResponse>('/api/v1/walk/fix', { filename }, { method: 'POST' });

/**
 * Merge two YAML configs (overlay devices with same name as base devices
 * REPLACE the base entry; overlay-only devices are appended; base-only
 * devices kept). Mirrors `niac config merge` on the CLI.
 */
export const mergeConfigs = (payload: { base: string; overlay: string }) =>
  requestJson<{
    merged: string;
    baseDevices: number;
    overlayDevices: number;
    mergedDevices: number;
  }>('/api/v1/config/merge', payload, { method: 'POST' });

/**
 * Import a config in one of the supported formats and get back normalised
 * YAML. format="java-dsl" handles legacy `.cfg` files (equivalent to
 * `niac config export`); format="yaml" passes through with validation only.
 */
export const importConfig = (payload: { format: 'yaml' | 'java-dsl'; content: string }) =>
  requestJson<{ yaml: string; devices: number }>('/api/v1/config/import', payload, {
    method: 'POST',
  });
export const fetchVersion = () => deduplicatedGet<VersionInfo>('/api/v1/version');
export const fetchTopology = () => deduplicatedGet<TopologyGraph>('/api/v1/topology');
export const fetchErrorTypes = () => deduplicatedGet<ErrorInjectionInfo>('/api/v1/errors');

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

export const fetchInterfaces = () => deduplicatedGet<InterfacesResponse>('/api/v1/interfaces');

// Fetch only usable interfaces (ethernet, WiFi, loopback)
export const fetchUsableInterfaces = () =>
  deduplicatedGet<InterfacesResponse>('/api/v1/interfaces?filter=usable');
export const fetchRuntimeStatus = () => deduplicatedGet<RuntimeStatus>('/api/v1/runtime');
export const fetchSimulationStatus = () => deduplicatedGet<SimulationStatus>('/api/v1/simulation');
export const startSimulation = (payload: SimulationRequest) =>
  requestJson<SimulationStatus>('/api/v1/simulation', payload, {
    method: 'POST',
  });
export const stopSimulation = () =>
  request<{ status: string }>('/api/v1/simulation', {
    method: 'DELETE',
  });

// Template API functions
export const fetchTemplates = () => deduplicatedGet<Template[]>('/api/v1/templates');

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
  deduplicatedGet<ProtocolDebugLevelsResponse>('/api/v1/debug/levels');

export const updateProtocolDebugLevels = (payload: UpdateProtocolDebugLevelsRequest) =>
  requestJson<ProtocolDebugLevelsResponse>('/api/v1/debug/levels', payload, {
    method: 'PUT',
  });

export const resetProtocolDebugLevels = () =>
  request<ResetProtocolDebugLevelsResponse>('/api/v1/debug/levels/reset', {
    method: 'POST',
  });

// PCAP Analyzer API functions
// Note: These are stub implementations - backend endpoint needs to be implemented
export const uploadPcap = (payload: PcapUploadRequest) =>
  requestJson<PcapUploadResponse>('/api/v1/pcap/upload', payload, {
    method: 'POST',
  });

export const analyzePcap = (analysisId: string) =>
  request<PcapAnalysisResult>(`/api/v1/pcap/analyze/${encodeURIComponent(analysisId)}`);

export const fetchPcapAnalysis = (analysisId: string) =>
  request<PcapAnalysisResult>(`/api/v1/pcap/${encodeURIComponent(analysisId)}`);

// ============================================================================
// Device Configuration API functions
// ============================================================================

/**
 * Fetch all devices from the configuration
 */
export const fetchConfigDevices = () =>
  deduplicatedGet<DeviceListResponse>('/api/v1/config/devices');

/**
 * Fetch a single device by hostname
 */
export const fetchConfigDevice = (hostname: string) =>
  request<DeviceDetailResponse>(`/api/v1/config/devices/${encodeURIComponent(hostname)}`);

/**
 * Create a new device
 */
export const createDevice = (device: Device) =>
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
export const fetchConfigSchema = () => deduplicatedGet<ConfigSchema>('/api/v1/config/schema');

/**
 * Fetch available walk files for SNMP configuration
 */
export const fetchWalkFiles = () => request<FileEntry[]>('/api/v1/files?kind=walks');

// ============================================================================
// User Config API functions
// ============================================================================

/**
 * Fetch all user-uploaded configs
 */
export const fetchUserConfigs = () => deduplicatedGet<UserConfigsResponse>('/api/v1/configs');

/**
 * Fetch content of a specific user config
 */
export const fetchUserConfigContent = (name: string) =>
  request<UserConfigContent>(`/api/v1/configs/${encodeURIComponent(name)}`);

/**
 * Upload a new user config
 */
export const uploadUserConfig = (payload: UploadUserConfigRequest) =>
  requestJson<UploadUserConfigResponse>('/api/v1/configs', payload, {
    method: 'POST',
  });

/**
 * Delete a user config
 */
export const deleteUserConfig = (name: string) =>
  request<{ success: boolean; message: string }>(`/api/v1/configs/${encodeURIComponent(name)}`, {
    method: 'DELETE',
  });
