/**
 * Template Types
 * Types for configuration templates
 */

export interface Template {
  /** Filename-derived identifier — pass this to /templates/{name} and /templates/use. */
  name: string;
  /** Optional human-readable label from the template's "# Display: ..." front-matter. */
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
   * Optional vendor key from the template's "# Vendor: ..." front-matter
   * (e.g. "cisco", "juniper"). When present, the Templates page groups
   * by vendor heading instead of generic type. Used by the vendor
   * template pack under cmd/niac/templates/vendor-templates/.
   */
  vendor?: string;
  tags?: string[];
  createdAt?: string;
  modifiedAt?: string;
}

export interface TemplateContent {
  name: string;
  content: string;
  format: 'yaml' | 'json';
}

export interface UseTemplateRequest {
  templateName: string;
  newConfigName?: string;
}

export interface UseTemplateResponse {
  success: boolean;
  configPath: string;
  message: string;
}

export interface UploadTemplateRequest {
  name: string;
  description: string;
  content: string;
  type?: Template['type'];
}

export interface UploadTemplateResponse {
  success: boolean;
  template: Template;
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
