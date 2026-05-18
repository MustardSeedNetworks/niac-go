import { type FC, memo, useCallback } from 'react';
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
      <td className="px-3 py-2 text-text-muted text-xs font-mono">{packet.number}</td>
      <td className="px-3 py-2 text-text-secondary text-xs font-mono">{formattedTime}</td>
      <td className="px-3 py-2 text-text-primary text-sm font-mono">{packet.sourceIp}</td>
      <td className="px-3 py-2 text-text-primary text-sm font-mono">{packet.destIp}</td>
      <td className="px-3 py-2">
        <Tag colorScheme={getProtocolColor(packet.protocol)} className="text-xs">
          {packet.protocol}
        </Tag>
      </td>
      <td className="px-3 py-2 text-text-muted text-xs text-right">{packet.length}</td>
      <td className="px-3 py-2 text-text-muted text-xs truncate max-w-xs">{packet.info}</td>
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
          <CardContent className="h-full flex items-center justify-center text-text-muted">
            <div className="text-center">
              <p className="text-sm">No packets to display</p>
              <SmallText>Upload a PCAP file to analyze</SmallText>
            </div>
          </CardContent>
        </Card>
      );
    }

    return (
      <Card className="border-surface-border bg-bg-surface/70 h-full flex flex-col">
        <CardContent className="h-full flex flex-col">
          {/* Packet count */}
          <div className="mb-3 flex items-center justify-between">
            <SmallText className="text-text-muted">
              Showing {packets.length} of {totalPackets} packets
            </SmallText>
          </div>

          {/* Packet Table */}
          <div className="flex-1 min-h-0 overflow-auto rounded-lg border border-surface-border">
            {packets.length === 0 ? (
              <div className="h-full flex items-center justify-center text-text-muted">
                <div className="text-center">
                  <p className="text-sm">No packets match filter</p>
                  <SmallText>Try adjusting the filter expression</SmallText>
                </div>
              </div>
            ) : (
              <table className="min-w-full divide-y divide-white/5">
                <thead className="bg-bg-surface/80 sticky top-0 z-10">
                  <tr>
                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-text-muted">
                      #
                    </th>
                    <th
                      className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-text-muted cursor-pointer hover:text-brand-accent select-none"
                      onClick={cycleTimeMode}
                      title="Click to cycle: Absolute / Relative / Delta"
                    >
                      {getTimeDisplayLabel(timeMode)}
                    </th>
                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-text-muted">
                      Source
                    </th>
                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-text-muted">
                      Destination
                    </th>
                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-text-muted">
                      Protocol
                    </th>
                    <th className="px-3 py-2 text-right text-xs font-semibold uppercase tracking-wide text-text-muted">
                      Length
                    </th>
                    <th className="px-3 py-2 text-left text-xs font-semibold uppercase tracking-wide text-text-muted">
                      Info
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-white/5">
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
                        idx > 0 ? packets[idx - 1].timestamp : null,
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
