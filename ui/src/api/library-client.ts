import { deduplicatedGet, request, requestJson } from './requestCore';
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
