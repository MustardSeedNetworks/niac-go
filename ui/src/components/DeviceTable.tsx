import { memo } from 'react';
import { useTranslation } from 'react-i18next';
import type { DeviceSummary } from '../api/types';
import { Button } from '../ui/Button';
import { Tag } from '../ui/Tag';
import { SmallText } from '../ui/Typography';
import { DataTable, type DataTableColumn } from './DataTable';

/**
 * DeviceTable renders a DeviceSummary[] as a table, switching to virtual
 * scrolling above 100 rows. Shared by DevicesPage (the running config's
 * flat device list) and SegmentsPage (the same device shape grouped by
 * VLAN segment, ADR 0008) so both surfaces render devices identically.
 */
export interface DeviceTableProps {
  devices: DeviceSummary[];
  /**
   * Name of the device whose detail pane is open, highlighted in the list.
   * Omitted where the table is a plain read-only list (SegmentsPage).
   */
  selectedName?: string | null;
  /** Supplied to make rows selectable; adds the per-row select control. */
  onSelect?: (device: DeviceSummary) => void;
}

export const DeviceTable = memo(({ devices, selectedName, onSelect }: DeviceTableProps) => {
  const { t } = useTranslation('pages');

  const columns: DataTableColumn<DeviceSummary>[] = [
    {
      key: 'name',
      header: 'Device',
      cell: (device) => <span className="font-semibold text-text-primary">{device.name}</span>,
    },
    {
      key: 'type',
      header: 'Type',
      cell: (device) => device.type,
    },
    {
      key: 'ips',
      header: t('devices.ipAddressesHeader'),
      cell: (device) => <span className="font-mono text-xs">{device.ips.join(', ') || '—'}</span>,
    },
    {
      key: 'protocols',
      header: 'Protocols',
      cell: (device) => (
        <div className="flex flex-wrap gap-compact">
          {device.protocols.map((proto) => (
            <Tag key={`${device.name}-${proto}`} colorScheme="purple">
              {proto}
            </Tag>
          ))}
          {device.protocols.length === 0 && <SmallText className="text-text-muted">None</SmallText>}
        </div>
      ),
    },
  ];

  if (onSelect) {
    // A real button rather than a click handler on the row: the list is the
    // navigation half of a list + detail, so the target has to be reachable
    // by keyboard and announce itself as actionable.
    columns.push({
      key: 'select',
      header: '',
      cell: (device) => (
        <Button
          size="sm"
          variant={device.name === selectedName ? 'solid' : 'outline'}
          onClick={() => onSelect(device)}
          aria-pressed={device.name === selectedName}
          data-testid={`device-select-${device.name}`}
        >
          {device.name === selectedName ? t('devices.selectedLabel') : t('devices.selectButton')}
        </Button>
      ),
    });
  }

  return (
    <DataTable
      rows={devices}
      columns={columns}
      getRowKey={(device) => device.name}
      rowClassName={(device) => (device.name === selectedName ? 'bg-brand-accent/10' : '')}
      emptyMessage={
        <div className="rounded-xl border border-surface-border bg-bg-base/50 pad-xl text-center text-text-muted">
          No devices defined in the loaded configuration.
        </div>
      }
      virtualization={{
        itemHeight: 60,
        containerHeight: 600,
        overscan: 5,
        threshold: 100,
        renderStatus: (visibleCount, totalCount) =>
          `Showing ${visibleCount} of ${totalCount} devices (virtual scrolling enabled)`,
      }}
    />
  );
});

DeviceTable.displayName = 'DeviceTable';
