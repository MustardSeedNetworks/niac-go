/**
 * Device Details Panel Component
 *
 * Displays detailed information about a selected device
 * in the topology visualization.
 */

import type { FC } from 'react';
import type { DeviceSummary } from '../../api/types';
import {
  topologyDeviceColors as deviceColors,
  topologyDeviceIcons as deviceIcons,
} from '../../constants/device-types';
import { Button } from '../../ui/Button';
import { Tag } from '../../ui/Tag';

interface DeviceDetailsPanelProps {
  device: DeviceSummary | null;
  onClose: () => void;
  onEdit?: (device: DeviceSummary) => void;
}

/**
 * DeviceDetailsPanel shows device name, type, IPs, protocols,
 * and provides actions to edit or close.
 */
export const DeviceDetailsPanel: FC<DeviceDetailsPanelProps> = ({ device, onClose, onEdit }) => {
  if (!device) {
    return null;
  }

  const deviceType = device.type?.toLowerCase() || 'unknown';
  const Icon = deviceIcons[deviceType] || deviceIcons.unknown;
  const color = deviceColors[deviceType] || deviceColors.unknown;

  return (
    <div className="absolute top-4 right-4 w-80 bg-bg-elevated/95 backdrop-blur-sm border border-white/10 rounded-xl p-4 shadow-2xl z-50">
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-center gap-3">
          <div
            className="p-2 rounded-lg"
            style={{
              backgroundColor: `color-mix(in srgb, ${color} 20%, transparent)`,
            }}
          >
            <Icon className="w-6 h-6" />
          </div>
          <div>
            <h3 className="font-semibold text-text-primary">{device.name}</h3>
            <p className="text-sm text-text-muted capitalize">{device.type}</p>
          </div>
        </div>
        <button
          type="button"
          onClick={onClose}
          className="text-text-muted hover:text-text-primary text-xl"
          title="Close the device details panel and return to the topology view"
          aria-label="Close device details"
        >
          &times;
        </button>
      </div>

      {device.ips && device.ips.length > 0 && (
        <div className="mb-3">
          <div className="text-xs text-text-muted uppercase tracking-wide mb-1">IP Addresses</div>
          <div className="space-y-1">
            {device.ips.map((ip) => (
              <code key={ip} className="block text-sm text-status-info font-mono">
                {ip}
              </code>
            ))}
          </div>
        </div>
      )}

      {device.protocols && device.protocols.length > 0 && (
        <div className="mb-4">
          <div className="text-xs text-text-muted uppercase tracking-wide mb-1">Protocols</div>
          <div className="flex flex-wrap gap-1">
            {device.protocols.map((proto) => (
              <Tag key={proto} colorScheme="purple" className="text-xs">
                {proto}
              </Tag>
            ))}
          </div>
        </div>
      )}

      <div className="flex gap-2">
        <Button size="sm" tone="violet" onClick={() => onEdit?.(device)} className="flex-1">
          Edit Device
        </Button>
        <Button size="sm" variant="outline" onClick={onClose}>
          Close
        </Button>
      </div>
    </div>
  );
};
