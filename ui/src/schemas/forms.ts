/**
 * Zod schemas for niac's UI forms.
 *
 * niac stays on zod (vs valibot in seed/stem) — see seed#1199 for the
 * tracking issue on cross-repo consistency. Cloning the existing
 * src/api/schemas.ts regex constants here keeps the form-layer
 * separated from the API DTO layer.
 */
import { z } from 'zod';

// Reuse regex sources from the API schema layer.
const HOSTNAME_REGEX = /^[a-zA-Z][a-zA-Z0-9._-]{0,252}$/;

/**
 * Clone-device form: a single field for the new hostname. The Go side
 * also validates; this layer is for inline UI errors before submit.
 */
export const CloneDeviceSchema = z.object({
  newHostname: z
    .string()
    .trim()
    .min(1, 'Hostname is required')
    .max(253, 'Hostname is too long (max 253 chars)')
    .regex(
      HOSTNAME_REGEX,
      'Hostname must start with a letter and contain only alphanumeric, dots, hyphens, or underscores',
    ),
});

export type CloneDeviceFormFields = z.infer<typeof CloneDeviceSchema>;

/**
 * Upload-template modal: name + description + YAML content + category.
 * The Go side does YAML parsing; this layer just blocks empty
 * submissions and overly long names.
 */
export const UploadTemplateSchema = z.object({
  name: z
    .string()
    .trim()
    .min(1, 'Template name is required')
    .max(64, 'Name is too long (max 64 chars)'),
  description: z.string().max(256, 'Description is too long (max 256 chars)'),
  content: z.string().min(1, 'Template content is required'),
  type: z.enum(['basic', 'router', 'switch', 'access-point', 'server', 'complete', 'custom']),
});

export type UploadTemplateFormFields = z.infer<typeof UploadTemplateSchema>;

/**
 * Error injection form: device + interface + error type + percentage value.
 * Used by ErrorInjectionPanel. The Go side validates the (deviceIp,
 * interface, errorType) tuple against the simulation registry.
 */
export const ErrorInjectionSchema = z.object({
  selectedDevice: z.string().min(1, 'Device is required'),
  selectedInterface: z
    .string()
    .trim()
    .min(1, 'Interface is required')
    .max(15, 'Interface name is too long (max 15 chars)'),
  selectedErrorType: z.string().min(1, 'Error type is required'),
  errorValue: z
    .number()
    .int('Value must be an integer')
    .min(0, 'Value must be 0 or more')
    .max(100, 'Value must be 100 or less'),
});

export type ErrorInjectionFormFields = z.infer<typeof ErrorInjectionSchema>;
