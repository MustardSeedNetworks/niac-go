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
  UseTemplateRequest,
  UseTemplateResponse,
  VersionInfo,
} from "./types";

const API_BASE = import.meta.env.VITE_API_BASE ?? "";
const API_TOKEN = import.meta.env.VITE_API_TOKEN ?? "";

function buildUrl(path: string) {
  if (path.startsWith("http")) {
    return path;
  }
  if (path.startsWith("/")) {
    return `${API_BASE}${path}`;
  }
  return `${API_BASE}/${path}`;
}

async function request<T>(path: string, init: RequestInit = {}) {
  try {
    const headers = new Headers(init.headers);
    headers.set("Accept", "application/json");
    if (API_TOKEN) {
      headers.set("Authorization", `Bearer ${API_TOKEN}`);
    }

    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 30000); // 30s timeout

    try {
      const response = await fetch(buildUrl(path), {
        ...init,
        headers,
        credentials: "same-origin",
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
      throw new Error("Network error: Unable to reach the server");
    }
    if (err instanceof DOMException && err.name === "AbortError") {
      throw new Error("Request timeout");
    }
    throw err;
  }
}

export const fetchStats = () => request<StackStatsResponse>("/api/v1/stats");
export const fetchDevices = () => request<DeviceSummary[]>("/api/v1/devices");
export const fetchHistory = () => request<HistoryRecord[]>("/api/v1/history");
export const fetchNeighbors = () => request<NeighborRecord[]>("/api/v1/neighbors");
export const fetchConfig = () => request<ConfigDocument>("/api/v1/config");
export const updateConfig = (payload: ConfigUpdateRequest) =>
  request<ConfigDocument>("/api/v1/config", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
export const fetchReplayStatus = () => request<ReplayState>("/api/v1/replay");
export const startReplay = (payload: ReplayRequest) =>
  request<ReplayState>("/api/v1/replay", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
export const stopReplay = () =>
  request<ReplayState>("/api/v1/replay", {
    method: "DELETE",
  });
export const fetchAlerts = () => request<AlertConfig>("/api/v1/alerts");
export const updateAlerts = (payload: AlertConfig) =>
  request<AlertConfig>("/api/v1/alerts", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
export const fetchFiles = (kind: "pcaps" | "walks") =>
  request<FileEntry[]>(`/api/v1/files?kind=${kind}`);
export const fetchVersion = () => request<VersionInfo>("/api/v1/version");
export const fetchTopology = () => request<TopologyGraph>("/api/v1/topology");
export const fetchErrorTypes = () => request<ErrorInjectionInfo>("/api/v1/errors");

export const injectError = (payload: {
  device_ip: string;
  interface: string;
  error_type: string;
  value: number;
}) =>
  request<{
    success: boolean;
    message: string;
    device_ip: string;
    interface: string;
    error_type: string;
    value: number;
  }>("/api/v1/errors", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });

export const clearError = (deviceIp: string, iface: string) =>
  request<{ success: boolean; message: string; device_ip: string; interface: string }>(
    `/api/v1/errors?device_ip=${encodeURIComponent(deviceIp)}&interface=${encodeURIComponent(iface)}`,
    { method: "DELETE" },
  );

export const clearAllErrors = () =>
  request<{ success: boolean; message: string }>("/api/v1/errors", {
    method: "DELETE",
  });

export const fetchInterfaces = () => request<InterfacesResponse>("/api/v1/interfaces");
export const fetchRuntimeStatus = () => request<RuntimeStatus>("/api/v1/runtime");
export const fetchSimulationStatus = () => request<SimulationStatus>("/api/v1/simulation");
export const startSimulation = (payload: SimulationRequest) =>
  request<SimulationStatus>("/api/v1/simulation", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });
export const stopSimulation = () =>
  request<{ status: string }>("/api/v1/simulation", {
    method: "DELETE",
  });

// Template API functions
export const fetchTemplates = () => request<Template[]>("/api/v1/templates");

export const fetchTemplateContent = (name: string) =>
  request<TemplateContent>(`/api/v1/templates/${encodeURIComponent(name)}`);

export const applyTemplate = (payload: UseTemplateRequest) =>
  request<UseTemplateResponse>("/api/v1/templates/use", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });

export const uploadTemplate = (payload: UploadTemplateRequest) =>
  request<UploadTemplateResponse>("/api/v1/templates", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });

export const deleteTemplate = (name: string) =>
  request<{ success: boolean; message: string }>(`/api/v1/templates/${encodeURIComponent(name)}`, {
    method: "DELETE",
  });

// Protocol Debug Levels API functions
export const fetchProtocolDebugLevels = () =>
  request<ProtocolDebugLevelsResponse>("/api/v1/debug/levels");

export const updateProtocolDebugLevels = (payload: UpdateProtocolDebugLevelsRequest) =>
  request<ProtocolDebugLevelsResponse>("/api/v1/debug/levels", {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });

export const resetProtocolDebugLevels = () =>
  request<ResetProtocolDebugLevelsResponse>("/api/v1/debug/levels/reset", {
    method: "POST",
  });

// PCAP Analyzer API functions
// Note: These are stub implementations - backend endpoint needs to be implemented
export const uploadPcap = (payload: PcapUploadRequest) =>
  request<PcapUploadResponse>("/api/v1/pcap/upload", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
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
export const fetchConfigDevices = () => request<DeviceListResponse>("/api/v1/config/devices");

/**
 * Fetch a single device by hostname
 */
export const fetchConfigDevice = (hostname: string) =>
  request<DeviceDetailResponse>(`/api/v1/config/devices/${encodeURIComponent(hostname)}`);

/**
 * Create a new device
 */
export const createDevice = (device: Device) =>
  request<DeviceMutationResponse>("/api/v1/config/devices", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(device),
  });

/**
 * Update an existing device
 */
export const updateDevice = (hostname: string, device: Partial<Device>) =>
  request<DeviceMutationResponse>(`/api/v1/config/devices/${encodeURIComponent(hostname)}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(device),
  });

/**
 * Delete a device
 */
export const deleteDevice = (hostname: string) =>
  request<DeviceMutationResponse>(`/api/v1/config/devices/${encodeURIComponent(hostname)}`, {
    method: "DELETE",
  });

/**
 * Clone a device with a new hostname
 */
export const cloneDevice = (hostname: string, payload: CloneDeviceRequest) =>
  request<DeviceMutationResponse>(`/api/v1/config/devices/${encodeURIComponent(hostname)}/clone`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(payload),
  });

/**
 * Fetch the JSON Schema for device configuration
 * Used for dynamic form generation
 */
export const fetchConfigSchema = () => request<ConfigSchema>("/api/v1/config/schema");

/**
 * Fetch available walk files for SNMP configuration
 */
export const fetchWalkFiles = () => request<FileEntry[]>("/api/v1/files?kind=walks");
