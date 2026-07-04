/**
 * Valibot schemas for niac's UI forms.
 *
 * Migrated from zod to valibot (#718) to match seed and stem — fleet
 * harmonization plus a smaller embedded bundle. The regex constants are
 * cloned here to keep the form layer separate from the API DTO layer.
 */
import * as v from 'valibot';

// Hostname source-of-truth regex (mirrors the Go-side validation).
const HOSTNAME_REGEX = /^[a-zA-Z][a-zA-Z0-9._-]{0,252}$/;

/**
 * Clone-device form: a single field for the new hostname. The Go side
 * also validates; this layer is for inline UI errors before submit.
 */
export const CloneDeviceSchema = v.object({
  newHostname: v.pipe(
    v.string(),
    v.trim(),
    v.minLength(1, 'Hostname is required'),
    v.maxLength(253, 'Hostname is too long (max 253 chars)'),
    v.regex(
      HOSTNAME_REGEX,
      'Hostname must start with a letter and contain only alphanumeric, dots, hyphens, or underscores',
    ),
  ),
});

export type CloneDeviceFormFields = v.InferOutput<typeof CloneDeviceSchema>;

/**
 * Upload-template modal: name + description + YAML content + category.
 * The Go side does YAML parsing; this layer just blocks empty
 * submissions and overly long names.
 */
export const UploadTemplateSchema = v.object({
  name: v.pipe(
    v.string(),
    v.trim(),
    v.minLength(1, 'Template name is required'),
    v.maxLength(64, 'Name is too long (max 64 chars)'),
  ),
  description: v.pipe(v.string(), v.maxLength(256, 'Description is too long (max 256 chars)')),
  content: v.pipe(v.string(), v.minLength(1, 'Template content is required')),
  type: v.picklist(['basic', 'router', 'switch', 'access-point', 'server', 'complete', 'custom']),
});

export type UploadTemplateFormFields = v.InferOutput<typeof UploadTemplateSchema>;

/**
 * Error injection form: device + interface + error type + percentage value.
 * Used by ErrorInjectionPanel. The Go side validates the (deviceIp,
 * interface, errorType) tuple against the simulation registry.
 */
export const ErrorInjectionSchema = v.object({
  selectedDevice: v.pipe(v.string(), v.minLength(1, 'Device is required')),
  selectedInterface: v.pipe(
    v.string(),
    v.trim(),
    v.minLength(1, 'Interface is required'),
    v.maxLength(15, 'Interface name is too long (max 15 chars)'),
  ),
  selectedErrorType: v.pipe(v.string(), v.minLength(1, 'Error type is required')),
  errorValue: v.pipe(
    v.number(),
    v.integer('Value must be an integer'),
    v.minValue(0, 'Value must be 0 or more'),
    v.maxValue(100, 'Value must be 100 or less'),
  ),
});

export type ErrorInjectionFormFields = v.InferOutput<typeof ErrorInjectionSchema>;
