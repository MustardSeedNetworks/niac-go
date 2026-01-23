import { type FC, memo, useMemo } from 'react';
import { useTimeDisplay } from '../hooks/useTimeDisplay';
import { Tag } from '../ui/Tag';
import { SmallText } from '../ui/Typography';
import { getProtocolColor } from '../utils/protocol-colors';
import { formatTimeByMode, getTimeDisplayLabel } from '../utils/time-display';

/**
 * Packet data structure for display in the list
 */
export interface Packet {
  id: string;
  timestamp: string;
  protocol: string;
  sourceIp: string;
  destIp: string;
  sourcePort?: number;
  destPort?: number;
  size: number;
  summary: string;
  rawData: string; // Hex encoded raw packet data
  headers?: Record<string, unknown>;
}

interface PacketListProps {
  packets: Packet[];
  selectedPacketId: string | null;
  onSelectPacket: (packet: Packet) => void;
  protocolFilter: string;
  searchQuery: string;
  autoScroll: boolean;
  getRowStyle?: (packet: Packet) => React.CSSProperties | undefined;
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
    packet: Packet;
    isSelected: boolean;
    onClick: () => void;
    formattedTime: string;
    rowStyle?: React.CSSProperties;
  }) => (
    <button
      type="button"
      onClick={onClick}
      className={`w-full text-left p-2 rounded-lg border transition-colors ${
        isSelected
          ? 'border-violet-500/50 bg-violet-900/30'
          : 'border-white/5 bg-gray-950/50 hover:bg-gray-900/50 hover:border-white/10'
      }`}
      style={rowStyle}
    >
      <div className="flex items-center gap-2">
        <Tag colorScheme={getProtocolColor(packet.protocol)} className="text-xs">
          {packet.protocol}
        </Tag>
        <SmallText className="text-gray-400 font-mono text-xs">{formattedTime}</SmallText>
      </div>
      <div className="mt-1 text-sm text-white truncate">
        {packet.sourceIp}
        {packet.sourcePort && `:${packet.sourcePort}`}
        <span className="text-gray-500 mx-1">-&gt;</span>
        {packet.destIp}
        {packet.destPort && `:${packet.destPort}`}
      </div>
      <SmallText className="text-gray-400 truncate block">{packet.summary}</SmallText>
    </button>
  ),
);

PacketRow.displayName = 'PacketRow';

/**
 * Packet List Component
 *
 * Displays a scrollable list of captured packets with filtering support.
 * Clicking a packet selects it for detailed viewing.
 */
export const PacketList: FC<PacketListProps> = memo(
  ({ packets, selectedPacketId, onSelectPacket, protocolFilter, searchQuery, autoScroll, getRowStyle }) => {
    const { mode: timeMode, cycleMode: cycleTimeMode } = useTimeDisplay();

    // Filter packets based on protocol and search query
    const filteredPackets = useMemo(() => {
      return packets.filter((packet) => {
        // Protocol filter
        if (
          protocolFilter !== 'All' &&
          packet.protocol.toUpperCase() !== protocolFilter.toUpperCase()
        ) {
          return false;
        }

        // Search filter
        if (searchQuery) {
          const query = searchQuery.toLowerCase();
          const searchableText = [
            packet.protocol,
            packet.sourceIp,
            packet.destIp,
            packet.summary,
            packet.sourcePort?.toString(),
            packet.destPort?.toString(),
          ]
            .filter(Boolean)
            .join(' ')
            .toLowerCase();

          return searchableText.includes(query);
        }

        return true;
      });
    }, [packets, protocolFilter, searchQuery]);

    if (packets.length === 0) {
      return (
        <div className="h-full flex items-center justify-center text-gray-400">
          <div className="text-center">
            <p className="text-sm">No packets captured</p>
            <SmallText>Waiting for packet stream...</SmallText>
          </div>
        </div>
      );
    }

    if (filteredPackets.length === 0) {
      return (
        <div className="h-full flex items-center justify-center text-gray-400">
          <div className="text-center">
            <p className="text-sm">No packets match filter</p>
            <SmallText>Try adjusting the filter criteria</SmallText>
          </div>
        </div>
      );
    }

    return (
      <div className="h-full flex flex-col">
        <button
          type="button"
          onClick={cycleTimeMode}
          className="text-xs text-gray-500 hover:text-violet-300 mb-1 text-left select-none"
          title="Click to cycle: Absolute / Relative / Delta"
        >
          Mode: {getTimeDisplayLabel(timeMode)}
        </button>
        <div
          className={`flex-1 overflow-y-auto space-y-1 pr-2 ${autoScroll ? 'scroll-smooth' : ''}`}
          style={{ scrollBehavior: autoScroll ? 'smooth' : 'auto' }}
        >
          {filteredPackets.map((packet, idx) => (
            <PacketRow
              key={packet.id}
              packet={packet}
              isSelected={selectedPacketId === packet.id}
              onClick={() => onSelectPacket(packet)}
              formattedTime={formatTimeByMode(
                packet.timestamp,
                timeMode,
                filteredPackets[0]?.timestamp ?? null,
                idx > 0 ? filteredPackets[idx - 1].timestamp : null,
              )}
              rowStyle={getRowStyle?.(packet)}
            />
          ))}
        </div>
      </div>
    );
  },
);

PacketList.displayName = 'PacketList';
