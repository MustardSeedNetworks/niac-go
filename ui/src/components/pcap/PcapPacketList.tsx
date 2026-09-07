import { type FC, memo } from 'react';
import { useTranslation } from 'react-i18next';
import type { PcapPacket } from '../../api/types';
import { useTimeDisplay } from '../../hooks/useTimeDisplay';
import { Card, CardContent } from '../../ui/Card';
import { DataTable, type DataTableColumn } from '../../ui/DataTable';
import { Tag } from '../../ui/Tag';
import { SmallText } from '../../ui/Typography';
import { getProtocolColor } from '../../utils/protocol-colors';
import { formatTimeByMode, getTimeDisplayLabel } from '../../utils/time-display';

interface PcapPacketListProps {
  packets: PcapPacket[];
  totalPackets: number;
  selectedPacketId: string | null;
  onSelectPacket: (packet: PcapPacket) => void;
  getRowStyle?: (packet: PcapPacket) => React.CSSProperties | undefined;
}

/**
 * PCAP Packet List Component
 *
 * Displays a table of pre-filtered packets with time mode cycling.
 * Filtering is handled externally via the FilterBar + useDisplayFilter hook.
 */
export const PcapPacketList: FC<PcapPacketListProps> = memo(
  ({ packets, totalPackets, selectedPacketId, onSelectPacket, getRowStyle }) => {
    const { t } = useTranslation('pages');
    const { mode: timeMode, cycleMode: cycleTimeMode } = useTimeDisplay();

    // Relative and delta time modes are computed against the first and the
    // preceding row, so the column needs the row's position, not just the row.
    const positionById = new Map(packets.map((packet, idx) => [packet.id, idx]));

    const columns: DataTableColumn<PcapPacket>[] = [
      {
        key: 'number',
        header: '#',
        cellClassName: 'text-text-muted text-xs font-mono',
        cell: (packet) => packet.number,
      },
      {
        key: 'time',
        header: (
          <button
            type="button"
            onClick={cycleTimeMode}
            title={t('packets.list.cycleTimeModeTitle')}
            className="uppercase tracking-wide hover:text-brand-accent"
          >
            {getTimeDisplayLabel(timeMode)}
          </button>
        ),
        cellClassName: 'text-text-secondary text-xs font-mono',
        cell: (packet) => {
          const idx = positionById.get(packet.id) ?? 0;
          return formatTimeByMode(
            packet.timestamp,
            timeMode,
            packets[0]?.timestamp ?? null,
            packets[idx - 1]?.timestamp ?? null,
          );
        },
      },
      {
        key: 'source',
        header: t('packets.list.headerSource'),
        cellClassName: 'text-text-primary text-sm font-mono',
        cell: (packet) => packet.sourceIp,
      },
      {
        key: 'destination',
        header: t('packets.list.headerDestination'),
        cellClassName: 'text-text-primary text-sm font-mono',
        cell: (packet) => packet.destIp,
      },
      {
        key: 'protocol',
        header: t('packets.list.headerProtocol'),
        cell: (packet) => (
          <Tag colorScheme={getProtocolColor(packet.protocol)} className="text-xs">
            {packet.protocol}
          </Tag>
        ),
      },
      {
        key: 'length',
        header: t('packets.list.headerLength'),
        align: 'right',
        cellClassName: 'text-text-muted text-xs',
        cell: (packet) => packet.length,
      },
      {
        key: 'info',
        header: t('packets.list.headerInfo'),
        cellClassName: 'text-text-muted text-xs truncate max-w-xs',
        cell: (packet) => packet.info,
      },
    ];

    if (totalPackets === 0) {
      return (
        <Card className="border-surface-border bg-bg-surface/70 h-full">
          <CardContent className="h-full flex-center text-text-muted">
            <div className="text-center">
              <p className="text-sm">{t('packets.list.noPacketsTitle')}</p>
              <SmallText>{t('packets.list.noPacketsDescription')}</SmallText>
            </div>
          </CardContent>
        </Card>
      );
    }

    return (
      <Card className="border-surface-border bg-bg-surface/70 h-full flex flex-col">
        <CardContent className="h-full flex flex-col">
          <div className="mb-heading flex-between">
            <SmallText className="text-text-muted">
              {t('packets.list.showingCount', { shown: packets.length, total: totalPackets })}
            </SmallText>
          </div>

          <DataTable
            rows={packets}
            columns={columns}
            getRowKey={(packet) => packet.id}
            onRowClick={onSelectPacket}
            rowStyle={getRowStyle}
            rowClassName={(packet) =>
              `transition-colors ${
                selectedPacketId === packet.id ? 'bg-brand-primary/30' : 'hover:bg-bg-surface/50'
              }`
            }
            stickyHeader
            containerClassName="flex-1 min-h-0 overflow-y-auto"
            emptyMessage={
              <div className="flex-1 min-h-0 flex-center rounded-xl border border-surface-border text-text-muted">
                <div className="text-center">
                  <p className="text-sm">{t('packets.list.noMatchTitle')}</p>
                  <SmallText>{t('packets.list.noMatchDescription')}</SmallText>
                </div>
              </div>
            }
          />
        </CardContent>
      </Card>
    );
  },
);

PcapPacketList.displayName = 'PcapPacketList';

export default PcapPacketList;
