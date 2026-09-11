/**
 * Shared device type constants
 *
 * One icon, one topology colour and one tag colour per device type, keyed on
 * the `DeviceType` union. There were previously two parallel families — one
 * union-keyed for the device list, one loose-string-keyed for the topology
 * canvas — which disagreed: an `access_point` was absent from the topology map
 * and drew the `unknown` glyph on the canvas while the legend, which passed
 * the hyphenated alias, drew a Wifi glyph for the same device (#2052).
 *
 * A device type reaches the UI as a free string authored in YAML, so aliases
 * are resolved on the way in by `normalizeDeviceType` rather than by giving
 * the maps extra keys — the maps stay exhaustive over the union, and the
 * exhaustiveness is checked.
 */

import { CircleHelp, Cpu, Monitor, Network, Router, Server, Shield, Wifi } from 'lucide-react';
import type { FC } from 'react';
import { DEVICE_TYPES, type DeviceType } from '../api/device-config-types';

// Icon type for lucide-react components
type LucideIcon = FC<{ className?: string }>;

/** Device type icons, one per type, shared by every surface. */
export const deviceTypeIcons: Record<DeviceType, LucideIcon> = {
  router: Router,
  switch: Network,
  access_point: Wifi,
  firewall: Shield,
  server: Server,
  workstation: Monitor,
  iot: Cpu,
  unknown: CircleHelp,
};

/**
 * Spellings that appear in authored YAML and in older configs, mapped to the
 * canonical type. Lookup is case-insensitive, so only distinct spellings
 * belong here — not capitalisation variants.
 */
const deviceTypeAliases: Record<string, DeviceType> = {
  ap: 'access_point',
  'access-point': 'access_point',
  accesspoint: 'access_point',
  host: 'workstation',
  pc: 'workstation',
  client: 'workstation',
};

/**
 * normalizeDeviceType resolves a device type as authored into one of the
 * canonical types. An unrecognised value becomes `unknown` — the UI draws a
 * question mark rather than guessing at a device it has no mapping for.
 */
export function normalizeDeviceType(raw: string | undefined | null): DeviceType {
  if (!raw) {
    return 'unknown';
  }
  const lowered = raw.toLowerCase();
  if ((DEVICE_TYPES as readonly string[]).includes(lowered)) {
    return lowered as DeviceType;
  }
  return deviceTypeAliases[lowered] ?? 'unknown';
}

/**
 * Tag color scheme type for UI components
 */
export type TagColorScheme = 'blue' | 'green' | 'purple' | 'yellow' | 'red' | 'gray';

/**
 * Device type colors for Tag components
 */
export const deviceTypeColors: Record<DeviceType, TagColorScheme> = {
  router: 'blue',
  switch: 'green',
  access_point: 'purple',
  firewall: 'red',
  server: 'yellow',
  workstation: 'gray',
  iot: 'purple',
  unknown: 'gray',
};

/**
 * Device type CSS custom property colors for topology view
 * Uses CSS variables defined in the theme
 */
export const topologyDeviceColors: Record<DeviceType, string> = {
  router: 'var(--color-device-router)',
  switch: 'var(--color-device-switch)',
  access_point: 'var(--color-device-ap)',
  firewall: 'var(--color-device-firewall)',
  server: 'var(--color-device-server)',
  workstation: 'var(--color-device-workstation)',
  iot: 'var(--color-device-iot)',
  unknown: 'var(--color-device-unknown)',
};

/**
 * Device type options for select/dropdown components
 */
export const deviceTypeOptions: { value: DeviceType; label: string }[] = [
  { value: 'router', label: 'Router' },
  { value: 'switch', label: 'Switch' },
  { value: 'access_point', label: 'Access Point' },
  { value: 'firewall', label: 'Firewall' },
  { value: 'server', label: 'Server' },
  { value: 'workstation', label: 'Workstation' },
  { value: 'iot', label: 'IoT Device' },
  { value: 'unknown', label: 'Unknown' },
];

/** Get the icon for a device type as authored. */
export function getDeviceIcon(type: DeviceType | string): LucideIcon {
  return deviceTypeIcons[normalizeDeviceType(type)];
}

/**
 * Get the topology icon for a device type.
 *
 * Kept as its own name because the topology surfaces call it, but it now
 * resolves through the same map as every other surface — the two maps
 * disagreeing is the defect this replaced.
 */
export function getTopologyDeviceIcon(type: string): LucideIcon {
  return deviceTypeIcons[normalizeDeviceType(type)];
}

/** Get the tag colour scheme for a device type as authored. */
export function getDeviceColor(type: DeviceType | string): TagColorScheme {
  return deviceTypeColors[normalizeDeviceType(type)];
}

/** Get the topology colour (a CSS variable) for a device type as authored. */
export function getTopologyDeviceColor(type: string): string {
  return topologyDeviceColors[normalizeDeviceType(type)];
}

/**
 * Get device label by type
 */
export function getDeviceLabel(type: DeviceType): string {
  const option = deviceTypeOptions.find((opt) => opt.value === type);
  return option?.label || 'Unknown';
}

/**
 * Tailwind CSS classes for device type colors
 * Used for icon backgrounds and text colors
 */
const deviceColorClasses: Record<TagColorScheme, { bg: string; text: string }> = {
  blue: { bg: 'bg-status-info/20', text: 'text-status-info' },
  green: { bg: 'bg-status-success/20', text: 'text-status-success' },
  purple: { bg: 'bg-brand-primary/20', text: 'text-brand-accent' },
  yellow: { bg: 'bg-status-warning/20', text: 'text-status-warning' },
  red: { bg: 'bg-status-error/20', text: 'text-status-error' },
  gray: { bg: 'bg-bg-muted/20', text: 'text-text-muted' },
};

/**
 * Get Tailwind CSS classes for device type colors
 * @param colorScheme - The color scheme (from deviceTypeColors)
 * @returns Object with bg and text class names
 */
export function getDeviceColorClasses(colorScheme: TagColorScheme): {
  bg: string;
  text: string;
} {
  return deviceColorClasses[colorScheme] || deviceColorClasses.gray;
}
