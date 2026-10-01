/**
 * Built-in scenario types
 */

export interface BuiltinScenario {
  /** Filename-derived identifier — pass this to /scenario/builtins/{name} and /scenario/builtins/copy. */
  name: string;
  /** Optional human-readable label from the scenario's "# Display: ..." front-matter. */
  displayName?: string;
  description: string;
  deviceCount: number;
  type:
    | 'basic'
    | 'router'
    | 'switch'
    | 'access-point'
    | 'server'
    | 'firewall'
    | 'complete'
    | 'custom';
  /**
   * Optional vendor key from the scenario's "# Vendor: ..." front-matter
   * (e.g. "cisco", "juniper"). When present, the picker groups
   * by vendor heading instead of generic type, as the vendor scenarios do.
   */
  vendor?: string;
  tags?: string[];
  createdAt?: string;
  modifiedAt?: string;
}

export interface BuiltinScenarioContent {
  name: string;
  content: string;
  format: 'yaml' | 'json';
}

export interface CopyBuiltinScenarioRequest {
  scenarioName: string;
  newConfigName?: string;
}

export interface CopyBuiltinScenarioResponse {
  success: boolean;
  configPath: string;
  message: string;
}

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
