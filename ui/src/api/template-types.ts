/**
 * Template Types
 * Types for configuration templates
 */

export interface Template {
  name: string;
  description: string;
  deviceCount: number;
  type: 'basic' | 'router' | 'switch' | 'access-point' | 'server' | 'complete' | 'custom';
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
