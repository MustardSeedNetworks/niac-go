/**
 * Device Details Panel Component
 *
 * Displays detailed information about a selected device
 * in the topology visualization.
 */

import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
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
  const { t } = useTranslation('pages');
  const { t: tCommon } = useTranslation('common');
  if (!device) {
    return null;
  }

  const deviceType = device.type?.toLowerCase() || 'unknown';
  const Icon = deviceIcons[deviceType] || deviceIcons.unknown;
  const color = deviceColors[deviceType] || deviceColors.unknown;

  return (
    <div className="absolute top-4 right-4 w-80 bg-bg-elevated/95 backdrop-blur-sm border border-surface-border rounded-xl pad shadow-2xl z-50">
      <div className="flex items-start justify-between mb-content">
        <div className="flex items-center gap-default">
          <div
            className="pad-xs rounded-lg"
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
          title={t('topology.deviceDetails.closeTitle')}
          aria-label={t('topology.deviceDetails.closeAriaLabel')}
        >
          &times;
        </button>
      </div>

      {device.ips && device.ips.length > 0 && (
        <div className="mb-heading">
          <div className="text-xs text-text-muted uppercase tracking-wide mb-tight">
            {t('devices.ipAddressesHeader')}
          </div>
          <div className="stack-xs">
            {device.ips.map((ip) => (
              <code key={ip} className="block text-sm text-status-info font-mono">
                {ip}
              </code>
            ))}
          </div>
        </div>
      )}

      {device.protocols && device.protocols.length > 0 && (
        <div className="mb-content">
          <div className="text-xs text-text-muted uppercase tracking-wide mb-tight">
            {t('packets.stats.protocols')}
          </div>
          <div className="flex flex-wrap gap-tight">
            {device.protocols.map((proto) => (
              <Tag key={proto} colorScheme="purple" className="text-xs">
                {proto}
              </Tag>
            ))}
          </div>
        </div>
      )}

      <div className="flex gap-compact">
        <Button size="sm" tone="violet" onClick={() => onEdit?.(device)} className="flex-1">
          {t('deviceEditor.editLabel')}
        </Button>
        <Button size="sm" variant="outline" onClick={onClose}>
          {tCommon('buttons.close')}
        </Button>
      </div>
    </div>
  );
};
