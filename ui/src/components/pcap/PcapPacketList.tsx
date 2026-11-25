import { memo } from 'react';
import { useTranslation } from 'react-i18next';
import type { PcapPacket } from '../../api/types';
import { useTimeDisplay } from '../../hooks/useTimeDisplay';
import { Card } from '../../shared/Card';
import { CategoryChip } from '../../shared/CategoryChip';
import { DataTable, type DataTableColumn } from '../../shared/DataTable';
import { EmptyState } from '../../shared/EmptyState';
import { ActionButton } from '../../ui/ActionButton';
import { SmallText } from '../../ui/Typography';
import { useTableLabels } from '../../ui/tableLabels';
import { formatTimeByMode, getTimeDisplayLabel } from '../../utils/time-display';

interface PcapPacketListProps {
  packets: PcapPacket[];
  totalPackets: number;
  selectedPacketId: string | null;
  onSelectPacket: (packet: PcapPacket) => void;
  /** The coloring rule matching a packet. Shown as a swatch on its number. */
  getRowStyle?: (packet: PcapPacket) => React.CSSProperties | undefined;
}

/**
 * PCAP Packet List Component
 *
 * Displays a table of pre-filtered packets, windowed once there are hundreds.
 * Filtering is handled externally via the FilterBar + useDisplayFilter hook.
 */
export const PcapPacketList = memo(function PcapPacketList({
  packets,
  totalPackets,
  selectedPacketId,
  onSelectPacket,
  getRowStyle,
}: PcapPacketListProps) {
  const { t } = useTranslation('pages');
  const { mode: timeMode, cycleMode: cycleTimeMode } = useTimeDisplay();

  // Relative and delta time modes are computed against the first and the
  // preceding row, so the column needs the row's position, not just the row.
  const positionById = new Map(packets.map((packet, idx) => [packet.id, idx]));

  const labels = useTableLabels<PcapPacket>(t('packets.list.headerPacket').toLowerCase());

  const columns: DataTableColumn<PcapPacket>[] = [
    {
      key: 'number',
      header: '#',
      cell: (packet) => (
        // The coloring rule has no row to paint (rows are one height so the
        // list can be windowed), so it colours the number.
        <span
          data-testid={`pcap-packet-${packet.number}`}
          className="inline-block rounded px-1.5 font-mono text-xs text-text-muted"
          style={getRowStyle?.(packet)}
        >
          {packet.number}
        </span>
      ),
    },
    {
      key: 'time',
      header: getTimeDisplayLabel(timeMode),
      cell: (packet) => {
        const idx = positionById.get(packet.id) ?? 0;
        return (
          <span className="whitespace-nowrap font-mono text-xs">
            {formatTimeByMode(
              packet.timestamp,
              timeMode,
              packets[0]?.timestamp ?? null,
              packets[idx - 1]?.timestamp ?? null,
            )}
          </span>
        );
      },
    },
    {
      key: 'source',
      header: t('packets.list.headerSource'),
      cell: (packet) => <span className="font-mono text-text-primary">{packet.sourceIp}</span>,
    },
    {
      key: 'destination',
      header: t('packets.list.headerDestination'),
      cell: (packet) => <span className="font-mono text-text-primary">{packet.destIp}</span>,
    },
    {
      key: 'protocol',
      header: t('packets.list.headerProtocol'),
      cell: (packet) => <CategoryChip label={packet.protocol} />,
    },
    {
      key: 'length',
      header: t('packets.list.headerLength'),
      align: 'right',
      cell: (packet) => <span className="font-mono text-xs">{packet.length}</span>,
    },
    {
      key: 'info',
      header: t('packets.list.headerInfo'),
      cell: (packet) => (
        <span className="block max-w-xs truncate text-xs text-text-muted" title={packet.info}>
          {packet.info}
        </span>
      ),
    },
  ];

  if (totalPackets === 0) {
    return (
      <Card className="h-full">
        <EmptyState
          title={t('packets.list.noPacketsTitle')}
          hint={t('packets.list.noPacketsDescription')}
        />
      </Card>
    );
  }

  return (
    <Card className="flex h-full flex-col">
      <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
        <SmallText className="text-text-muted">
          {t('packets.list.showingCount', { shown: packets.length, total: totalPackets })}
        </SmallText>
        <ActionButton
          variant="ghost"
          size="sm"
          onClick={cycleTimeMode}
          title={t('packets.list.cycleTimeModeTitle')}
        >
          {t('packets.list.timeColumn', { mode: getTimeDisplayLabel(timeMode) })}
        </ActionButton>
      </div>

      <DataTable
        rows={packets}
        columns={columns}
        getRowKey={(packet) => packet.id}
        caption={t('packets.list.headerPacket')}
        labels={labels}
        onRowClick={onSelectPacket}
        {...(selectedPacketId ? { selectedRowKey: selectedPacketId } : {})}
        virtualize={{ rowHeight: 44 }}
        maxHeight={560}
        empty={
          <EmptyState
            title={t('packets.list.noMatchTitle')}
            hint={t('packets.list.noMatchDescription')}
          />
        }
      />
    </Card>
  );
});

export default PcapPacketList;
