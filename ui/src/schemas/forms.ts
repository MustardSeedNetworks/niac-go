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
