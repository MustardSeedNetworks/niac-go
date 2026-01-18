/**
 * Shared device type constants
 *
 * Centralized definitions for device types, icons, colors, and labels
 * used across the application.
 */

import {
  Cpu,
  HardDrive,
  Laptop,
  Monitor,
  Network,
  Router,
  Server,
  Shield,
  Wifi,
} from 'lucide-react';
import type { FC } from 'react';
import type { DeviceType } from '../api/types';

// Icon type for lucide-react components
type LucideIcon = FC<{ className?: string }>;

/**
 * Device type icons mapping
 * Maps device types to their corresponding lucide-react icons
 */
export const deviceTypeIcons: Record<DeviceType, LucideIcon> = {
  router: Router,
  switch: Server,
  accessPoint: Wifi,
  firewall: Shield,
  server: HardDrive,
  workstation: Monitor,
  iot: Cpu,
  unknown: Server,
};

/**
 * Extended device icons for topology view
 * Includes additional aliases and network-specific mappings
 */
export const topologyDeviceIcons: Record<string, LucideIcon> = {
  router: Router,
  switch: Network,
  firewall: Shield,
  server: Server,
  workstation: Laptop,
  'access-point': Wifi,
  accessPoint: Wifi,
  ap: Wifi,
  iot: Cpu,
  storage: HardDrive,
  unknown: Network,
};

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
  accessPoint: 'purple',
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
export const topologyDeviceColors: Record<string, string> = {
  router: 'var(--color-device-router)',
  switch: 'var(--color-device-switch)',
  firewall: 'var(--color-device-firewall)',
  server: 'var(--color-device-server)',
  workstation: 'var(--color-device-workstation)',
  'access-point': 'var(--color-device-ap)',
  accessPoint: 'var(--color-device-ap)',
  ap: 'var(--color-device-ap)',
  iot: 'var(--color-device-iot)',
  storage: 'var(--color-device-storage)',
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

/**
 * Get device icon by type with fallback
 */
export function getDeviceIcon(type: DeviceType | string): LucideIcon {
  if (type in deviceTypeIcons) {
    return deviceTypeIcons[type as DeviceType];
  }
  if (type in topologyDeviceIcons) {
    return topologyDeviceIcons[type];
  }
  return Server;
}

/**
 * Get device color by type with fallback
 */
export function getDeviceColor(type: DeviceType | string): TagColorScheme {
  if (type in deviceTypeColors) {
    return deviceTypeColors[type as DeviceType];
  }
  return 'gray';
}

/**
 * Get topology device color (CSS variable) by type with fallback
 */
export function getTopologyDeviceColor(type: string): string {
  return topologyDeviceColors[type] || topologyDeviceColors.unknown;
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
  blue: { bg: 'bg-blue-500/20', text: 'text-blue-400' },
  green: { bg: 'bg-green-500/20', text: 'text-green-400' },
  purple: { bg: 'bg-purple-500/20', text: 'text-purple-400' },
  yellow: { bg: 'bg-yellow-500/20', text: 'text-yellow-400' },
  red: { bg: 'bg-red-500/20', text: 'text-red-400' },
  gray: { bg: 'bg-gray-500/20', text: 'text-gray-400' },
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
