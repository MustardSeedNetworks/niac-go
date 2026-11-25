import { Copy, Pencil, Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import type { Device } from '../../api/types';
import { useDeviceList } from '../../contexts/DeviceListContext';
import { DataTable, type DataTableColumn } from '../../shared/DataTable';
import { ActionButton } from '../../ui/ActionButton';
import { ContainedTable } from '../../ui/ContainedTable';
import { useTableLabels } from '../../ui/tableLabels';
import { DeviceTypeChip } from '../DeviceTypeChip';
import { ProtocolChips } from '../ProtocolChips';

/**
 * Rows per page. Selection (for the bulk delete) is page-wide in the shared
 * DataTable, and a library of 543 devices is searched or filtered long before
 * it is paged, so the table pages rather than windows.
 */
const PAGE_SIZE = 50;

export function DeviceTableView() {
  const { t } = useTranslation('devices');
  const { t: tCommon } = useTranslation('common');
  const {
    devices,
    selectedDevices,
    onSelectionChange,
    onEdit,
    onClone,
    onDelete,
    getDeviceProtocols,
  } = useDeviceList();
  const labels = useTableLabels<Device>(tCommon('labels.devices').toLowerCase(), (device) =>
    t('list.selectDeviceAria', { hostname: device.hostname }),
  );

  const columns: DataTableColumn<Device>[] = [
    {
      key: 'hostname',
      header: t('list.tableHeaderHostname'),
      sortValue: (device) => device.hostname,
      cell: (device) => (
        <ActionButton
          action="edit"
          variant="ghost"
          size="sm"
          onClick={() => onEdit(device.hostname)}
          title={t('list.openEditorTitle', { hostname: device.hostname })}
          className="max-w-64 justify-start truncate px-1 font-semibold"
        >
          {device.hostname}
        </ActionButton>
      ),
    },
    {
      key: 'type',
      header: t('list.tableHeaderType'),
      sortValue: (device) => device.type ?? '',
      cell: (device) => <DeviceTypeChip type={device.type} />,
    },
    {
      key: 'ip',
      header: t('list.tableHeaderIpAddress'),
      cell: (device) => (
        <span className="whitespace-nowrap font-mono text-xs">
          {device.ip || device.ips?.[0] || '—'}
          {device.ips && device.ips.length > 1 ? (
            <span className="ml-1 text-text-muted">+{device.ips.length - 1}</span>
          ) : null}
        </span>
      ),
    },
    {
      key: 'protocols',
      header: t('list.tableHeaderProtocols'),
      cell: (device) => <ProtocolChips protocols={getDeviceProtocols(device)} />,
    },
  ];

  return (
    <ContainedTable>
      <DataTable
        rows={devices}
        columns={columns}
        getRowKey={(device) => device.hostname}
        caption={t('list.listAria')}
        labels={labels}
        pageSize={PAGE_SIZE}
        selection={{ selectedKeys: selectedDevices, onChange: onSelectionChange }}
        data-testid="device-list-rows"
        empty={null}
        rowActions={(device) => (
          <>
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
          </>
        )}
      />
    </ContainedTable>
  );
}
