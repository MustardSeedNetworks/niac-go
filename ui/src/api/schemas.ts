/**
 * Zod Validation Schemas
 *
 * Provides runtime validation for API request payloads.
 * Schemas match the TypeScript interfaces in ./types.ts.
 *
 * FIX #186: No form validation library
 */

import { z } from 'zod';

// ============================================================================
// Device Schemas
// ============================================================================

const MAC_REGEX = /^([0-9A-Fa-f]{2}:){5}[0-9A-Fa-f]{2}$/;
const IP_REGEX = /^(\d{1,3}\.){3}\d{1,3}$/;
const HOSTNAME_REGEX = /^[a-zA-Z][a-zA-Z0-9._-]{0,252}$/;

export const deviceTypeSchema = z.enum([
  'router',
  'switch',
  'access_point',
  'firewall',
  'server',
  'workstation',
  'iot',
  'unknown',
]);

export const deviceSchema = z.object({
  hostname: z
    .string()
    .min(1, 'Hostname is required')
    .max(253, 'Hostname must be 253 characters or fewer')
    .regex(
      HOSTNAME_REGEX,
      'Hostname must start with a letter and contain only alphanumeric, dots, hyphens, or underscores',
    ),
  mac: z.string().regex(MAC_REGEX, 'MAC address must be in format XX:XX:XX:XX:XX:XX'),
  ip: z.string().regex(IP_REGEX, 'Invalid IP address format').optional(),
  type: deviceTypeSchema.optional(),
  vlan: z
    .number()
    .int()
    .min(1, 'VLAN must be between 1 and 4094')
    .max(4094, 'VLAN must be between 1 and 4094')
    .optional(),
  babble: z.boolean().optional(),
  mapToIp: z.string().regex(IP_REGEX, 'Invalid IP address format').optional(),
});

export const createDeviceSchema = z.object({
  device: deviceSchema,
});

// ============================================================================
// Replay Schemas
// ============================================================================

export const replayRequestSchema = z.object({
  file: z.string().min(1, 'File path is required'),
  loopMs: z.number().int().min(0, 'Loop interval must be non-negative').optional(),
  scale: z
    .number()
    .min(0.01, 'Scale must be greater than 0')
    .max(100, 'Scale must be 100 or less')
    .optional(),
  data: z.string().optional(),
});

// ============================================================================
// Config Schemas
// ============================================================================

export const configUpdateSchema = z.object({
  content: z.string().min(1, 'Configuration content is required'),
});

// ============================================================================
// Helper
// ============================================================================

export type ValidationResult<T> =
  | { success: true; data: T }
  | { success: false; errors: Array<{ path: string; message: string }> };

export function validate<T>(schema: z.ZodSchema<T>, data: unknown): ValidationResult<T> {
  const result = schema.safeParse(data);
  if (result.success) {
    return { success: true, data: result.data };
  }
  return {
    success: false,
    errors: result.error.issues.map((issue) => ({
      path: issue.path.join('.'),
      message: issue.message,
    })),
  };
}
