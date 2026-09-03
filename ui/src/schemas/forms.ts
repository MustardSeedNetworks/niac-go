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

// MAC + IP regexes mirror the Go `validate:"mac"` / `validate:"ip"` tags on
// converter.Device (internal/converter/types.go). Kept here so the device
// editor can surface format errors inline before the round-trip to the server.
const MAC_REGEX = /^([0-9A-Fa-f]{2}[:-]){5}[0-9A-Fa-f]{2}$/;
// `ips` accepts IPv6 too (the clinic scenario authors both), so an IPv4-only
// check would reject valid authoring. Loose on purpose: the daemon's
// `validate:"ip"` is the arbiter, this only catches a typo before the trip.
const IPV4_OCTET = String.raw`(25[0-5]|2[0-4]\d|1?\d?\d)`;
const IP_REGEX = new RegExp(
  `^(${IPV4_OCTET}(\\.${IPV4_OCTET}){3}|[0-9A-Fa-f]{0,4}(:[0-9A-Fa-f]{0,4}){2,7})$`,
);

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
    // SNMP ifName, not a Linux device name: the values NIAC simulates run to
    // "HundredGigabitEthernet0/0/1" (27 chars). A 15-char IFNAMSIZ-style cap
    // rejected names this form's own dropdown offers, and the server imposes no
    // length limit at all — only non-empty (#1476).
    v.maxLength(255, 'Interface name is too long (max 255 chars)'),
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
 * Device editor: the identity fields, validated before the round trip.
 *
 * The editor's model is the authored YAML document, so these are the daemon's
 * own key names. Everything else in the document is validated server-side by
 * the one validator (P1b-4) and reported with codes; only identity is worth
 * catching inline, because it is what the author types by hand.
 *
 * Identity is `mac` XOR `vendor`, which the daemon enforces as
 * ErrDeviceMACSourceConflict. `v.forward` attaches the cross-field issue to a
 * field: a bare `v.check` produces an issue with no path, which no input can
 * display.
 */
export const AuthoredDeviceSchema = v.pipe(
  v.object({
    // Absent and empty are the same unnamed device to an author, and only one
    // of them has a message worth showing.
    name: v.optional(
      v.pipe(
        v.string(),
        v.trim(),
        v.minLength(1, 'Name is required'),
        v.maxLength(253, 'Name is too long (max 253 chars)'),
        v.regex(
          HOSTNAME_REGEX,
          'Name must start with a letter and contain only alphanumeric, dots, hyphens, or underscores',
        ),
      ),
      '',
    ),
    mac: v.optional(
      v.pipe(
        v.string(),
        v.check(
          (s) => s === '' || MAC_REGEX.test(s),
          'MAC must be six hex octets, e.g. 00:1A:2B:3C:4D:5E',
        ),
      ),
    ),
    vendor: v.optional(v.string()),
    ips: v.optional(
      v.array(
        v.pipe(
          v.string(),
          v.check((s) => s === '' || IP_REGEX.test(s), 'Each address must be a valid IP address'),
        ),
      ),
    ),
  }),
  v.forward(
    v.check(
      ({ mac, vendor }) => Boolean(mac?.trim()) !== Boolean(vendor?.trim()),
      'A device is identified by a MAC address or by a vendor, not both and not neither',
    ),
    ['mac'],
  ),
);

export type AuthoredDeviceFormFields = v.InferOutput<typeof AuthoredDeviceSchema>;
