import { memo } from 'react';
import { useTranslation } from 'react-i18next';
import type { DeviceSummary } from '../api/types';
import { DataTable, type DataTableColumn } from '../shared/DataTable';
import { EmptyState } from '../shared/EmptyState';
import { ContainedTable } from '../ui/ContainedTable';
import { useTableLabels } from '../ui/tableLabels';
import { DeviceTypeChip } from './DeviceTypeChip';
import { ProtocolChips } from './ProtocolChips';

/**
 * DeviceTable renders a DeviceSummary[] as the shared DataTable, virtualised
 * above 100 rows (the enterprise packs run 300 to 543 devices). Shared by
 * DevicesPage (the running config's flat device list) and SegmentsPage (the
 * same device shape grouped by VLAN segment, ADR 0008) so both surfaces render
 * devices identically.
 */
export interface DeviceTableProps {
  devices: DeviceSummary[];
  /**
   * Name of the device whose detail pane is open, highlighted in the list.
   * Omitted where the table is a plain read-only list (SegmentsPage).
   */
  selectedName?: string | null;
  /** Supplied to make devices selectable: the name becomes a button. */
  onSelect?: (device: DeviceSummary) => void;
}

/** Rows are one height so the virtual window can place them by arithmetic. */
const ROW_HEIGHT = 44;
const VIRTUAL_HEIGHT = 480;

export const DeviceTable = memo(({ devices, selectedName, onSelect }: DeviceTableProps) => {
  const { t } = useTranslation('pages');
  const { t: tCommon } = useTranslation('common');
  const labels = useTableLabels<DeviceSummary>(tCommon('labels.devices').toLowerCase());

  const columns: DataTableColumn<DeviceSummary>[] = [
    {
      key: 'name',
      header: t('devices.deviceHeader'),
      sortValue: (device) => device.name,
      cell: (device) =>
        onSelect ? (
          // A real button rather than a click handler on the row: the list is
          // the navigation half of a list + detail, so the target has to be
          // reachable by keyboard and announce itself as actionable.
          <button
            type="button"
            onClick={() => onSelect(device)}
            aria-pressed={device.name === selectedName}
            data-testid={`device-select-${device.name}`}
            className="max-w-64 truncate rounded font-semibold text-text-primary hover:text-brand-primary-strong focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-primary"
          >
            {device.name}
          </button>
        ) : (
          <span className="font-semibold text-text-primary">{device.name}</span>
        ),
    },
    {
      key: 'type',
      header: tCommon('labels.type'),
      sortValue: (device) => device.type,
      cell: (device) => <DeviceTypeChip type={device.type} />,
    },
    {
      key: 'ips',
      header: t('devices.ipAddressesHeader'),
      cell: (device) => (
        <span className="whitespace-nowrap font-mono text-xs">{device.ips.join(', ') || '—'}</span>
      ),
    },
    {
      key: 'protocols',
      header: t('devices.protocolsHeader'),
      cell: (device) => <ProtocolChips protocols={device.protocols} />,
    },
  ];

  return (
    <ContainedTable label={t('devices.tableCaption')}>
      <DataTable
        rows={devices}
        columns={columns}
        getRowKey={(device) => device.name}
        caption={t('devices.tableCaption')}
        labels={labels}
        {...(selectedName ? { selectedRowKey: selectedName } : {})}
        virtualize={{ rowHeight: ROW_HEIGHT }}
        maxHeight={VIRTUAL_HEIGHT}
        data-testid="device-table"
        empty={
          <div data-testid="device-table-empty">
            <EmptyState title={tCommon('emptyState.noDevicesConfigured')} />
          </div>
        }
      />
    </ContainedTable>
  );
});

DeviceTable.displayName = 'DeviceTable';
