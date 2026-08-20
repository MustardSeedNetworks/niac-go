import { type FC, memo, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import type { PcapPacket } from '../../api/types';
import { useTimeDisplay } from '../../hooks/useTimeDisplay';
import { Card, CardContent } from '../../ui/Card';
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
 * Individual packet row component - memoized for performance
 */
const PacketRow = memo(
  ({
    packet,
    isSelected,
    onClick,
    formattedTime,
    rowStyle,
  }: {
    packet: PcapPacket;
    isSelected: boolean;
    onClick: () => void;
    formattedTime: string;
    rowStyle?: React.CSSProperties;
  }) => (
    <tr
      onClick={onClick}
      className={`cursor-pointer transition-colors ${
        isSelected ? 'bg-brand-primary/30' : 'hover:bg-bg-surface/50'
      }`}
      style={rowStyle}
    >
      <td className="px-3 py-row text-text-muted text-xs font-mono">{packet.number}</td>
      <td className="px-3 py-row text-text-secondary text-xs font-mono">{formattedTime}</td>
      <td className="px-3 py-row text-text-primary text-sm font-mono">{packet.sourceIp}</td>
      <td className="px-3 py-row text-text-primary text-sm font-mono">{packet.destIp}</td>
      <td className="px-3 py-row">
        <Tag colorScheme={getProtocolColor(packet.protocol)} className="text-xs">
          {packet.protocol}
        </Tag>
      </td>
      <td className="px-3 py-row text-text-muted text-xs text-right">{packet.length}</td>
      <td className="px-3 py-row text-text-muted text-xs truncate max-w-xs">{packet.info}</td>
    </tr>
  ),
);

PacketRow.displayName = 'PacketRow';

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

    const handleSelectPacket = useCallback(
      (packet: PcapPacket) => {
        onSelectPacket(packet);
      },
      [onSelectPacket],
    );

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
          {/* Packet count */}
          <div className="mb-heading flex-between">
            <SmallText className="text-text-muted">
              {t('packets.list.showingCount', { shown: packets.length, total: totalPackets })}
            </SmallText>
          </div>

          {/* Packet Table */}
          <div className="flex-1 min-h-0 overflow-auto rounded-lg border border-surface-border">
            {packets.length === 0 ? (
              <div className="h-full flex-center text-text-muted">
                <div className="text-center">
                  <p className="text-sm">{t('packets.list.noMatchTitle')}</p>
                  <SmallText>{t('packets.list.noMatchDescription')}</SmallText>
                </div>
              </div>
            ) : (
              <table className="min-w-full divide-y divide-knob/5">
                <thead className="bg-bg-surface/80 sticky top-0 z-10">
                  <tr>
                    <th className="px-3 py-row text-left text-xs font-semibold uppercase tracking-wide text-text-muted">
                      #
                    </th>
                    <th
                      className="px-3 py-row text-left text-xs font-semibold uppercase tracking-wide text-text-muted cursor-pointer hover:text-brand-accent select-none"
                      onClick={cycleTimeMode}
                      title={t('packets.list.cycleTimeModeTitle')}
                    >
                      {getTimeDisplayLabel(timeMode)}
                    </th>
                    <th className="px-3 py-row text-left text-xs font-semibold uppercase tracking-wide text-text-muted">
                      {t('packets.list.headerSource')}
                    </th>
                    <th className="px-3 py-row text-left text-xs font-semibold uppercase tracking-wide text-text-muted">
                      {t('packets.list.headerDestination')}
                    </th>
                    <th className="px-3 py-row text-left text-xs font-semibold uppercase tracking-wide text-text-muted">
                      {t('packets.list.headerProtocol')}
                    </th>
                    <th className="px-3 py-row text-right text-xs font-semibold uppercase tracking-wide text-text-muted">
                      {t('packets.list.headerLength')}
                    </th>
                    <th className="px-3 py-row text-left text-xs font-semibold uppercase tracking-wide text-text-muted">
                      {t('packets.list.headerInfo')}
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-knob/5">
                  {packets.map((packet, idx) => (
                    <PacketRow
                      key={packet.id}
                      packet={packet}
                      isSelected={selectedPacketId === packet.id}
                      onClick={() => handleSelectPacket(packet)}
                      formattedTime={formatTimeByMode(
                        packet.timestamp,
                        timeMode,
                        packets[0]?.timestamp ?? null,
                        packets[idx - 1]?.timestamp ?? null,
                      )}
                      rowStyle={getRowStyle?.(packet)}
                    />
                  ))}
                </tbody>
              </table>
            )}
          </div>
        </CardContent>
      </Card>
    );
  },
);

PcapPacketList.displayName = 'PcapPacketList';

export default PcapPacketList;
