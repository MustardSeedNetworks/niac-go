import { parse as parseYaml, stringify as stringifyYaml } from 'yaml';
import type { AuthoredDevice } from '../components/device-editor/generated/authored-device.generated';

/**
 * The device editor's document model: the daemon's own authored YAML.
 *
 * The editor used to hold a camelCase `Device` projection and rebuild YAML from
 * it on save, which meant every authored field the projection had no property
 * for was dropped — 167 of them. Reading and writing the authored document
 * itself makes the round trip an identity instead of a mapping, so parity is a
 * property of the design rather than a list of fields somebody has to maintain.
 *
 * The daemon decodes with `KnownFields(true)`: an unknown key is a hard parse
 * error, not a field quietly ignored. Emitting exactly what was parsed, minus
 * the empties the author cleared, is what keeps that safe.
 */

/** Parse a `rawYaml` device document as served by GET /config/devices/{name}. */
export const parseAuthoredDevice = (rawYaml: string): AuthoredDevice => {
  const parsed: unknown = parseYaml(rawYaml);
  if (typeof parsed !== 'object' || parsed === null || Array.isArray(parsed)) {
    return {};
  }

  return parsed as AuthoredDevice;
};

/**
 * Drop the values an author has cleared.
 *
 * Editing a field to empty means "unset", and the generated inputs represent
 * that as `undefined` / `''`. Emitting those would either write a real empty
 * value or, for a cleared object, an explicit `null` the loader reads as
 * "present and defaulted" — a different device from the one on screen.
 * Booleans and `0` are values, so only `undefined`, `''` and emptied
 * containers are pruned.
 */
const prune = (value: unknown): unknown => {
  if (Array.isArray(value)) {
    const entries = value.map(prune).filter((entry) => entry !== undefined);
    return entries.length > 0 ? entries : undefined;
  }
  if (typeof value === 'object' && value !== null) {
    const entries = Object.entries(value)
      .map(([key, entry]) => [key, prune(entry)] as const)
      .filter(([, entry]) => entry !== undefined);
    return entries.length > 0 ? Object.fromEntries(entries) : undefined;
  }
  if (value === '' || value === undefined || value === null) return undefined;
  if (typeof value === 'number' && Number.isNaN(value)) return undefined;

  return value;
};

/** Serialize an authored device back to the `rawYaml` the daemon parses. */
export const serializeAuthoredDevice = (device: AuthoredDevice): string => {
  // lineWidth: 0 disables folding — a wrapped scalar is valid YAML but reads
  // as a mangled value in the preview pane and in a diff.
  return stringifyYaml(prune(device) ?? {}, { lineWidth: 0 }).trimEnd();
};
