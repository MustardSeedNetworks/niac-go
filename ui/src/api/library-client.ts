import { deduplicatedGet, request, requestJson } from './requestCore';
import { requestJsonWithProgress } from './requestUpload';
import type {
  LibraryNetwork,
  LibraryNetworkContent,
  UploadLibraryNetworkRequest,
  UploadLibraryNetworkResponse,
} from './types';

/**
 * Content library endpoint wrappers (#548 walks/pcaps, #897 L4 networks).
 * Split out of client.ts to keep that file under the TS line cap as the
 * library surface grows; call sites still import from '../api/client',
 * which re-exports everything below.
 */

// Library file listings. Backed by GET /api/v1/library/{walks,pcaps},
// which return [{name, sizeBytes, modifiedAt, source}]. The picker
// integrations on the device editor (SNMP walks) and Packets / Traffic
// pages (PCAPs) consume these to surface library content without
// hitting the older /api/v1/files routes.
export type LibrarySource = 'starter' | 'bundle' | 'user';

export interface LibraryFileEntry {
  name: string;
  sizeBytes: number;
  modifiedAt: string;
  source: LibrarySource;
  edited: boolean;
}
export const fetchLibraryWalks = () => deduplicatedGet<LibraryFileEntry[]>('/api/v1/library/walks');
export const fetchLibraryPcaps = () => deduplicatedGet<LibraryFileEntry[]>('/api/v1/library/pcaps');

// =====================================================================
// Library networks — the single source of truth for user-saved YAML
// configs (#897 L4). Replaces the legacy /api/v1/configs surface: the
// daemon migrates any pre-existing $HOME/.niac/configs files into the
// library on first start (internal/api/configs_migrate.go), so the UI
// only ever talks to /api/v1/library/networks.
// =====================================================================

export const fetchLibraryNetworks = () =>
  deduplicatedGet<LibraryNetwork[]>('/api/v1/library/networks');

export const fetchLibraryNetworkContent = (name: string) =>
  request<LibraryNetworkContent>(`/api/v1/library/networks/${encodeURIComponent(name)}`);

export const uploadLibraryNetwork = (payload: UploadLibraryNetworkRequest) =>
  requestJson<UploadLibraryNetworkResponse>('/api/v1/library/networks', payload, {
    method: 'POST',
  });

export const deleteLibraryNetwork = (name: string) =>
  request<void>(`/api/v1/library/networks/${encodeURIComponent(name)}`, { method: 'DELETE' });

export interface ScenarioDraftEntry {
  name: string;
  revision: string;
  modifiedAt: string;
  sizeBytes: number;
}

export interface ScenarioDraft extends ScenarioDraftEntry {
  content: string;
  format: 'yaml';
}

export interface DraftTopologyEndpoint {
  device: string;
  interface: string;
}

export interface DraftTopologyLinkProperties {
  vlans: number[];
  nativeVlan: number;
  fdbOnly: boolean;
}

export type DraftTopologyMutation =
  | {
      operation: 'add_device';
      device: {
        name: string;
        type: string;
        vendor?: string;
        mac?: string;
        macSuffix?: number;
        sysObjectId?: string;
        profileRole?: string;
        ips?: string[];
        interfaces?: Array<{
          name: string;
          type: string;
          mtu: number;
          speed: number;
          duplex: 'full' | 'half';
          adminStatus: 'up' | 'down';
          operStatus: 'up' | 'down';
        }>;
        properties?: Record<string, string>;
      };
    }
  | {
      operation: 'connect' | 'update_link';
      link: {
        source: DraftTopologyEndpoint;
        target: DraftTopologyEndpoint;
        properties: DraftTopologyLinkProperties;
      };
    }
  | {
      operation: 'disconnect';
      link: { source: DraftTopologyEndpoint; target: DraftTopologyEndpoint };
    }
  | { operation: 'move_device'; position: { device: string; x: number; y: number } };

export const fetchScenarioDrafts = () =>
  deduplicatedGet<ScenarioDraftEntry[]>('/api/v1/library/drafts');

export const fetchScenarioDraft = (name: string) =>
  request<ScenarioDraft>(`/api/v1/library/drafts/${encodeURIComponent(name)}`);

export const createScenarioDraft = (name: string, content: string) =>
  requestJson<ScenarioDraft>('/api/v1/library/drafts', { name, content }, { method: 'POST' });

export const createScenarioDraftFromTemplate = (name: string, templateName: string) =>
  request<ScenarioDraft>('/api/v1/library/drafts', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, templateName }),
  });

export const replaceScenarioDraft = (name: string, revision: string, content: string) =>
  requestJson<ScenarioDraft>(
    `/api/v1/library/drafts/${encodeURIComponent(name)}`,
    { content },
    { method: 'PUT', headers: { 'If-Match': `"${revision}"` } },
  );

export const deleteScenarioDraft = (name: string, revision: string) =>
  request<void>(`/api/v1/library/drafts/${encodeURIComponent(name)}`, {
    method: 'DELETE',
    headers: { 'If-Match': `"${revision}"` },
  });

export const mutateScenarioDraftTopology = (
  name: string,
  revision: string,
  mutation: DraftTopologyMutation,
) =>
  requestJson<ScenarioDraft>(
    `/api/v1/library/drafts/${encodeURIComponent(name)}/topology`,
    mutation,
    { method: 'PATCH', headers: { 'If-Match': `"${revision}"` } },
  );

/**
 * Revert a library walk to its preserved pristine original, discarding
 * any edits made since the walk was first written. Backed by POST
 * /api/v1/library/walks/revert; the daemon 404s if the walk has no
 * preserved original to revert to (see library.PreserveOriginal
 * server-side for the preserve-once contract).
 */
export const revertWalk = (name: string) =>
  requestJson<LibraryFileEntry>('/api/v1/library/walks/revert', { name }, { method: 'POST' });

/**
 * Per-walk outcome reported by sanitizeWalk / sanitizeWalksBatch. Mirrors
 * the daemon's sanitizeWalkResult shape.
 */
export interface SanitizeWalkResult {
  name: string;
  success: boolean;
  error?: string;
  ipsTransformed?: number;
  hostnamesTransformed?: number;
}

export interface SanitizeWalkBatchResponse {
  results: SanitizeWalkResult[];
  sanitized: number;
  failed: number;
}

/**
 * Sanitize a library walk in place, replacing IPs/hostnames with
 * deterministic placeholders. Preserves the walk's pristine original
 * (idempotent — a no-op if already preserved) so it stays recoverable via
 * revertWalk. Backed by POST /api/v1/library/walks/sanitize.
 */
export const sanitizeWalk = (name: string) =>
  requestJson<LibraryFileEntry>('/api/v1/library/walks/sanitize', { name }, { method: 'POST' });

/**
 * Sanitize multiple library walks in one request. Backed by POST
 * /api/v1/library/walks/sanitize-batch; per-name failures don't fail the
 * whole request — see SanitizeWalkBatchResponse.results.
 */
export const sanitizeWalksBatch = (names: string[]) =>
  requestJson<SanitizeWalkBatchResponse>(
    '/api/v1/library/walks/sanitize-batch',
    { names },
    { method: 'POST' },
  );

// =====================================================================
// Content bundle install (#897 L3b) — the air-gapped/manual-install path.
// POST /api/v1/library/install extracts a gzip-tar bundle (the same format
// `niac content install --bundle` and the embedded/deb starter bundles
// use) over networks/walks/pcaps at once, so it's admin-scoped like
// config/import (see internal/api/routes.go).
// =====================================================================

export interface LibraryInstallRequest {
  filename: string;
  /** Base64-encoded gzip-tar content bundle. */
  data: string;
}

export interface LibraryInstallResponse {
  success: boolean;
  files: number;
  directories: number;
  bytes: number;
  perKind: Partial<Record<'networks' | 'walks' | 'pcaps', number>>;
  message: string;
}

export const installContentBundle = (payload: LibraryInstallRequest) =>
  requestJson<LibraryInstallResponse>('/api/v1/library/install', payload, { method: 'POST' });

/**
 * installContentBundleWithProgress is installContentBundle's
 * progress-reporting sibling — mirrors uploadPcapWithProgress so the
 * uploader UI can render a determinate progress bar while a (potentially
 * large, base64-inflated) bundle is in flight.
 */
export const installContentBundleWithProgress = (
  payload: LibraryInstallRequest,
  onProgress: (percent: number) => void,
  signal?: AbortSignal,
) =>
  requestJsonWithProgress<LibraryInstallResponse>(
    '/api/v1/library/install',
    payload,
    onProgress,
    signal,
  );
