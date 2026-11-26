import { deviceTypeIcons, normalizeDeviceType } from '../constants/device-types';
import { useDeviceTypeLabels } from '../hooks/useDeviceTypeLabels';
import { CategoryChip } from '../shared/CategoryChip';

/**
 * A device's type as a neutral chip with its glyph. The type is a category,
 * not a verdict, so it never wears a status hue (2026-11-22 audit: a router
 * read blue, a firewall red, an L3 switch green).
 */
export function DeviceTypeChip({ type }: { type: string | undefined }) {
  const labels = useDeviceTypeLabels();
  const normalized = normalizeDeviceType(type);
  const Icon = deviceTypeIcons[normalized];
  return <CategoryChip label={labels[normalized]} icon={<Icon className="h-3.5 w-3.5" />} />;
}
