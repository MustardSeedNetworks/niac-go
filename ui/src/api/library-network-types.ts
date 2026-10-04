// Library Network Types — mirrors internal/library.NetworkEntry /
// NetworkContent (internal/library/list.go). The library's networks/
// store is the single source of truth for user-saved YAML configs
// (#897 L4); see client.ts's "Library networks" section.

export interface LibraryNetwork {
  name: string;
  description?: string;
  useCase?: string;
  deviceCount: number;
  modifiedAt: string;
  sizeBytes: number;
  source: 'starter' | 'bundle' | 'user';
  valid: boolean;
  error?: string;
}

export interface LibraryNetworkContent {
  name: string;
  content: string;
  format: 'yaml' | 'json';
  source: 'starter' | 'bundle' | 'user';
}

export interface UploadLibraryNetworkRequest {
  name: string;
  content: string;
}

export interface UploadLibraryNetworkResponse {
  success: boolean;
  name: string;
}
