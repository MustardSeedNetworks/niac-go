import type { Device, FileEntry } from '../../api/types';

/**
 * Common props for all protocol section components
 */
export interface ProtocolSectionBaseProps {
  device: Device;
  isExpanded: boolean;
  onToggle: () => void;
}

/**
 * Props for protocol sections that can be enabled/disabled
 */
export interface ProtocolSectionProps extends ProtocolSectionBaseProps {
  onUpdate: <K extends keyof Device>(field: K, value: Device[K]) => void;
}

/**
 * Props for SNMP section (needs walk files)
 */
export interface SNMPSectionProps extends ProtocolSectionProps {
  walkFiles: FileEntry[] | null;
}

/**
 * Common input class names for device editor forms
 */
export const inputClassName =
  'w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none';

export const monoInputClassName =
  'w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none font-mono';

export const selectClassName =
  'w-full rounded-lg border border-white/10 bg-gray-950/60 p-3 text-sm text-white focus:border-violet-400 focus:outline-none';

export const checkboxClassName =
  'w-4 h-4 rounded border-gray-600 bg-gray-800 text-violet-600 focus:ring-violet-500';

export const smallInputClassName =
  'flex-1 rounded-lg border border-white/10 bg-gray-950/60 p-2 text-sm text-white placeholder-gray-500 focus:border-violet-400 focus:outline-none font-mono';
