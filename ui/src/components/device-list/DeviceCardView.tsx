import { Copy, Edit3, Network, Trash2 } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { Device } from '../../api/types';
import {
  deviceTypeColors,
  deviceTypeIcons,
  getDeviceColorClasses,
  normalizeDeviceType,
} from '../../constants/device-types';
import { iconSizes } from '../../constants/sizes';
import { useDeviceList } from '../../contexts/DeviceListContext';
import { ActionButton } from '../../ui/ActionButton';
import { Card, CardContent } from '../../ui/Card';
import { Tag } from '../../ui/Tag';
import { VirtualList } from './VirtualList';

/** A card plus its row gap; windowing needs every row of cards to be this tall. */
const CARD_ROW_HEIGHT_PX = 248;

/** Tailwind's `sm`, `lg` and `xl` widths: one more column at each. */
const COLUMN_BREAKPOINTS_PX = [640, 1024, 1280] as const;

function columnsForViewport(): number {
  return (
    1 +
    COLUMN_BREAKPOINTS_PX.filter((px) => window.matchMedia(`(min-width: ${px}px)`).matches).length
  );
}

function useCardColumns(): number {
  const [columns, setColumns] = useState(columnsForViewport);
  useEffect(() => {
    const lists = COLUMN_BREAKPOINTS_PX.map((px) => window.matchMedia(`(min-width: ${px}px)`));
    const update = () => setColumns(columnsForViewport());
    for (const list of lists) list.addEventListener('change', update);
    return () => {
      for (const list of lists) list.removeEventListener('change', update);
    };
  }, []);
  return columns;
}

function chunk<T>(items: T[], size: number): T[][] {
  const rows: T[][] = [];
  for (let i = 0; i < items.length; i += size) rows.push(items.slice(i, i + size));
  return rows;
}

export function DeviceCardView() {
  const { t } = useTranslation('devices');
  const {
    devices,
    selectedDevices,
    onSelectDevice,
    onEdit,
    onClone,
    onDelete,
    getDeviceProtocols,
  } = useDeviceList();
  const columns = useCardColumns();
  const rows = useMemo(() => chunk(devices, columns), [devices, columns]);

  const renderCard = (device: Device) => {
    const safeType = normalizeDeviceType(device.type);
    const DeviceIcon = deviceTypeIcons[safeType];
    const typeColor = deviceTypeColors[safeType];
    const colorClasses = getDeviceColorClasses(typeColor);
    const deviceProtocols = getDeviceProtocols(device);

    return (
      <div key={device.hostname} data-testid="device-list-card" className="h-full rounded-2xl">
        <Card
          className={`group border-surface-border bg-bg-surface/70 hover:border-brand-primary/30 transition-all h-full ${
            selectedDevices.has(device.hostname)
              ? 'ring-2 ring-brand-primary/50 border-brand-primary/30'
              : ''
          }`}
        >
          <CardContent className="pad stack">
            {/* Header with checkbox and type icon */}
            <div className="flex items-start justify-between">
              <label className="flex items-center gap-default">
                <input
                  type="checkbox"
                  checked={selectedDevices.has(device.hostname)}
                  onChange={() => onSelectDevice(device.hostname)}
                  className="h-4 w-4 rounded border-border-muted bg-bg-elevated text-brand-primary focus:ring-brand-primary cursor-pointer"
                />
                <div className={`pad-xs rounded-lg ${colorClasses.bg}`}>
                  <DeviceIcon className={`${iconSizes.lg} ${colorClasses.text}`} />
                </div>
              </label>
              <Tag colorScheme={typeColor} className="text-xs capitalize">
                {device.type?.replace('_', ' ') || t('list.unknownType')}
              </Tag>
            </div>

            <ActionButton
              action="edit"
              onClick={() => onEdit(device.hostname)}
              className="w-full text-left rounded-lg focus:outline-none focus:ring-2 focus:ring-brand-primary focus:ring-offset-2 focus:ring-offset-surface-base"
              aria-label={t('list.editDeviceAria', { hostname: device.hostname })}
            >
              {/* Device name and IP */}
              <div>
                <h3 className="font-semibold text-text-primary group-hover:text-brand-accent transition-colors truncate">
                  {device.hostname}
                </h3>
                <div className="flex items-center gap-tight mt-tight">
                  <Network className={`${iconSizes.sm} text-text-muted`} />
                  <span className="text-sm text-text-muted font-mono">
                    {device.ip || device.ips?.[0] || t('list.noIp')}
                  </span>
                  {device.ips && device.ips.length > 1 && (
                    <span className="text-xs text-text-muted">+{device.ips.length - 1}</span>
                  )}
                </div>
              </div>

              {/* MAC Address */}
              <div className="text-xs text-text-muted font-mono truncate">{device.mac}</div>

              {/* Protocols */}
              <div className="flex flex-wrap gap-tight">
                {deviceProtocols.length > 0 ? (
                  deviceProtocols.slice(0, 3).map((proto) => (
                    <Tag key={proto} colorScheme="gray" className="text-xs">
                      {proto}
                    </Tag>
                  ))
                ) : (
                  <span className="text-text-muted text-xs">{t('list.noProtocols')}</span>
                )}
                {deviceProtocols.length > 3 && (
                  <Tag colorScheme="gray" className="text-xs">
                    +{deviceProtocols.length - 3}
                  </Tag>
                )}
              </div>
            </ActionButton>

            {/* Actions */}
            <div className="flex justify-end gap-tight pt-2 border-t border-surface-border">
              <ActionButton
                action="edit"
                onClick={() => onEdit(device.hostname)}
                className="pad-xs text-text-muted hover:text-brand-accent hover:bg-surface-hover rounded-lg transition-colors"
                title={t('list.editDeviceTitle', { hostname: device.hostname })}
                aria-label={t('list.editDeviceAria', { hostname: device.hostname })}
              >
                <Edit3 className={iconSizes.md} />
              </ActionButton>
              <ActionButton
                action="edit"
                onClick={() => onClone(device.hostname)}
                className="pad-xs text-text-muted hover:text-status-info hover:bg-surface-hover rounded-lg transition-colors"
                title={t('list.cloneDeviceTitle', { hostname: device.hostname })}
                aria-label={t('list.cloneDeviceAria', { hostname: device.hostname })}
              >
                <Copy className={iconSizes.md} />
              </ActionButton>
              <ActionButton
                action="delete"
                onClick={() => onDelete(device.hostname)}
                className="pad-xs text-text-muted hover:text-status-error hover:bg-surface-hover rounded-lg transition-colors"
                title={t('list.deleteDeviceTitle', { hostname: device.hostname })}
                aria-label={t('list.deleteDeviceAria', { hostname: device.hostname })}
              >
                <Trash2 className={iconSizes.md} />
              </ActionButton>
            </div>
          </CardContent>
        </Card>
      </div>
    );
  };

  return (
    <VirtualList
      items={rows}
      itemHeight={CARD_ROW_HEIGHT_PX}
      getKey={(row) => row[0]?.hostname ?? ''}
      ariaLabel={t('list.listAria')}
      data-testid="device-list-rows"
      renderItem={(row) => (
        <div
          className="grid h-full gap-comfortable pb-4"
          style={{ gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))` }}
        >
          {row.map(renderCard)}
        </div>
      )}
    />
  );
}
