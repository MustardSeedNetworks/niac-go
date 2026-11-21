// Library Network Types — mirrors internal/library.NetworkEntry /
// NetworkContent (internal/library/list.go). The library's networks/
// store is the single source of truth for user-saved YAML configs
// (#897 L4); see client.ts's "Library networks" section.

/**
 * Where a library network came from. `example` is a scenario shipped in the
 * daemon binary: read-only, never a file on disk, and duplicated into the user's
 * library to be edited. A file with the same name shadows it, so a name is
 * never listed twice.
 */
export type LibraryNetworkSource = 'bundle' | 'user' | 'example';

export interface LibraryNetwork {
  name: string;
  description?: string;
  useCase?: string;
  deviceCount: number;
  modifiedAt: string;
  sizeBytes: number;
  source: LibraryNetworkSource;
  valid: boolean;
  error?: string;
}

export interface LibraryNetworkContent {
  name: string;
  content: string;
  format: 'yaml' | 'json';
  source: LibraryNetworkSource;
}

export interface UploadLibraryNetworkRequest {
  name: string;
  content: string;
}

/** Saves a copy of a shipped example into the library (server-side). */
export interface DuplicateLibraryNetworkRequest {
  copyFrom: string;
  name?: string;
}

export interface UploadLibraryNetworkResponse {
  success: boolean;
  name: string;
}
