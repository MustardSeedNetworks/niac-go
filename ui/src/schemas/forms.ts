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

// MAC + IPv4 regexes mirror the Go `validate:"mac"` / `validate:"ip"` tags on
// converter.Device (internal/converter/types.go). Kept here so the device
// editor can surface format errors inline before the round-trip to the server.
const MAC_REGEX = /^([0-9A-Fa-f]{2}[:-]){5}[0-9A-Fa-f]{2}$/;
const IPV4_REGEX = /^(25[0-5]|2[0-4]\d|1?\d?\d)(\.(25[0-5]|2[0-4]\d|1?\d?\d)){3}$/;

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
 * Used by ErrorInjectionPanel. The Go side validates the (device,
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

/**
 * Device editor: the two always-required identity fields plus the optional
 * primary IP. The device object carries many more (per-protocol) sections, but
 * only these need format-level validation before save — the Go side validates
 * the full structure. `safeParse` against this schema replaces the editor's old
 * presence-only checks (`if (!device.hostname.trim())`) and drives inline field
 * errors. Extra keys on the parsed object are ignored by `v.object`, so it can
 * be run against a whole `Device` without stripping the other sections (the
 * caller keeps using the untouched device for the actual save).
 */
export const DeviceFormSchema = v.object({
  hostname: v.pipe(
    v.string(),
    v.trim(),
    v.minLength(1, 'Hostname is required'),
    v.maxLength(253, 'Hostname is too long (max 253 chars)'),
    v.regex(
      HOSTNAME_REGEX,
      'Hostname must start with a letter and contain only alphanumeric, dots, hyphens, or underscores',
    ),
  ),
  mac: v.pipe(
    v.string(),
    v.trim(),
    v.minLength(1, 'MAC address is required'),
    v.regex(MAC_REGEX, 'MAC must be six hex octets, e.g. 00:1A:2B:3C:4D:5E'),
  ),
  ip: v.optional(
    v.pipe(
      v.string(),
      // Primary IP is optional; an empty string means "no management IP".
      v.check((s) => s === '' || IPV4_REGEX.test(s), 'Primary IP must be a valid IPv4 address'),
    ),
  ),
});

export type DeviceFormFields = v.InferOutput<typeof DeviceFormSchema>;
