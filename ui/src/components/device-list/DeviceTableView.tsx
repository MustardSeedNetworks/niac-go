import { Copy, Edit3, Trash2 } from 'lucide-react';
import type { FC } from 'react';
import { useTranslation } from 'react-i18next';
import { deviceTypeColors, deviceTypeIcons } from '../../constants/device-types';
import { iconSizes } from '../../constants/sizes';
import { useDeviceList } from '../../contexts/DeviceListContext';
import { Card, CardContent } from '../../ui/Card';
import { Tag } from '../../ui/Tag';

export const DeviceTableView: FC = () => {
  const { t } = useTranslation('devices');
  const {
    devices,
    selectedDevices,
    onSelectDevice,
    onSelectAll,
    onEdit,
    onClone,
    onDelete,
    getDeviceProtocols,
  } = useDeviceList();
  return (
    <Card className="border-surface-border bg-bg-surface/70">
      <CardContent className="p-0">
        {/* Table header */}
        <div>
          <div className="flex items-center gap-comfortable border-b border-surface-border px-4 py-row-lg bg-bg-base/40">
            <input
              type="checkbox"
              checked={selectedDevices.size === devices.length && devices.length > 0}
              onChange={onSelectAll}
              className="h-4 w-4 rounded border-border-muted bg-bg-elevated text-brand-primary focus:ring-brand-primary"
              aria-label={t('list.selectAllAria')}
            />
            <div className="flex-1 grid grid-cols-12 gap-comfortable text-sm font-medium text-text-muted">
              <div className="col-span-3">{t('list.tableHeaderHostname')}</div>
              <div className="col-span-2">{t('list.tableHeaderType')}</div>
              <div className="col-span-2">{t('list.tableHeaderIpAddress')}</div>
              <div className="col-span-3">{t('list.tableHeaderProtocols')}</div>
              <div className="col-span-2 text-right">{t('list.tableHeaderActions')}</div>
            </div>
          </div>
        </div>

        {/* Device rows */}
        <div className="divide-y divide-knob/5">
          {devices.map((device) => {
            // Defensive: wild device types like "ap" / "access-point" aren't
            // in the DeviceType union; fall back to 'unknown' so the lookup
            // can't be undefined.
            const safeType =
              device.type && device.type in deviceTypeIcons ? device.type : 'unknown';
            const DeviceIcon = deviceTypeIcons[safeType];
            const typeColor = deviceTypeColors[safeType];
            const deviceProtocols = getDeviceProtocols(device);

            return (
              <div
                key={device.hostname}
                className={`flex items-center gap-comfortable px-4 py-row-lg hover:bg-surface-hover transition-colors ${
                  selectedDevices.has(device.hostname) ? 'bg-brand-primary/10' : ''
                }`}
              >
                <input
                  type="checkbox"
                  checked={selectedDevices.has(device.hostname)}
                  onChange={() => onSelectDevice(device.hostname)}
                  className="h-4 w-4 rounded border-border-muted bg-bg-elevated text-brand-primary focus:ring-brand-primary"
                  aria-label={t('list.selectDeviceAria', { hostname: device.hostname })}
                />
                <div className="flex-1 grid grid-cols-12 gap-comfortable items-center">
                  {/* Hostname */}
                  <div className="col-span-3 flex items-center gap-compact">
                    <DeviceIcon className={`${iconSizes.md} text-text-muted`} />
                    <button
                      type="button"
                      onClick={() => onEdit(device.hostname)}
                      className="text-text-primary hover:text-brand-accent font-medium truncate"
                      title={t('list.openEditorTitle', { hostname: device.hostname })}
                    >
                      {device.hostname}
                    </button>
                  </div>

                  {/* Type */}
                  <div className="col-span-2">
                    <Tag colorScheme={typeColor} className="text-xs capitalize">
                      {device.type?.replace('_', ' ') || t('list.unknownType')}
                    </Tag>
                  </div>

                  {/* IP */}
                  <div className="col-span-2">
                    <span className="text-text-secondary text-sm font-mono">
                      {device.ip || device.ips?.[0] || '—'}
                    </span>
                    {device.ips && device.ips.length > 1 && (
                      <span className="ml-tight text-text-muted text-xs">
                        +{device.ips.length - 1}
                      </span>
                    )}
                  </div>

                  {/* Protocols */}
                  <div className="col-span-3 flex flex-wrap gap-tight">
                    {deviceProtocols.length > 0 ? (
                      deviceProtocols.slice(0, 4).map((proto) => (
                        <Tag key={proto} colorScheme="gray" className="text-xs">
                          {proto}
                        </Tag>
                      ))
                    ) : (
                      <span className="text-text-muted text-sm">{t('list.noProtocols')}</span>
                    )}
                    {deviceProtocols.length > 4 && (
                      <Tag colorScheme="gray" className="text-xs">
                        +{deviceProtocols.length - 4}
                      </Tag>
                    )}
                  </div>

                  {/* Actions */}
                  <div className="col-span-2 flex justify-end gap-tight">
                    <button
                      type="button"
                      onClick={() => onEdit(device.hostname)}
                      className="pad-xs text-text-muted hover:text-brand-accent hover:bg-surface-hover rounded-lg transition-colors"
                      title={t('list.editDeviceTitle', { hostname: device.hostname })}
                      aria-label={t('list.editDeviceAria', { hostname: device.hostname })}
                    >
                      <Edit3 className={iconSizes.md} />
                    </button>
                    <button
                      type="button"
                      onClick={() => onClone(device.hostname)}
                      className="pad-xs text-text-muted hover:text-status-info hover:bg-surface-hover rounded-lg transition-colors"
                      title={t('list.cloneDeviceTitle', { hostname: device.hostname })}
                      aria-label={t('list.cloneDeviceAria', { hostname: device.hostname })}
                    >
                      <Copy className={iconSizes.md} />
                    </button>
                    <button
                      type="button"
                      onClick={() => onDelete(device.hostname)}
                      className="pad-xs text-text-muted hover:text-status-error hover:bg-surface-hover rounded-lg transition-colors"
                      title={t('list.deleteDeviceTitle', { hostname: device.hostname })}
                      aria-label={t('list.deleteDeviceAria', { hostname: device.hostname })}
                    >
                      <Trash2 className={iconSizes.md} />
                    </button>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </CardContent>
    </Card>
  );
};
