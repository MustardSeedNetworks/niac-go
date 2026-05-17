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

// User Config Types

export interface UserConfig {
  name: string;
  path: string;
  deviceCount: number;
  modifiedAt: string;
  size: number;
}

export interface UserConfigContent {
  name: string;
  content: string;
  format: 'yaml' | 'json';
}

export interface UserConfigsResponse {
  configs: UserConfig[];
}

export interface UploadUserConfigRequest {
  name: string;
  content: string;
}

export interface UploadUserConfigResponse {
  success: boolean;
  message: string;
  path: string;
}
