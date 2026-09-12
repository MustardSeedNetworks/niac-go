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

import type { FC } from 'react';
import type { AuthoredDeviceType, DeviceType } from '../api/device-config-types';
import { DEVICE_TYPES } from '../components/device-editor/generated/sections.generated';
import {
  AccessPointSymbol,
  FirewallSymbol,
  HostSymbol,
  IotSymbol,
  Layer3SwitchSymbol,
  PrinterSymbol,
  RouterSymbol,
  ServerSymbol,
  SwitchSymbol,
  UnknownSymbol,
  VoipPhoneSymbol,
  WorkstationSymbol,
} from '../ui/icons/deviceSymbols';

/** Shape shared by the vendored device symbols and by lucide-react icons. */
type IconComponent = FC<{ className?: string }>;

/**
 * Device type icons, one per type, shared by every surface.
 *
 * These are the vendored device symbols rather than lucide glyphs: lucide is a
 * general-purpose UI set, so at canvas size a switch and a server came out as
 * near-identical boxes. See ui/icons/NOTICE.md.
 */
export const deviceTypeIcons: Record<DeviceType, IconComponent> = {
  router: RouterSymbol,
  switch: SwitchSymbol,
  'layer3-switch': Layer3SwitchSymbol,
  ap: AccessPointSymbol,
  'access-point': AccessPointSymbol,
  firewall: FirewallSymbol,
  server: ServerSymbol,
  host: HostSymbol,
  workstation: WorkstationSymbol,
  iot: IotSymbol,
  printer: PrinterSymbol,
  'voip-phone': VoipPhoneSymbol,
  unknown: UnknownSymbol,
};

/**
 * Spellings that appear in authored YAML and in older configs, mapped to the
 * canonical type. Lookup is case-insensitive, so only distinct spellings
 * belong here — not capitalisation variants.
 */
const deviceTypeAliases: Record<string, AuthoredDeviceType> = {
  // `ap` is a schema value in its own right, not a misspelling — the schema
  // accepts both spellings for one device. Folding it here rather than letting
  // it pass through is what stops the UI offering two identical "Access Point"
  // filters for what an operator thinks of as one kind of device.
  ap: 'access-point',
  // The runtime's own LLDP/CDP capability switches accept these spellings
  // (internal/protocols/lldp.go, cdp.go), so the map does too.
  access_point: 'access-point',
  accesspoint: 'access-point',
  'wireless-ap': 'access-point',
  wireless_ap: 'access-point',
  phone: 'voip-phone',
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
  // Aliases resolve first: some of them, like `ap`, are themselves schema
  // values, so a membership check ahead of this would pass them through
  // unfolded.
  const lowered = raw.toLowerCase();
  const alias = deviceTypeAliases[lowered];
  if (alias !== undefined) {
    return alias;
  }
  return DEVICE_TYPES.includes(lowered) ? (lowered as AuthoredDeviceType) : 'unknown';
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
  'layer3-switch': 'green',
  ap: 'purple',
  'access-point': 'purple',
  firewall: 'red',
  server: 'yellow',
  host: 'gray',
  workstation: 'gray',
  iot: 'purple',
  printer: 'yellow',
  'voip-phone': 'blue',
  unknown: 'gray',
};

/**
 * Device type CSS custom property colors for topology view
 * Uses CSS variables defined in the theme
 */
export const topologyDeviceColors: Record<DeviceType, string> = {
  router: 'var(--color-device-router)',
  switch: 'var(--color-device-switch)',
  'layer3-switch': 'var(--color-device-layer3-switch)',
  ap: 'var(--color-device-ap)',
  'access-point': 'var(--color-device-ap)',
  firewall: 'var(--color-device-firewall)',
  server: 'var(--color-device-server)',
  host: 'var(--color-device-host)',
  workstation: 'var(--color-device-workstation)',
  iot: 'var(--color-device-iot)',
  printer: 'var(--color-device-printer)',
  'voip-phone': 'var(--color-device-voip-phone)',
  unknown: 'var(--color-device-unknown)',
};

/**
 * Device type options for select/dropdown components
 */
export const deviceTypeOptions: { value: DeviceType; label: string }[] = [
  { value: 'router', label: 'Router' },
  { value: 'switch', label: 'Switch' },
  { value: 'layer3-switch', label: 'Layer 3 Switch' },
  { value: 'access-point', label: 'Access Point' },
  { value: 'firewall', label: 'Firewall' },
  { value: 'server', label: 'Server' },
  { value: 'workstation', label: 'Workstation' },
  { value: 'iot', label: 'IoT Device' },
  { value: 'printer', label: 'Printer' },
  { value: 'voip-phone', label: 'VoIP Phone' },
  { value: 'unknown', label: 'Unknown' },
];

/** Get the icon for a device type as authored. */
export function getDeviceIcon(type: DeviceType | string): IconComponent {
  return deviceTypeIcons[normalizeDeviceType(type)];
}

/**
 * Get the topology icon for a device type.
 *
 * Kept as its own name because the topology surfaces call it, but it now
 * resolves through the same map as every other surface — the two maps
 * disagreeing is the defect this replaced.
 */
export function getTopologyDeviceIcon(type: string): IconComponent {
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
