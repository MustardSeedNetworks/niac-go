import { Copy, Pencil, Trash2 } from 'lucide-react';
import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { Device } from '../../api/types';
import { useDeviceList } from '../../contexts/DeviceListContext';
import { Card } from '../../shared/Card';
import { ActionButton } from '../../ui/ActionButton';
import { DeviceTypeChip } from '../DeviceTypeChip';
import { ProtocolChips } from '../ProtocolChips';
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
    const deviceProtocols = getDeviceProtocols(device);
    const selected = selectedDevices.has(device.hostname);

    return (
      <div key={device.hostname} data-testid="device-list-card" className="h-full">
        <Card className={`h-full ${selected ? 'border-brand-primary' : ''}`}>
          <div className="stack">
            <div className="flex items-center justify-between gap-2">
              <input
                type="checkbox"
                checked={selected}
                onChange={() => onSelectDevice(device.hostname)}
                className="h-4 w-4 accent-brand-primary"
                aria-label={t('list.selectDeviceAria', { hostname: device.hostname })}
              />
              <DeviceTypeChip type={device.type} />
            </div>

            <div className="min-w-0">
              <ActionButton
                action="edit"
                variant="ghost"
                onClick={() => onEdit(device.hostname)}
                className="w-full justify-start px-1 font-semibold"
                aria-label={t('list.editDeviceAria', { hostname: device.hostname })}
              >
                <span className="truncate">{device.hostname}</span>
              </ActionButton>
              <p className="mt-1 truncate font-mono text-sm text-text-muted">
                {device.ip || device.ips?.[0] || t('list.noIp')}
                {device.ips && device.ips.length > 1 ? ` +${device.ips.length - 1}` : ''}
              </p>
              <p className="truncate font-mono text-xs text-text-muted">{device.mac}</p>
            </div>

            <ProtocolChips protocols={deviceProtocols} />

            <div className="flex justify-end gap-1 border-t border-hairline pt-2">
              <ActionButton
                action="edit"
                variant="ghost"
                size="sm"
                iconOnly
                icon={<Pencil className="h-4 w-4" />}
                onClick={() => onEdit(device.hostname)}
                title={t('list.editDeviceTitle', { hostname: device.hostname })}
                aria-label={t('list.editDeviceAria', { hostname: device.hostname })}
              />
              <ActionButton
                action="edit"
                variant="ghost"
                size="sm"
                iconOnly
                icon={<Copy className="h-4 w-4" />}
                onClick={() => onClone(device.hostname)}
                title={t('list.cloneDeviceTitle', { hostname: device.hostname })}
                aria-label={t('list.cloneDeviceAria', { hostname: device.hostname })}
              />
              <ActionButton
                action="delete"
                variant="destructive"
                size="sm"
                iconOnly
                icon={<Trash2 className="h-4 w-4" />}
                onClick={() => onDelete(device.hostname)}
                title={t('list.deleteDeviceTitle', { hostname: device.hostname })}
                aria-label={t('list.deleteDeviceAria', { hostname: device.hostname })}
              />
            </div>
          </div>
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
