import { memo } from 'react';
import { useTranslation } from 'react-i18next';
import type { DeviceSummary } from '../api/types';
import { Tag } from '../ui/Tag';
import { SmallText } from '../ui/Typography';
import { DataTable, type DataTableColumn } from './DataTable';

/**
 * DeviceTable renders a DeviceSummary[] as a table, switching to virtual
 * scrolling above 100 rows. Shared by DevicesPage (the running config's
 * flat device list) and SegmentsPage (the same device shape grouped by
 * VLAN segment, ADR 0008) so both surfaces render devices identically.
 */
export const DeviceTable = memo(({ devices }: { devices: DeviceSummary[] }) => {
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

  return (
    <DataTable
      rows={devices}
      columns={columns}
      getRowKey={(device) => device.name}
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
