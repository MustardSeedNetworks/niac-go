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
  'w-full rounded-lg border border-surface-border bg-bg-base/60 pad-sm text-sm text-text-primary placeholder-gray-500 focus:border-brand-accent focus:outline-none';

export const monoInputClassName =
  'w-full rounded-lg border border-surface-border bg-bg-base/60 pad-sm text-sm text-text-primary placeholder-gray-500 focus:border-brand-accent focus:outline-none font-mono';

export const selectClassName =
  'w-full rounded-lg border border-surface-border bg-bg-base/60 pad-sm text-sm text-text-primary focus:border-brand-accent focus:outline-none';

export const checkboxClassName =
  'w-4 h-4 rounded border-border-muted bg-bg-elevated text-brand-primary focus:ring-brand-primary';

export const smallInputClassName =
  'flex-1 rounded-lg border border-surface-border bg-bg-base/60 pad-xs text-sm text-text-primary placeholder-gray-500 focus:border-brand-accent focus:outline-none font-mono';
