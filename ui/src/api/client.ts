import { deduplicatedGet, request, requestJson, requestText } from './requestCore';
import { requestJsonWithProgress } from './requestUpload';
import type {
  AlertConfig,
  CloneDeviceRequest,
  ConfigDocument,
  ConfigSchema,
  ConfigUpdateRequest,
  DebugLevelResponse,
  Device,
  DeviceBatchDeleteRequest,
  DeviceBatchDeleteResponse,
  DeviceDetailResponse,
  DeviceInterface,
  DeviceListResponse,
  DeviceMutationResponse,
  DeviceSummary,
  ErrorInjectionInfo,
  HistoryRecord,
  InterfacesResponse,
  ModelDescriptor,
  NeighborRecord,
  PcapAnalysisResult,
  PcapUploadRequest,
  PcapUploadResponse,
  ReplayRequest,
  ReplayState,
  RuntimeStatus,
  SegmentSummary,
  SimulationPreflightReport,
  SimulationPreflightRequest,
  SimulationRequest,
  SimulationStatus,
  StackStatsResponse,
  StandaloneCaptureRequest,
  StandaloneCaptureStatus,
  SynthesizeWalkRequest,
  SynthesizeWalkResponse,
  Template,
  TemplateContent,
  TopologyGraph,
  UpdateDebugLevelRequest,
  UseTemplateRequest,
  UseTemplateResponse,
  VersionInfo,
  WalkAnalyzeResponse,
  WalkBatchValidationResponse,
  WalkValidationResponse,
} from './types';

/**
 * Thin API endpoint wrappers. Each function below is just a typed
 * shorthand for one path + method, leaving auth, CSRF, retry, case
 * conversion, and request deduplication to requestCore.ts.
 *
 * Functions are grouped by domain (stats / config / replay / alerts
 * / files / templates / etc.); call sites import directly from this file.
 */

// =====================================================================
// Stats, devices, history, neighbors, config
// =====================================================================

export const fetchStats = () => deduplicatedGet<StackStatsResponse>('/api/v1/stats');
export const fetchDevices = () => deduplicatedGet<DeviceSummary[]>('/api/v1/devices');
// ADR 0008: multi-VLAN segments grouping the same devices /api/v1/devices
// reports flat. A flat (non-segmented) config still reports one untagged
// segment, so the shape is uniform.
export const fetchSegments = () => deduplicatedGet<SegmentSummary[]>('/api/v1/segments');
export const fetchHistory = () => deduplicatedGet<HistoryRecord[]>('/api/v1/history');
export const fetchNeighbors = () => deduplicatedGet<NeighborRecord[]>('/api/v1/neighbors');
export const fetchConfig = () => deduplicatedGet<ConfigDocument>('/api/v1/config');
export const updateConfig = (payload: ConfigUpdateRequest) =>
  requestJson<ConfigDocument>('/api/v1/config', payload, { method: 'PUT' });

// =====================================================================
// PCAP replay
// =====================================================================

export const fetchReplayStatus = () => deduplicatedGet<ReplayState>('/api/v1/replay');
export const startReplay = (payload: ReplayRequest) =>
  requestJson<ReplayState>('/api/v1/replay', payload, { method: 'POST' });
export const stopReplay = () => request<ReplayState>('/api/v1/replay', { method: 'DELETE' });

// =====================================================================
// Alerts
// =====================================================================

export const fetchAlerts = () => deduplicatedGet<AlertConfig>('/api/v1/alerts');
export const updateAlerts = (payload: AlertConfig) =>
  requestJson<AlertConfig>('/api/v1/alerts', payload, { method: 'PUT' });

// =====================================================================
// Files + SNMP walks
// =====================================================================

export const validateWalk = (filename: string) =>
  requestJson<WalkValidationResponse>('/api/v1/walk/validate', { filename }, { method: 'POST' });
export const fixWalk = (filename: string) =>
  requestJson<WalkValidationResponse>('/api/v1/walk/fix', { filename }, { method: 'POST' });

/**
 * Validate every walk file referenced by the running config in one call.
 * Used by the "Validate all" control on the walk validator page so operators
 * don't have to check each device's walk file one at a time.
 */
export const validateAllWalks = () =>
  requestJson<WalkBatchValidationResponse>('/api/v1/walk/validate-all', {}, { method: 'POST' });

/**
 * Parse a walk file into device identity, interface inventory, and
 * LLDP/CDP neighbors. Same filename resolution as validate/fix: a
 * library-relative name ("cisco/c3900.walk") or a legacy config-dir path.
 */
export const analyzeWalk = (filename: string) =>
  requestJson<WalkAnalyzeResponse>('/api/v1/walk/analyze', { filename }, { method: 'POST' });

// =====================================================================
// Config merge + import
// =====================================================================

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

// =====================================================================
// Version, topology, error injection
// =====================================================================

export const fetchVersion = () => deduplicatedGet<VersionInfo>('/api/v1/version');
export const fetchTopology = () => deduplicatedGet<TopologyGraph>('/api/v1/topology');

// Server-side topology export. The daemon renders the running topology as
// Graphviz DOT or GraphML (for Graphviz / yEd / gephi interop) — richer than
// the client-side JSON snapshot. Returns the raw document text.
export const exportTopology = (format: 'dot' | 'graphml') =>
  requestText(`/api/v1/topology/export?format=${format}`);
export const fetchErrorTypes = () => deduplicatedGet<ErrorInjectionInfo>('/api/v1/errors');

// Content library endpoint wrappers (walks/pcaps listing, revert/sanitize,
// and the unified networks store — #548, #897 L4) live in
// library-client.ts, imported directly by call sites (Biome's
// noBarrelFile/noReExportAll bans re-exporting them from here).

/**
 * Baseline walk synthesis (#546 p2). GET the (vendor, model) catalog to
 * populate the device editor's "Synthesize baseline walk" picker; POST
 * generates a walk for the given device from the chosen profile, writes
 * it into the library, preserves any prior original, and attaches it to
 * the device's walk_file server-side. See
 * docs/design/2026-05-baseline-walk-synthesis.md for the full contract.
 */
export const fetchSynthesizeWalkModels = () =>
  deduplicatedGet<ModelDescriptor[]>('/api/v1/synthesize-walk/models');

export const synthesizeWalk = (hostname: string, payload: SynthesizeWalkRequest) =>
  requestJson<SynthesizeWalkResponse>(
    `/api/v1/devices/${encodeURIComponent(hostname)}/synthesize-walk`,
    payload,
    { method: 'POST' },
  );

/**
 * Per-device interface list (#897 p5f). Feeds the Error-Injection panel's
 * interface picker so it's a dropdown scoped to the selected device rather
 * than free text that silently no-ops on a typo.
 */
export const fetchDeviceInterfaces = (hostname: string) =>
  deduplicatedGet<DeviceInterface[]>(`/api/v1/devices/${encodeURIComponent(hostname)}/interfaces`);

/**
 * Per-type device editor schema (#546 part 1). The daemon serves a
 * small static table describing which sections of the device editor
 * apply to each device type; the editor hides the rest. Falls back
 * to the "unknown" schema (show everything) on the daemon side, so
 * an unknown type never breaks the form.
 */
export interface DeviceEditorSchema {
  type: string;
  label: string;
  visibleSections: string[];
}
export const fetchDeviceEditorSchema = (deviceType: string) =>
  deduplicatedGet<DeviceEditorSchema>(`/api/v1/device-schemas/${encodeURIComponent(deviceType)}`);
export const fetchDeviceEditorSchemas = () =>
  deduplicatedGet<DeviceEditorSchema[]>('/api/v1/device-schemas');

export const injectError = (payload: {
  device: string;
  interface: string;
  errorType: string;
  value: number;
}) =>
  requestJson<{
    success: boolean;
    message: string;
    device: string;
    interface: string;
    errorType: string;
    value: number;
  }>('/api/v1/errors', payload, { method: 'POST' });

export const clearError = (device: string, iface: string, errorType: string) =>
  request<{
    success: boolean;
    message: string;
    device: string;
    interface: string;
    errorType: string;
  }>(
    `/api/v1/errors?device=${encodeURIComponent(device)}&interface=${encodeURIComponent(iface)}&errorType=${encodeURIComponent(errorType)}`,
    { method: 'DELETE' },
  );

export const clearAllErrors = () =>
  request<{ success: boolean; message: string }>('/api/v1/errors', {
    method: 'DELETE',
  });

// =====================================================================
// Interfaces, runtime + simulation lifecycle
// =====================================================================

export const fetchInterfaces = () => deduplicatedGet<InterfacesResponse>('/api/v1/interfaces');

// Fetch only usable interfaces (ethernet, WiFi, loopback)
export const fetchUsableInterfaces = () =>
  deduplicatedGet<InterfacesResponse>('/api/v1/interfaces?filter=usable');
export const fetchRuntimeStatus = () => deduplicatedGet<RuntimeStatus>('/api/v1/runtime');
export const fetchSimulationStatus = () => deduplicatedGet<SimulationStatus>('/api/v1/simulation');
export const preflightSimulation = (payload: SimulationPreflightRequest) =>
  requestJson<SimulationPreflightReport>('/api/v1/simulation/preflight', payload, {
    method: 'POST',
  });
export const startSimulation = (payload: SimulationRequest) =>
  requestJson<SimulationStatus>('/api/v1/simulation', payload, { method: 'POST' });
export const stopSimulation = () =>
  request<{ status: string }>('/api/v1/simulation', { method: 'DELETE' });

// =====================================================================
// Standalone packet capture (PCAP Inspector "sniff without a sim" mode)
// =====================================================================

export const fetchCaptureStatus = () => deduplicatedGet<StandaloneCaptureStatus>('/api/v1/capture');
export const startStandaloneCapture = (payload: StandaloneCaptureRequest) =>
  requestJson<StandaloneCaptureStatus>('/api/v1/capture', payload, {
    method: 'POST',
  });
export const stopStandaloneCapture = () =>
  request<{ status: string }>('/api/v1/capture', {
    method: 'DELETE',
  });

// =====================================================================
// Templates
// =====================================================================

export const fetchTemplates = () => deduplicatedGet<Template[]>('/api/v1/templates');

export const fetchTemplateContent = (name: string) =>
  request<TemplateContent>(`/api/v1/templates/${encodeURIComponent(name)}`);

export const applyTemplate = (payload: UseTemplateRequest) =>
  requestJson<UseTemplateResponse>('/api/v1/templates/use', payload, { method: 'POST' });

// =====================================================================
// Global debug level
// =====================================================================

export const fetchDebugLevel = () => deduplicatedGet<DebugLevelResponse>('/api/v1/debug/level');

export const updateDebugLevel = (payload: UpdateDebugLevelRequest) =>
  requestJson<DebugLevelResponse>('/api/v1/debug/level', payload, { method: 'PUT' });

// =====================================================================
// PCAP analyser
// =====================================================================

export const uploadPcap = (payload: PcapUploadRequest) =>
  requestJson<PcapUploadResponse>('/api/v1/pcap/upload', payload, { method: 'POST' });

/**
 * uploadPcapWithProgress is uploadPcap's progress-reporting sibling — used
 * by the uploader UI so it can render a determinate progress bar while the
 * (potentially large, base64-inflated) capture is in flight.
 */
export const uploadPcapWithProgress = (
  payload: PcapUploadRequest,
  onProgress: (percent: number) => void,
  signal?: AbortSignal,
) =>
  requestJsonWithProgress<PcapUploadResponse>('/api/v1/pcap/upload', payload, onProgress, signal);

export const fetchPcapAnalysis = (analysisId: string) =>
  request<PcapAnalysisResult>(`/api/v1/pcap/${encodeURIComponent(analysisId)}`);

// =====================================================================
// Device configuration (CRUD on saved devices)
// =====================================================================

export const fetchConfigDevices = () =>
  deduplicatedGet<DeviceListResponse>('/api/v1/config/devices');

export const fetchConfigDevice = (hostname: string) =>
  request<DeviceDetailResponse>(`/api/v1/config/devices/${encodeURIComponent(hostname)}`);

export const createDevice = (device: Device) =>
  requestJson<DeviceMutationResponse>('/api/v1/config/devices', device, { method: 'POST' });

export const updateDevice = (hostname: string, device: Partial<Device>) =>
  requestJson<DeviceMutationResponse>(
    `/api/v1/config/devices/${encodeURIComponent(hostname)}`,
    device,
    { method: 'PUT' },
  );

export const deleteDevice = (hostname: string) =>
  request<DeviceMutationResponse>(`/api/v1/config/devices/${encodeURIComponent(hostname)}`, {
    method: 'DELETE',
  });

export const deleteDevices = (hostnames: string[]) =>
  requestJson<DeviceBatchDeleteResponse>(
    '/api/v1/config/devices',
    { hostnames } satisfies DeviceBatchDeleteRequest,
    { method: 'DELETE' },
  );

export const cloneDevice = (hostname: string, payload: CloneDeviceRequest) =>
  requestJson<DeviceMutationResponse>(
    `/api/v1/config/devices/${encodeURIComponent(hostname)}/clone`,
    payload,
    { method: 'POST' },
  );

export const fetchConfigSchema = () => deduplicatedGet<ConfigSchema>('/api/v1/config/schema');
